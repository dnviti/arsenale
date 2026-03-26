# Build stage
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /gocache .

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates wget

COPY --from=builder /gocache /usr/local/bin/gocache

EXPOSE 6380
EXPOSE 6381

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:6381/health || exit 1

ENTRYPOINT ["/usr/local/bin/gocache"]
