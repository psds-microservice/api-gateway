package errors

import "errors"

// Доменные ошибки. Транспортный слой (grpc_server, handler) маппит их в gRPC codes и HTTP status.
var (
	ErrStreamNotFound = errors.New("stream not found")
	ErrClientNotFound = errors.New("client not found")
	ErrInvalidRequest = errors.New("invalid request")
)
