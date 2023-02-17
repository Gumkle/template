package main

import (
	"example_generated_project/pkg/infra/config"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	applicationConfig, err := config.NewApplicationConfig()
	if err != nil {
		log.Fatalf("Failed to read configuration file: %v", err)
	}
	fmt.Printf("Application launched! Welcome to %s!\n", applicationConfig.ApplicationName)
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"hello": "Hello world!",
		})
	})
	r.Run()	// listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
