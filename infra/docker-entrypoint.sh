#!/bin/sh
# From https://github.com/psds-microservice/infra (for psds)
# Adapted for api-gateway: PROTO_ROOT=pkg/api_gateway, OUTPUT=pkg/gen
set -e

echo "Protoc Go Builder (psds-microservice/infra)"
echo "==========================================="

PROTO_ROOT="${PROTO_ROOT:-pkg/api_gateway}"
OUTPUT_DIR="${OUTPUT_DIR:-pkg/gen}"
INCLUDE_DIRS="-I ${PROTO_ROOT} -I /include"

if [ $# -gt 0 ]; then
  echo "Custom command: $@"
  exec "$@"
else
  echo "Proto root: ${PROTO_ROOT}"
  echo "Output dir: ${OUTPUT_DIR}"
  echo ""

  mkdir -p "${OUTPUT_DIR}"

  for proto_file in ${PROTO_ROOT}/*.proto; do
    [ -f "$proto_file" ] || continue
    rel_path="${proto_file#${PROTO_ROOT}/}"
    echo "Processing: ${rel_path}"

    protoc ${INCLUDE_DIRS} \
      --go_out="${OUTPUT_DIR}" --go_opt=paths=source_relative \
      --go-grpc_out="${OUTPUT_DIR}" --go-grpc_opt=paths=source_relative \
      "${proto_file}"

    if [ $? -eq 0 ]; then
      echo "  Generated OK"
    else
      echo "  Failed: ${rel_path}"
      exit 1
    fi
  done

  echo ""
  echo "All proto files generated: ${OUTPUT_DIR}/"
fi
