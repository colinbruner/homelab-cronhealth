export type CheckStatus = 'new' | 'up' | 'down' | 'alerting' | 'silenced';

export interface Check {
  id: string;
  name: string;
  slug: string;
  period_seconds: number;
  grace_seconds: number;
  status: CheckStatus;
  last_ping_at: string | null;
  last_alerted_at: string | null;
  created_at: string;
  created_by: string | null;
}

export interface Ping {
  id: string;
  check_id: string;
  received_at: string;
  source_ip: string | null;
  exit_code: number | null;
}

export interface Alert {
  id: string;
  check_id: string;
  started_at: string;
  resolved_at: string | null;
  alert_count: number;
  check_name?: string;
}

export interface Silence {
  id: string;
  check_id: string;
  silenced_by: string | null;
  starts_at: string;
  ends_at: string | null;
  reason: string | null;
  created_at: string;
}

export interface User {
  user_id: string;
  email: string;
}

export interface NotificationChannel {
  id: string;
  user_id: string;
  label: string;
  type: 'email' | 'sms';
  target: string;
  enabled: boolean;
  created_at: string;
}

export interface CreateCheckRequest {
  name: string;
  period_seconds: number;
  grace_seconds?: number;
  channel_ids?: string[];
}

export interface UpdateCheckRequest {
  name: string;
  period_seconds: number;
  grace_seconds?: number;
}

export interface SnoozeRequest {
  duration_minutes: number;
}

export interface SilenceRequest {
  reason?: string;
  ends_at?: string;
}

export interface ApiError {
  error: string;
}
