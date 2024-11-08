package repository

import (
	"errors"
	"fmt"
	"time"

	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/domain"
	"gorm.io/gorm"
)

type AppointmentRepository interface {
	IsDoctorAvailable(doctorId string, patientId string, reqTime time.Time, duration time.Duration) (bool, string, string, error)
	ConfirmAppointment(appointment domain.Appointment) error
	CancelAppointment(appointment domain.Appointment, reason string) (string, error)
	GetLatestAppointmentId() (int, error)
	FetchAppointmentsByPatient(patientId string) ([]domain.Appointment, error)
	CheckVideoAppoitment(patientId string) (bool, domain.Appointment, error)
	SaveVideoAppointment(roomid string, appointmentid, specializationId int) error
	GetAppointmentDetails(orderid string) (domain.Appointment, error)
	FetchTodayAppointments() ([]domain.Appointment, error)
	CreateSpecialization(specialize domain.Specialization) (string, error)
	GetSpecializationStats(param string) ([]domain.SpecializationStats, error)
	GetTotalAppointment(param string) (int, error)
}
type appointmentRepository struct {
	db *gorm.DB
}

func NewAppoinmentRepository(db *gorm.DB) AppointmentRepository {
	return &appointmentRepository{
		db: db,
	}
}
func (r *appointmentRepository) IsDoctorAvailable(doctorId string, patientId string, reqTime time.Time, duration time.Duration) (bool, string, string, error) {
	var appointment domain.Appointment

	// Check if the patient has already booked an appointment that day
	err := r.db.Model(&domain.Appointment{}).
		Where("doctor_id = ? AND patient_id = ? AND DATE(appointment_time) = DATE(?)", doctorId, patientId, reqTime).
		First(&appointment).Error
	if err == nil {
		if appointment.Status == "Pending" {
			return false, "http://localhost:8080/api/v1/payment?orderId=" + appointment.PaymentId, fmt.Sprintf("you have already booked an appointment for this day (%v) but not completed the payment so Please complete payment using belove URL and confirm your shedule!", appointment.AppointmentTime), nil
		}
		return false, "", "", fmt.Errorf("you have already booked an appointment for this day (%v)", appointment.AppointmentTime)
	}

	// Check if there's an available slot for the requested time considering the duration
	var overlappingCount int64
	err = r.db.Model(&domain.Appointment{}).
		Where("doctor_id = ? AND appointment_time <= ? AND appointment_time + interval '1 hour' >= ?", doctorId, reqTime, reqTime.Add(duration)).
		Count(&overlappingCount).Error
	if err != nil {
		return false, "", "", err
	}

	// Return if there's a free slot
	if overlappingCount == 0 && isWithinWorkingHours(reqTime) {
		return true, "", "", nil
	}

	// If no available slot, suggest the closest available slot with a 4-hour gap
	suggestedTime := suggestAlternativeSlot(r.db, doctorId, reqTime)
	if suggestedTime.IsZero() {
		return false, "", "No available slots within working hours", nil
	}
	return false, "", "No slot available. Suggested next slot: " + suggestedTime.Format("3:04 PM"), nil
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
func (r *appointmentRepository) ConfirmAppointment(appointment domain.Appointment) error {
	return r.db.Create(&appointment).Error
}
func (r *appointmentRepository) CancelAppointment(appointment domain.Appointment, reason string) (string, error) {
	if err := r.db.Where("appointment_id=? AND patient_id=?", appointment.AppointmentId, appointment.PatientId).First(&appointment).Error; err != nil {
		return "", errors.New("appointment not found")
	}
	if appointment.Status == "pending" {
		return "", errors.New("this appointment payment not completed")
	} else if !appointment.AppointmentTime.After(time.Now()) {
		return "", errors.New("this appointment is already started")
	}
	if err := r.db.Model(&appointment).Update("status", "cancelled").Error; err !=
		nil {
		return "", err
	}
	return "Appointment cancelled successfully", nil
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
func (r *appointmentRepository) CheckVideoAppoitment(patientId string) (bool, domain.Appointment, error) {
	var appointment domain.Appointment
	err := r.db.Where("patient_id =? AND type = ? AND appointment_time >?", patientId, "video", time.Now()).First(&appointment).Error
	if err != nil {
		return false, domain.Appointment{}, errors.New("patient appointment not found")
	}
	return true, appointment, nil
}
func (r *appointmentRepository) SaveVideoAppointment(roomid string, appointmentid, specializationId int) error {
	appointment := domain.VideoTreatment{
		VideoTreatmentId: roomid,
		AppointmentId:    appointmentid,
	}
	if err := r.db.Create(&appointment).Error; err != nil {
		return err
	}
	return nil

}
func (r *appointmentRepository) GetAppointmentDetails(orderid string) (domain.Appointment, error) {
	var appointment domain.Appointment
	err := r.db.Where("order_id = ?", orderid).First(&appointment).Error
	if err != nil {
		return domain.Appointment{}, err
	}
	return appointment, nil
}
func (r *appointmentRepository) FetchTodayAppointments() ([]domain.Appointment, error) {
	var appointments []domain.Appointment
	err := r.db.Where("appointment_time >= ? AND appointment_time <= ?", time.Now(), time.Now().Add(12*time.Hour)).Find(&appointments).Error
	if err != nil {
		return nil, err
	}
	return appointments, nil

}
func (r *appointmentRepository) CreateSpecialization(specialize domain.Specialization) (string, error) {
	if err := r.db.Create(&specialize).Error; err != nil {
		return "Category is already exist", err
	}
	return "Category created successfully", nil
}
func (r *appointmentRepository) GetSpecializationStats(param string) ([]domain.SpecializationStats, error) {
	var results []struct {
		SpecializationName string
		AppointmentCount   int32
	}

	// Start building the base query
	query := r.db.Table("appointments").
		Select("specializations.name as specialization_name, COUNT(appointments.id) as appointment_count").
		Joins("JOIN specializations ON appointments.specialization_id = specializations.id").
		Group("specializations.name")

	// Add conditional filters based on the `param` value
	switch param {
	case "day":
		query = query.Where("DATE(appointments.appointment_time) = CURRENT_DATE")
	case "week":
		query = query.Where("appointment_time >= CURRENT_DATE - INTERVAL '7 days'")
	case "month":
		query = query.Where("appointment_time >= CURRENT_DATE - INTERVAL '1 month'")
		// "all" and "default" cases don’t need a `WHERE` clause, as they include all data
	}

	// Execute the query and handle any errors
	if err := query.Scan(&results).Error; err != nil {
		return nil, err
	}

	// Convert the results to the expected response format
	var specializationStats []domain.SpecializationStats
	for _, r := range results {
		specializationStats = append(specializationStats, domain.SpecializationStats{
			Name:  r.SpecializationName,
			Count: int(r.AppointmentCount),
		})
	}
	return specializationStats, nil
}
func (r *appointmentRepository) GetTotalAppointment(param string) (int, error) {
	var totalAppointment int64 // int64 for GORM Count compatibility

	query := r.db.Model(&domain.Appointment{})

	// Apply filter based on the 'param' argument
	switch param {
	case "day":
		query = query.Where("DATE_TRUNC('day', appointment_time) = DATE_TRUNC('day', CURRENT_TIMESTAMP)")
	case "week":
		query = query.Where("DATE_TRUNC('week', appointment_time) = DATE_TRUNC('week', CURRENT_TIMESTAMP)")
	case "month":
		query = query.Where("DATE_TRUNC('month', appointment_time) = DATE_TRUNC('month', CURRENT_TIMESTAMP)")
	}

	// Count the total number of appointments
	if err := query.Count(&totalAppointment).Error; err != nil {
		return 0, err
	}

	fmt.Printf("Fetching total appointments filtered by: %v, Value: %v\n", param, totalAppointment)
	return int(totalAppointment), nil
}
