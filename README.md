# API Gateway

Дубликат [haqury/api-gateway](https://github.com/haqury/api-gateway) на основе архитектуры psds-microservice.

Dual-режим: HTTP REST API + gRPC для видеостриминга.

## Требования

- Go 1.21+
- protoc, protoc-gen-go, protoc-gen-go-grpc (для сборки)
- Docker (опционально, для `make proto-docker`)

## Инфраструктура (psds)

Как в [user-service](https://github.com/psds-microservice/user-service):

- **[psds-microservice/infra](https://github.com/psds-microservice/infra)**: образ для генерации proto. Инфра **не хранится** в проекте — при `make proto-build` клонируется во временную папку, собирается образ, папка удаляется.
- `common.proto`, `video.proto`, `client_info.proto` хранятся локально в `pkg/api_gateway/`. Без зависимости от helpy.

## Установка protoc-плагинов

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Убедитесь, что `$GOPATH/bin` или `~/go/bin` в PATH.

## Сборка

```bash
go mod download

# Генерация proto (как в user-service: local protoc или Docker)
make proto

make build
```

Windows: можно использовать `.\scripts\proto-gen.ps1` вместо `make proto`.

## Запуск

### Dual-режим (HTTP + gRPC)

```bash
make run-dual
# или
go run ./cmd/api-gateway server --debug --grpc-port=9090
```

- HTTP REST API: http://localhost:8080
- gRPC: localhost:9090
- Health: http://localhost:8080/health

### Простой HTTP-режим

```bash
make run
go run ./cmd/api-gateway
```

## Команды CLI

```
api-gateway server         # Запуск dual-сервера
api-gateway migrate        # Миграции (заглушка)
api-gateway worker         # Воркеры (заглушка)
api-gateway version        # Версия
api-gateway health-check   # Проверка здоровья (заглушка)
api-gateway generate-docs  # Генерация документации (заглушка)
api-gateway help           # Справка
```

Точка входа как в api-00003: из корня репозитория `go run .` или `go build -o bin/api-gateway .` — вызывается `commands.Execute()`.

## Конфигурация

- Конфигурация только из .env (без YAML)
- Секция `user_service` — подключение к user-service (host, port, timeouts); при недоступности используется stub-клиент
- Переменные окружения: см. `.env.example`

## API Endpoints

- `GET /health` — health check
- `GET /api/v1/status` — статус API
- `POST /api/v1/video/start` — старт стрима
- `POST /api/v1/video/frame` — отправка кадра (JSON или multipart)
- `POST /api/v1/video/stop` — остановка стрима
- `GET /api/v1/video/active` — активные стримы
- `GET /api/v1/video/stats/:client_id` — статистика
- `GET /api/v1/test/endpoints` — тестовые endpoints

## Структура проекта

```
cmd/api-gateway/     # Точка входа, CLI, dual server
internal/
  app/               # Application, Router (Gin)
  config/            # Config (.env)
  controller/        # VideoStream, ClientInfo сервисы
  grpc_server/       # gRPC VideoStreamService
  handler/           # HTTP handlers
  database/          # GORM, PostgreSQL
  model/             # Сущности
pkg/
  api_gateway/       # .proto файлы
  gen/               # Сгенерированный Go-код (make proto)
build/               # Dockerfiles, air
deployments/         # docker-compose
```

## Разработка

```bash
make dev     # hot reload (требуется air)
make test
make tidy
make update  # обновить зависимости (go get -u ./... && go mod tidy)
```
