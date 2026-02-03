# Aligned with https://github.com/psds-microservice/infra (for psds)
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache \
    protobuf \
    protobuf-dev \
    git \
    make \
    curl \
    wget \
    unzip

# Устанавливаем protoc-gen-go и protoc-gen-go-grpc (фиксированные версии)
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.33.0 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# Для поддержки импортов google/api/annotations.proto (если понадобится)
RUN mkdir -p /include && \
    wget -q -O /tmp/googleapis.zip https://github.com/googleapis/googleapis/archive/master.zip && \
    unzip -q /tmp/googleapis.zip -d /tmp && \
    mv /tmp/googleapis-master/google /include/ && \
    rm -rf /tmp/googleapis.zip /tmp/googleapis-master

ENV PATH="/root/go/bin:${PATH}"
COPY infra/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

WORKDIR /workspace
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
