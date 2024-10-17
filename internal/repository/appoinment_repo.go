package repository

import (
	"errors"
	"time"

	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/domain"
	"gorm.io/gorm"
)

type AppointmentRepository interface {
	IsDoctorAvailable(doctorId string, patientId string, reqTime time.Time, duration time.Duration) (bool, string, error)
	ConfirmAppointment(appointment *domain.Appointment) error
	GetLatestAppointmentId() (int, error)
	FetchAppointmentsByPatient(patientId string) ([]domain.Appointment, error)
	CheckVideoAppoitment(patientId string) (string, error)
}
type appointmentRepository struct {
	db *gorm.DB
}

func NewAppoinmentRepository(db *gorm.DB) AppointmentRepository {
	return &appointmentRepository{
		db: db,
	}
}
func (r *appointmentRepository) IsDoctorAvailable(doctorId string, patientId string, reqTime time.Time, duration time.Duration) (bool, string, error) {
	var appointment domain.Appointment

	// Check if the patient has already booked an appointment that day
	err := r.db.Model(&domain.Appointment{}).
		Where("doctor_id = ? AND patient_id = ? AND DATE(appointment_time) = DATE(?)", doctorId, patientId, reqTime).
		First(&appointment).Error
	if err == nil {
		return false, "You have already booked an appointment for this day", nil
	}

	// Check if there's an available slot for the requested time considering the duration
	var overlappingCount int64
	err = r.db.Model(&domain.Appointment{}).
		Where("doctor_id = ? AND appointment_time <= ? AND appointment_time + interval '1 hour' >= ?", doctorId, reqTime, reqTime.Add(duration)).
		Count(&overlappingCount).Error
	if err != nil {
		return false, "", err
	}

	// Return if there's a free slot
	if overlappingCount == 0 && isWithinWorkingHours(reqTime) {
		return true, "", nil
	}

	// If no available slot, suggest the closest available slot with a 4-hour gap
	suggestedTime := suggestAlternativeSlot(r.db, doctorId, reqTime)
	if suggestedTime.IsZero() {
		return false, "No available slots within working hours", nil
	}
	return false, "No slot available. Suggested next slot: " + suggestedTime.Format("3:04 PM"), nil
}
func isWithinWorkingHours(reqTime time.Time) bool {
	hour := reqTime.Hour()
	return hour >= 8 && hour < 19
}
func suggestAlternativeSlot(db *gorm.DB, doctorId string, reqTime time.Time) time.Time {
	const gapHours = 4
	for {
		reqTime = reqTime.Add(time.Hour * gapHours)
		if !isWithinWorkingHours(reqTime) {
			// If it's outside working hours, shift to the next working day at 8 AM
			reqTime = time.Date(reqTime.Year(), reqTime.Month(), reqTime.Day()+1, 8, 0, 0, 0, reqTime.Location())
		}

		// Check if the new time has an available slot
		var overlappingCount int64
		err := db.Model(&domain.Appointment{}).
			Where("doctor_id = ? AND appointment_time <= ? AND appointment_time + interval '1 hour' >= ?", doctorId, reqTime, reqTime).
			Count(&overlappingCount).Error

		if err == nil && overlappingCount == 0 {
			return reqTime
		}
	}
}

// ConfirmAppointment saves the confirmed appointment in the database
func (r *appointmentRepository) ConfirmAppointment(appointment *domain.Appointment) error {
	return r.db.Create(appointment).Error
}
func (r *appointmentRepository) GetLatestAppointmentId() (int, error) {
	var latestAppointment domain.Appointment

	// Query the database for the latest appointment by ID
	if err := r.db.Order("appointment_id desc").First(&latestAppointment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// If no appointments exist, start from 0
			return 0, nil
		}
		return 0, err // Return error if query fails
	}

	return latestAppointment.AppointmentId, nil
}
func (r *appointmentRepository) FetchAppointmentsByPatient(patientId string) ([]domain.Appointment, error) {
	var appointments []domain.Appointment
	err := r.db.Where("patient_id = ?", patientId).Order("appointment_time ASC").Find(&appointments).Error
	if err != nil {
		return nil, err
	}
	return appointments, nil
}
func (r *appointmentRepository) CheckVideoAppoitment(patientId string) (string, error) {
	var appointment domain.Appointment
	err := r.db.Where("patient_id =? AND type = ? AND appointment_time >?", patientId, "video", time.Now()).First(&appointment).Error
	if err != nil {
		return "", errors.New("patient appoitment not found")
	}
	return "patient appointment available", nil
}
