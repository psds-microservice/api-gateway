.PHONY: help proto build run run-dual tidy test dev clean update

APP_NAME = api-gateway
BIN_DIR = bin
PROTO_ROOT = pkg/api_gateway
GEN_DIR = pkg/gen

.DEFAULT_GOAL := help

help:
	@echo "API Gateway - Makefile"
	@echo ""
	@echo "Commands:"
	@echo "  make proto           - Generate Go code from proto (requires protoc + plugins)"
	@echo "  make proto-docker     - Generate via Docker (image from psds-microservice/infra)"
	@echo "  make proto-docker-cmd - Same via Docker with explicit protoc command"
	@echo "  make build      - Build binary"
	@echo "  make run        - Run HTTP server (simple mode)"
	@echo "  make run-dual   - Run dual server (HTTP + gRPC)"
	@echo "  make dev        - Run with hot reload (requires air)"
	@echo "  make tidy       - go mod tidy"
	@echo "  make test       - Run tests"
	@echo "  make update     - Update dependencies (go get -u ./... && go mod tidy)"
	@echo "  make clean      - Clean build artifacts"

# common.proto берётся из github.com/psds-microservice/helpy (см. HELPY_MOD)
HELPY_MOD := github.com/psds-microservice/helpy
HELPY_DIR := $(shell go list -m -f '{{.Dir}}' $(HELPY_MOD) 2>/dev/null)

proto:
	@echo "Generating proto files (common from $(HELPY_MOD), requires protoc + plugins)..."
	@mkdir -p $(GEN_DIR)
	@if [ -z "$(HELPY_DIR)" ]; then echo "Run: go mod download && go mod tidy"; exit 1; fi
	@protoc -I "$(HELPY_DIR)" -I $(PROTO_ROOT) \
		--go_out=$(GEN_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(GEN_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_ROOT)/video.proto $(PROTO_ROOT)/client_info.proto && \
		rm -f $(GEN_DIR)/common.pb.go && \
		echo "Done. Check $(GEN_DIR)/" || (echo "Run: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"; echo "      go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"; exit 1)

# Proto image from psds-microservice/infra; common.proto из helpy (монтируется из go mod)
proto-docker:
	@HELPY_DIR=$$(go list -m -f '{{.Dir}}' $(HELPY_MOD) 2>/dev/null); \
	if [ -z "$$HELPY_DIR" ]; then echo "Run: go mod download"; exit 1; fi; \
	echo "Generating proto via Docker (infra + helpy)..."; \
	docker build -t api-gateway-protoc -f infra/protoc-go.Dockerfile . && \
	docker run --rm -v "$$(pwd):/workspace" -v "$$HELPY_DIR:/helpy:ro" -w /workspace \
		api-gateway-protoc sh -c 'protoc -I /helpy -I pkg/api_gateway -I /include \
		--go_out=pkg/gen --go_opt=paths=source_relative \
		--go-grpc_out=pkg/gen --go-grpc_opt=paths=source_relative \
		pkg/api_gateway/video.proto pkg/api_gateway/client_info.proto' && \
	rm -f $(GEN_DIR)/common.pb.go && \
	echo "Done. Check $(GEN_DIR)/"

# Docker: монтирует helpy из go mod, генерирует только video + client_info (common из helpy)
proto-docker-cmd:
	@HELPY_DIR=$$(go list -m -f '{{.Dir}}' $(HELPY_MOD) 2>/dev/null); \
	if [ -z "$$HELPY_DIR" ]; then echo "Run: go mod download"; exit 1; fi; \
	docker build -t api-gateway-protoc -f infra/protoc-go.Dockerfile . 2>/dev/null || true; \
	docker run --rm -v "$$(pwd):/workspace" -v "$$HELPY_DIR:/helpy:ro" -w /workspace api-gateway-protoc sh -c '\
		protoc -I /helpy -I pkg/api_gateway -I /include --go_out=pkg/gen --go_opt=paths=source_relative \
		--go-grpc_out=pkg/gen --go-grpc_opt=paths=source_relative \
		pkg/api_gateway/video.proto pkg/api_gateway/client_info.proto'; \
	rm -f pkg/gen/common.pb.go; echo "Done."

build: proto
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/$(APP_NAME) ./cmd/api-gateway
	@echo "Build complete: $(BIN_DIR)/$(APP_NAME)"

run:
	go run ./cmd/api-gateway

run-dual:
	@echo "Starting dual server (HTTP:8080 + gRPC:9090)..."
	go run ./cmd/api-gateway server --debug --grpc-port=9090

tidy:
	go mod tidy

update:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy
	make proto
	@echo "Done."

test:
	go test ./...

dev:
	@if command -v air > /dev/null 2>&1; then \
		air -c build/local/.air.toml; \
	else \
		echo "air not installed. Run: go install github.com/cosmtrek/air@latest"; \
		make run-dual; \
	fi

clean:
	rm -rf $(BIN_DIR) coverage.out
	go clean
