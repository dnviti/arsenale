# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.25.8
FROM docker.io/library/golang:${GO_VERSION}-alpine AS build

ARG SERVICE=control-plane-api
WORKDIR /src

COPY backend/go.mod ./
RUN go mod download

COPY backend/. ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/service ./cmd/${SERVICE}

FROM docker.io/library/alpine:3.22

RUN apk add --no-cache ca-certificates wget && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=build /out/service /usr/local/bin/service

USER appuser

ENTRYPOINT ["/usr/local/bin/service"]
