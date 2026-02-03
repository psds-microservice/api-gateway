# Сравнение api-gateway с user-service

Сравнение структуры и функционала проектов `api-gateway` и `user-service`.

---

## Структура директорий

| Компонент | user-service | api-gateway | Комментарий |
|-----------|--------------|-------------|-------------|
| `.gitattributes` | ✅ | ❌ | user-service: `infra/*.sh text eol=lf` |
| `api/` | ✅ openapi.json, swagger, README | ❌ | OpenAPI/Swagger из proto |
| `third_party/` | ✅ google/api (annotations, http) | ❌ | Для protoc-gen-openapiv2 |
| `build/` | используется Makefile | dev ссылается на `build/local/.air.toml` | api-gateway: папки air может не быть |
| `main.go` (корень) | ❌ | ✅ | api-gateway: точка входа `go run .` |

---

## cmd/ — точка входа и CLI

| Аспект | user-service | api-gateway |
|--------|--------------|-------------|
| **Фреймворк CLI** | Cobra (`spf13/cobra`) | Собственный (flag + `handleCommand`) |
| **Структура** | `cmd/root.go`, `api.go`, `migrate.go`, `seed.go` + `cmd/user-service/main.go` | `cmd/api-gateway/main.go`, `commands.go`, `dual_server.go` |
| **Команды** | `api`, `migrate`, `seed` | `server`, `worker`, `version` (через `--mode`) |
| **Точка входа** | `./cmd/user-service` | `./cmd/api-gateway` |

**Чего не хватает в api-gateway:**
- Cobra для единообразного CLI
- Отдельные команды `api`, `migrate`, `seed` (если планируется БД)

---

## pkg/ — публичные пакеты

| Папка | user-service | api-gateway |
|-------|--------------|-------------|
| `constants/` | http.go, paths.go, roles.go | ❌ |
| `gen/` | user_service/*.pb.go | gen/*.pb.go (плоская структура) |
| `*_service/` proto | user_service/user_service.proto | api_gateway/video.proto, client_info.proto, common.proto |

**Чего не хватает в api-gateway:**
- `pkg/constants/` — HTTP-коды, пути, роли, константы


---

## Makefile

| Цель | user-service | api-gateway |
|------|--------------|-------------|
| `proto`, `proto-build`, `proto-generate` | ✅ | ✅ |
| `proto-openapi` | ✅ OpenAPI из .proto | ❌ |
| `proto-clean` | ✅ | ❌ |
| `proto-pkg`, `proto-pkg-simple`, `proto-pkg-script` | ✅ | ❌ (есть scripts/proto-gen.ps1) |
| `build` (с ldflags Version, Commit, BuildDate) | ✅ | ❌ |
| `run` (через build) | ✅ | go run без build |
| `run-dev` | ✅ | ❌ |
| `migrate`, `migrate-create` | ✅ | ❌ |
| `seed`, `db-init` | ✅ | ❌ |
| `worker` | ✅ | ❌ |
| `health-check` | ✅ | ❌ |
| `test` (с proto, -race, coverage) | ✅ | базовый `go test ./...` |
| `bench` | ✅ | ❌ |
| `load-test` (k6) | ✅ | ❌ |
| `lint`, `vet`, `fmt` | ✅ | ❌ |
| `security-check` (gosec) | ✅ | ❌ |
| `install-deps` | ✅ | ❌ |
| `update` (с vendor) | ✅ | ✅ |
| `generate-docs` | ✅ | ❌ |

---

## deployments/

| Аспект | user-service | api-gateway |
|--------|--------------|-------------|
| Порт gRPC в docker-compose | 9090 | ❌ (только 8080) |
| Переменные APP_HOST, APP_PORT, GRPC_PORT | ✅ | ❌ |
| Dockerfile CMD | `["./user-service", "api"]` | `["./api-gateway"]` |
| Go версия в Dockerfile | 1.24 | 1.21 |
| ldflags `-s -w` | ✅ | ❌ |

api-gateway в dual-режиме должен слушать gRPC на 9090, в docker-compose порт не проброшен.

---

## Конфигурация

| Аспект | user-service | api-gateway |
|--------|--------------|-------------|
| Источник | Только .env | config/config.yaml + .env |
| godotenv | ❌ | ✅ |
| YAML | ❌ | ✅ |

---

## CI/CD (.github/workflows)

Оба проекта имеют похожий `ci.yml`: checkout, setup-go 1.21, tidy, build, test.

**Рекомендации для api-gateway:**
- Добавить `make proto` перед build (если proto не коммитится)
- Выровнять go-version с user-service (1.24)

---

## Итоговый чек-лист: чего не хватает в api-gateway

### Структура и инфраструктура
- [ ] `.gitattributes` (для shell-скриптов infra)
- [ ] `pkg/constants/` (http, paths, roles)
- [ ] `third_party/google/api/` (если нужен OpenAPI из proto)
- [ ] `api/` с OpenAPI/Swagger (если нужна документация из proto)
- [ ] `build/local/.air.toml` (для `make dev`)

### CLI
- [ ] Cobra вместо custom flag-based CLI
- [ ] Отдельные команды: `api`, `migrate`, `seed` (при добавлении БД)

### Internal
- [ ] `internal/auth/` (JWT, blacklist)
- [ ] `internal/middleware/auth`
- [ ] `internal/validator`
- [ ] `internal/mapper` (при использовании БД)
- [ ] `internal/consumer` (при использовании очередей)

### Makefile
- [ ] BUILD_INFO, COMMIT_HASH, BUILD_DATE в ldflags
- [ ] `proto-openapi`, `proto-clean`
- [ ] `run-dev`, `health-check`
- [ ] `lint`, `vet`, `fmt`, `security-check`
- [ ] `bench`, `load-test`
- [ ] `install-deps`, `generate-docs`

### Deployments
- [ ] Порт 9090 для gRPC в docker-compose
- [ ] APP_HOST, APP_PORT, GRPC_PORT в environment
- [ ] CMD `["./api-gateway", "server", "--grpc-port=9090"]` для dual-режима
- [ ] Обновить Go в Dockerfile до 1.24
- [ ] ldflags `-s -w` в сборке

### Database (при необходимости)
- [ ] Реальные миграции в `database/migrations/`
- [ ] Seeds в `database/seeds/`
- [ ] `internal/database/`, `internal/command/migrate.go`, `seed.go`
