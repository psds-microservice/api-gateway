package main

import (
	"log"

	"github.com/psds-microservice/api-gateway/cmd"
	_ "github.com/psds-microservice/infra" // для go mod vendor (proto-build)
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
