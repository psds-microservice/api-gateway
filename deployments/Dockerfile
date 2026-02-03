FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /api-gateway ./cmd/api-gateway

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /api-gateway .
EXPOSE 8080 9090
CMD ["./api-gateway", "api", "--grpc-port=9090"]
