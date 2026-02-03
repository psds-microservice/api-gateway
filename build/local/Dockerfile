# Local development — build and run from source
FROM golang:1.21-alpine
WORKDIR /app
RUN go install github.com/cosmtrek/air@latest
COPY go.mod ./
RUN go mod download
COPY . .
EXPOSE 8080
CMD ["air", "-c", "/app/build/local/.air.toml"]
