# Standalone image build (GitHub/GHCR) using the committed, checksum-verified module graph.
FROM golang:1.26-alpine AS builder
RUN apk add --no-cache git ca-certificates
ENV GOPROXY=https://proxy.golang.org,direct
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /notifications ./cmd/notifications

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /notifications /app/notifications
COPY --from=builder /src/migrations /app/migrations
EXPOSE 10110
USER nobody
CMD ["/app/notifications"]
