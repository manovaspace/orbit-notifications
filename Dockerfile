FROM golang:1.26-alpine AS builder
RUN apk add --no-cache git ca-certificates
ENV GOPROXY=direct
WORKDIR /src/orbit/orbit-notifications
COPY orbit/orbit-observability /src/orbit/orbit-observability
COPY orbit/orbit-notifications/go.mod orbit/orbit-notifications/go.sum ./
RUN go mod download
COPY orbit/orbit-notifications/ .
RUN CGO_ENABLED=0 go build -o /notifications ./cmd/notifications

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /notifications /app/notifications
COPY orbit/orbit-notifications/migrations /app/migrations
EXPOSE 10110
CMD ["/app/notifications"]
