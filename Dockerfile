# --- Build stage ---
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/cronhealth-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/cronhealth-poller ./cmd/poller

# --- API image ---
FROM scratch AS api
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /bin/cronhealth-api /cronhealth-api
ENTRYPOINT ["/cronhealth-api"]

# --- Poller image ---
FROM scratch AS poller
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /bin/cronhealth-poller /cronhealth-poller
ENTRYPOINT ["/cronhealth-poller"]
