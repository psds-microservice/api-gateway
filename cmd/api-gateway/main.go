package main

import (
	"log"

	"github.com/psds-microservice/api-gateway/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
