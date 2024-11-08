package main

import (
	"log"
	"os"

	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/config"
)

func main() {
	config.LoadEnv()
	port := os.Getenv("APPOINTMENT_PORT")
	listener, server := config.GRPCSetup(port)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
}
