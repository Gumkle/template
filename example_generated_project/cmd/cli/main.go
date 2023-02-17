package main

import (
	"example_generated_project/pkg/infra/config"
	"fmt"
	"log"
)

func main() {
	applicationConfig, err := config.NewApplicationConfig()
	if err != nil {
		log.Fatalf("Failed to read configuration file: %v", err)
	}
	fmt.Printf("Hello world! Welcome to %s!\n", applicationConfig.ApplicationName)
}
