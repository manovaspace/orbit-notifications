# Standalone image build (GitHub/GHCR) using the committed, checksum-verified module graph.
FROM golang:1.27.3-alpine AS builder
RUN apk add --no-cache git ca-certificates
ENV GOPROXY=https://proxy.golang.org,direct
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /notifications ./cmd/notifications

FROM alpine:3.24
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /notifications /app/notifications
EXPOSE 10012 10013
USER nobody
CMD ["/app/notifications"]
