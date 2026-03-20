package notify

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	sestypes "github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	snstypes "github.com/aws/aws-sdk-go-v2/service/sns/types"

	"github.com/colinbruner/cronhealth/internal/db"
)

// Notifier defines the interface for sending alert and recovery notifications.
type Notifier interface {
	SendAlert(ctx context.Context, check db.Check, channels []db.NotificationChannel) error
	SendRecovery(ctx context.Context, check db.Check, channels []db.NotificationChannel) error
}

// Config holds settings for the AWS-backed notifier.
type Config struct {
	AWSRegion  string
	SESFrom    string
	SNSEnabled bool
}

// AWSNotifier sends notifications via AWS SES (email) and SNS (SMS).
type AWSNotifier struct {
	sesClient *ses.Client
	snsClient *sns.Client
	cfg       Config
}

// NewAWSNotifier creates an AWSNotifier with configured SES and SNS clients.
func NewAWSNotifier(ctx context.Context, cfg Config) (*AWSNotifier, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.AWSRegion),
	)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	return &AWSNotifier{
		sesClient: ses.NewFromConfig(awsCfg),
		snsClient: sns.NewFromConfig(awsCfg),
		cfg:       cfg,
	}, nil
}

// SendAlert sends an alert notification to all provided channels.
// Failures are logged but not returned; notifications are non-blocking.
func (n *AWSNotifier) SendAlert(ctx context.Context, check db.Check, channels []db.NotificationChannel) error {
	subject := fmt.Sprintf("[cronhealth] ALERT: %s missed its ping", check.Name)
	body := formatAlertBody(check)
	smsMsg := formatAlertSMS(check)

	for _, ch := range channels {
		if !ch.Enabled {
			continue
		}
		switch ch.Type {
		case "email":
			n.sendEmail(ctx, ch.Target, subject, body)
		case "sms":
			if n.cfg.SNSEnabled {
				n.sendSMS(ctx, ch.Target, smsMsg)
			}
		default:
			slog.Warn("unknown notification channel type", "type", ch.Type, "channel_id", ch.ID)
		}
	}

	return nil
}

// SendRecovery sends a recovery notification to all provided channels.
// Failures are logged but not returned; notifications are non-blocking.
func (n *AWSNotifier) SendRecovery(ctx context.Context, check db.Check, channels []db.NotificationChannel) error {
	subject := fmt.Sprintf("[cronhealth] RESOLVED: %s is back", check.Name)
	body := formatRecoveryBody(check)
	smsMsg := formatRecoverySMS(check)

	for _, ch := range channels {
		if !ch.Enabled {
			continue
		}
		switch ch.Type {
		case "email":
			n.sendEmail(ctx, ch.Target, subject, body)
		case "sms":
			if n.cfg.SNSEnabled {
				n.sendSMS(ctx, ch.Target, smsMsg)
			}
		default:
			slog.Warn("unknown notification channel type", "type", ch.Type, "channel_id", ch.ID)
		}
	}

	return nil
}

func (n *AWSNotifier) sendEmail(ctx context.Context, to, subject, body string) {
	input := &ses.SendEmailInput{
		Source: aws.String(n.cfg.SESFrom),
		Destination: &sestypes.Destination{
			ToAddresses: []string{to},
		},
		Message: &sestypes.Message{
			Subject: &sestypes.Content{
				Data:    aws.String(subject),
				Charset: aws.String("UTF-8"),
			},
			Body: &sestypes.Body{
				Text: &sestypes.Content{
					Data:    aws.String(body),
					Charset: aws.String("UTF-8"),
				},
			},
		},
	}

	_, err := n.sesClient.SendEmail(ctx, input)
	if err != nil {
		slog.Error("SES send failed",
			"to", to,
			"subject", subject,
			"error", err,
		)
	}
}

func (n *AWSNotifier) sendSMS(ctx context.Context, phoneNumber, message string) {
	input := &sns.PublishInput{
		PhoneNumber: aws.String(phoneNumber),
		Message:     aws.String(message),
		MessageAttributes: map[string]snstypes.MessageAttributeValue{
			"AWS.SNS.SMS.SMSType": {
				DataType:    aws.String("String"),
				StringValue: aws.String("Transactional"),
			},
		},
	}

	_, err := n.snsClient.Publish(ctx, input)
	if err != nil {
		slog.Error("SNS send failed",
			"phone", phoneNumber,
			"error", err,
		)
	}
}

// formatAlertBody builds the email body for an alert notification.
func formatAlertBody(check db.Check) string {
	lastPing := "never"
	if check.LastPingAt != nil {
		lastPing = check.LastPingAt.Format(time.RFC3339)
	}
	return fmt.Sprintf(
		"Check: %s\nStatus: %s\nLast ping: %s\n\nThis check has missed its expected ping window.",
		check.Name,
		check.Status,
		lastPing,
	)
}

// formatRecoveryBody builds the email body for a recovery notification.
func formatRecoveryBody(check db.Check) string {
	lastPing := "never"
	if check.LastPingAt != nil {
		lastPing = check.LastPingAt.Format(time.RFC3339)
	}
	return fmt.Sprintf(
		"Check: %s\nStatus: %s\nLast ping: %s\n\nThis check has recovered and is reporting healthy again.",
		check.Name,
		check.Status,
		lastPing,
	)
}

// formatAlertSMS builds a short SMS message for an alert notification.
func formatAlertSMS(check db.Check) string {
	delta := "never"
	if check.LastPingAt != nil {
		delta = timeSince(*check.LastPingAt)
	}
	return fmt.Sprintf("[cronhealth] %s missed ping. Last seen: %s", check.Name, delta)
}

// formatRecoverySMS builds a short SMS message for a recovery notification.
func formatRecoverySMS(check db.Check) string {
	return fmt.Sprintf("[cronhealth] %s recovered.", check.Name)
}

// timeSince returns a human-readable duration string since the given time.
func timeSince(t time.Time) string {
	d := time.Since(t).Truncate(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh %dm ago", int(d.Hours()), int(d.Minutes())%60)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd %dh ago", days, hours)
}

// NoopNotifier is a no-op implementation of Notifier for testing and development.
// It logs notifications instead of sending them.
type NoopNotifier struct{}

// SendAlert logs the alert but does not send any notification.
func (n *NoopNotifier) SendAlert(_ context.Context, check db.Check, channels []db.NotificationChannel) error {
	slog.Info("noop: alert notification",
		"check", check.Name,
		"check_id", check.ID,
		"channels", len(channels),
	)
	return nil
}

// SendRecovery logs the recovery but does not send any notification.
func (n *NoopNotifier) SendRecovery(_ context.Context, check db.Check, channels []db.NotificationChannel) error {
	slog.Info("noop: recovery notification",
		"check", check.Name,
		"check_id", check.ID,
		"channels", len(channels),
	)
	return nil
}
