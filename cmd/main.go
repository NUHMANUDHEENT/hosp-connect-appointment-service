package main

import (
	"log"
	"os"

	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/config"
	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/di"
)

func main() {
	config.LoadEnv()
	port := os.Getenv("APPT_PORT")
	di.EnsureTopicExists(os.Getenv("KAFKA_BROKER"),"appointment_topic")
	listener, server := config.GRPCSetup(port)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
}
