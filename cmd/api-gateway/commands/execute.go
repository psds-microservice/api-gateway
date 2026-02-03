package commands

import (
	"fmt"
	"os"
)

// Version, Commit, BuildDate — заполняются при сборке
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

// Execute — точка входа из корневого main.go (api-00003-совместимость)
func Execute() {
	if err := handleCommand(os.Args[1:]); err != nil {
		fmt.Printf("Ошибка: %v\n", err)
		os.Exit(1)
	}
}
