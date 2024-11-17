package utils

import (
	"log"

	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/service"
	"github.com/robfig/cron/v3"
)

func StartCroneSheduler(serviceInterface service.AppointmentService) {
	croneSheduler := cron.New()
	_, err := croneSheduler.AddFunc("52 15 * * *", serviceInterface.SendDialyReminders)
	if err != nil {
		log.Fatalf("Failed to schedule reminder job: %v", err)
	}
	croneSheduler.Start()

	select {}
}
