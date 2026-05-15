# syntax=docker/dockerfile:1

FROM docker.io/golang:1.26.2-alpine AS builder

WORKDIR /app

ENV CGO_ENABLED=0 \
    GOOS=linux

COPY certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY go.mod go.sum ./
ENV GOPROXY=http://host.docker.internal:3000
ENV GOSUMDB=off
RUN go mod download

COPY . .

RUN go build -trimpath -ldflags="-s -w" -o monitor ./cmd/monitor


FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /

COPY --from=builder /app/monitor /monitor
COPY certs/ca-certificates.crt /etc/ssl/certs/

USER nonroot:nonroot

ENTRYPOINT ["/monitor"]