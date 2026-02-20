# No replace, no go.work: inject local modules into GOMODCACHE.
FROM golang:1.26-alpine AS builder
ENV GOMODCACHE=/go/pkg/mod
WORKDIR /app
COPY helpy /go/pkg/mod/github.com/psds-microservice/helpy@v0.1.0
COPY infra /go/pkg/mod/github.com/psds-microservice/infra@v0.0.3
COPY user-service /go/pkg/mod/github.com/psds-microservice/user-service@v0.0.0-20260217153622-6c08928941a7
COPY api-gateway ./api-gateway
WORKDIR /app/api-gateway
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /api-gateway ./cmd/api-gateway

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /api-gateway .
EXPOSE 8080 9090
CMD ["./api-gateway", "api", "--grpc-port=9090"]
