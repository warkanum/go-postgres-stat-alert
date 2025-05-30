FROM golang:alpine AS builder
WORKDIR /app
COPY ../ .
RUN CGO_ENABLED=0 go build -o postgres-stat-alert-linux-amd64

FROM scratch
COPY --from=builder /app/postgres-stat-alert-linux-amd64 /
ENTRYPOINT ["/postgres-stat-alert-linux-amd64"]