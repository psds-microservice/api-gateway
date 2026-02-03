# API (OpenAPI/Swagger из proto)

## Генерация OpenAPI

Установить плагин:
```bash
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
```

Сгенерировать api/openapi.json:
```bash
make proto-openapi
```

## Swagger UI

После запуска сервиса:
- **Swagger UI:** http://localhost:8080/swagger/index.html
- **Спека:** http://localhost:8080/openapi.json
