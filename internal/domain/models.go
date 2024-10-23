package domain

import (
	"time"

	"gorm.io/gorm"
)

type Appointment struct {
	gorm.Model
	AppointmentId    int
	PatientId        string
	DoctorId         string
	SpecializationId int32
	AppointmentTime  time.Time
	Duration         time.Duration
	Status           string
	PaymentId        string
	Type             string
}

type Availability struct {
	Id         int
	DoctorId   string
	DoctorName string
	DateTime   time.Time
}
type VideoTreatment struct {
	gorm.Model
	VideoTreatmentId string
	AppointmentId    int 
	Appointment      Appointment
}
type AppointmentEvent struct {
	AppointmentId   int
	Email string
	VideoURL string
	DoctorId        string
	AppointmentDate string
	Type            string
}