package constants

// Базовые пути API
const (
	BasePathAPI = "/api/v1"
	BasePathV1  = BasePathAPI
)

// Health
const (
	PathHealth = "/health"
	PathReady  = "/ready"
)

// Video (относительно BasePathAPI)
const (
	PathVideoStart    = "/video/start"
	PathVideoFrame    = "/video/frame"
	PathVideoStop     = "/video/stop"
	PathVideoActive   = "/video/active"
	PathVideoStats    = "/video/stats/:client_id"
	PathVideoStreams  = "/video/client/:client_id/streams"
	PathVideoStream   = "/video/stream/:stream_id"
	PathVideoAllStats = "/video/all-stats"
)

// Clients (относительно BasePathAPI)
const (
	PathClientsConnected    = "/clients/connected"
	PathClientsDisconnected = "/clients/disconnected"
	PathClientsUpdate       = "/clients/:client_id"
	PathClientsGet          = "/clients/:client_id"
	PathClientsActive       = "/clients/active"
)

// Swagger
const (
	PathSwagger = "/swagger"
)

// Status
const (
	PathStatus        = "/status"
	PathTestEndpoints = "/test/endpoints"
)
