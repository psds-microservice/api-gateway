.PHONY: help proto build run run-dual tidy test dev clean update proto-build proto-generate proto-generate-local proto-generate-docker proto-openapi

APP_NAME = api-gateway
BIN_DIR = bin
PROTO_ROOT = pkg/api_gateway
GEN_DIR = pkg/gen
GO_MODULE = github.com/psds-microservice/api-gateway
PROTOC_IMAGE = local/protoc-go:latest
OPENAPI_OUT = api
OPENAPI_SPEC = $(OPENAPI_OUT)/openapi.json

.DEFAULT_GOAL := help

help:
	@echo "API Gateway - Makefile"
	@echo ""
	@echo "Commands:"
	@echo "  make proto        - Build image and generate proto (like user-service)"
	@echo "  make proto-build  - Build protoc image from psds-microservice/infra"
	@echo "  make proto-generate - Generate Go code (local protoc or Docker)"
	@echo "  make proto-openapi - Generate OpenAPI/Swagger from proto"
	@echo "  make build      - Build binary"
	@echo "  make run        - Run HTTP server (simple mode)"
	@echo "  make run-dual   - Run dual server (HTTP + gRPC)"
	@echo "  make dev        - Run with hot reload (requires air)"
	@echo "  make tidy       - go mod tidy"
	@echo "  make test       - Run tests"
	@echo "  make update     - Update dependencies"
	@echo "  make clean      - Clean build artifacts"

## Proto (как в user-service)
proto: proto-build proto-generate

## OpenAPI/Swagger из proto (единый источник — .proto)
proto-openapi:
	@command -v protoc >/dev/null 2>&1 || (echo "Установите protoc" && exit 1); \
	command -v protoc-gen-openapiv2 >/dev/null 2>&1 || (echo "Установите: go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest" && exit 1)
	@mkdir -p $(OPENAPI_OUT)
	@PATH="$$(go env GOPATH)/bin:$$PATH"; \
	protoc -I $(PROTO_ROOT) -I third_party \
		--openapiv2_out=$(OPENAPI_OUT) \
		--openapiv2_opt=logtostderr=true \
		--openapiv2_opt=allow_merge=true \
		--openapiv2_opt=merge_file_name=openapi \
		$(PROTO_ROOT)/video.proto $(PROTO_ROOT)/client_info.proto
	@if [ -f $(OPENAPI_OUT)/openapi.swagger.json ]; then cp $(OPENAPI_OUT)/openapi.swagger.json $(OPENAPI_OUT)/openapi.json; echo "OpenAPI: $(OPENAPI_SPEC)"; elif [ -f $(OPENAPI_OUT)/openapi.json ]; then echo "OpenAPI: $(OPENAPI_SPEC)"; else echo "Проверьте вывод protoc выше"; fi

# Сборка образа: infra клонируется во временную папку, не в проект
INFRA_TMP := $(or $(TMPDIR),/tmp)/api-gateway-infra-build
proto-build:
	@echo "Building protoc-go image..."
	@rm -rf "$(INFRA_TMP)" && mkdir -p "$(INFRA_TMP)" && \
		git clone --depth 1 https://github.com/psds-microservice/infra.git "$(INFRA_TMP)/repo" && \
		mkdir -p "$(INFRA_TMP)/repo/infra" && cp "$(INFRA_TMP)/repo/docker-entrypoint.sh" "$(INFRA_TMP)/repo/infra/" && \
		docker build -t $(PROTOC_IMAGE) -f "$(INFRA_TMP)/repo/protoc-go.Dockerfile" "$(INFRA_TMP)/repo" && \
		rm -rf "$(INFRA_TMP)"
	@echo "Docker image built"

# Генерация: локальный protoc или Docker
proto-generate:
	@PATH="$$(go env GOPATH 2>/dev/null)/bin:$$PATH"; \
	if command -v protoc >/dev/null 2>&1 && command -v protoc-gen-go >/dev/null 2>&1 && command -v protoc-gen-go-grpc >/dev/null 2>&1; then \
		$(MAKE) proto-generate-local; \
	else \
		$(MAKE) proto-generate-docker; \
	fi

proto-generate-local:
	@echo "Generating Go code (local protoc)..."
	@mkdir -p $(GEN_DIR)
	@command -v protoc-gen-grpc-gateway >/dev/null 2>&1 || (echo "Установите: go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest" && exit 1)
	@PATH="$$(go env GOPATH)/bin:$$PATH"; \
	for f in $(PROTO_ROOT)/*.proto; do \
		[ -f "$$f" ] || continue; \
		echo "Processing: $$f"; \
		protoc -I $(PROTO_ROOT) -I third_party \
			--go_out=. --go_opt=module=$(GO_MODULE) \
			--go-grpc_out=. --go-grpc_opt=module=$(GO_MODULE) \
			--grpc-gateway_out=. --grpc-gateway_opt=module=$(GO_MODULE) \
			"$$f" || exit 1; \
	done
	@echo "Generated in $(GEN_DIR)"

proto-generate-docker:
	@echo "Generating Go code (Docker)..."
	@mkdir -p $(GEN_DIR)
	@docker run --rm \
		-v "$(CURDIR):/workspace" \
		-w /workspace \
		--entrypoint sh \
		$(PROTOC_IMAGE) \
		-c ' \
		PROTO_ROOT="$(PROTO_ROOT)" MODULE="$(GO_MODULE)" && \
		find $$PROTO_ROOT -name "*.proto" 2>/dev/null | while read f; do \
		echo "Processing $$f" && \
		protoc -I $$PROTO_ROOT -I third_party -I /include \
		--go_out=. --go_opt=module=$$MODULE \
		--go-grpc_out=. --go-grpc_opt=module=$$MODULE \
		--grpc-gateway_out=. --grpc-gateway_opt=module=$$MODULE \
		"$$f" || exit 1; \
		done && echo "Generated in $(GEN_DIR)" \
		'
	@echo "Proto files generated"

build: proto
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/$(APP_NAME) ./cmd/api-gateway
	@echo "Build complete: $(BIN_DIR)/$(APP_NAME)"

run:
	go run ./cmd/api-gateway

run-dual:
	@echo "Starting dual server (HTTP:8080 + gRPC:9090)..."
	go run ./cmd/api-gateway api --debug --grpc-port=9090

tidy:
	go mod tidy

update:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy
	go mod vendor
	$(MAKE) proto
	$(MAKE) proto-openapi
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
