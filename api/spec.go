// Package api содержит OpenAPI-спеку (api/openapi.json), встроенную в бинарь.
// Так GET /openapi.json работает независимо от текущей рабочей директории.
package api

import _ "embed"

//go:embed openapi.json
var OpenAPISpec []byte
