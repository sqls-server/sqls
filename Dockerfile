FROM golang:1.22-alpine AS builder

WORKDIR /build
COPY . .
RUN go build -o sqls .

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/sqls /app/sqls

ENTRYPOINT ["/app/sqls"]
