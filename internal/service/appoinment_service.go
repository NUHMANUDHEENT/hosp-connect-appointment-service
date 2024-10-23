package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/NUHMANUDHEENT/hosp-connect-pb/proto/appointment"
	doctorpb "github.com/NUHMANUDHEENT/hosp-connect-pb/proto/doctor"
	patientpb "github.com/NUHMANUDHEENT/hosp-connect-pb/proto/patient"
	paymentpb "github.com/NUHMANUDHEENT/hosp-connect-pb/proto/payment"
	"github.com/google/uuid"

	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/config"
	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/domain"
	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/repository"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AppointmentService interface {
	CheckAvailability(CategoryId int32, reqtime time.Time) ([]domain.Availability, error)
	CheckAvailabilityByDoctorId(doctorID string) (*appointment.CheckAvailabilityByDoctorIdResponse, error)
	ConfirmAppointment(appointment domain.Appointment) (string, error)
	CreateRoomForVideoTreatment(patientId, doctorId string, specializationId int64) (string, error)
	GetUpcomingAppointments(patientId string) ([]domain.Appointment, error)
}

type appointmentService struct {
	repo          repository.AppointmentRepository
	DoctorClient  doctorpb.DoctorServiceClient
	PaymentClient paymentpb.PaymentServiceClient
	PatientClient patientpb.PatientServiceClient
}

// Initialize appointment service with gRPC client
func NewAppoinmentService(repo repository.AppointmentRepository, DoctorClient doctorpb.DoctorServiceClient, paymentClient paymentpb.PaymentServiceClient, patientClient patientpb.PatientServiceClient) AppointmentService {

	return &appointmentService{
		repo:          repo,
		DoctorClient:  DoctorClient,
		PaymentClient: paymentClient,
		PatientClient: patientClient,
	}
}
func (a *appointmentService) CheckAvailability(CategoryId int32, reqtime time.Time) ([]domain.Availability, error) {
	availability := []domain.Availability{}

	// Convert Go's time.Time to Protobuf Timestamp
	reqTimestamp := timestamppb.New(reqtime)

	// Call the DoctorService's GetAvailability method
	resp, err := a.DoctorClient.GetAvailability(context.Background(), &doctorpb.GetAvailabilityRequest{
		CategoryId:        CategoryId,
		RequestedDateTime: reqTimestamp, // Passing the converted timestamp
	})
	if err != nil {
		return availability, err
	}

	// Debugging: Print the entire response
	fmt.Println("Full Response:", resp) // Check full response structure

	for _, slot := range resp.AvailableSlots {
		availability = append(availability, domain.Availability{
			DoctorId:   slot.DoctorId,
			DoctorName: slot.DoctorName,
		})
	}

	fmt.Println("Final Availability:", availability)
	return availability, nil
}
func (s *appointmentService) CheckAvailabilityByDoctorId(doctorID string) (*appointment.CheckAvailabilityByDoctorIdResponse, error) {
	available, err := s.DoctorClient.CheckAvailabilityByDoctorId(context.Background(), &doctorpb.CheckAvailabilityByDoctorIdRequest{
		DoctorId: doctorID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check doctor availability: %w", err)
	}

	availability := &appointment.CheckAvailabilityByDoctorIdResponse{
		DoctorId: available.DoctorId,
	}
	for _, v := range available.DoctorAvailability {
		availability.DoctorAvailability = append(availability.DoctorAvailability, &appointment.DoctorAvailability{
			DateTime:    v.DateTime,
			IsAvailable: v.IsAvailable,
		})
	}
	return availability, nil
}
func (s *appointmentService) ConfirmAppointment(appointment domain.Appointment) (string, error) {
	// Check if the requested time slot is available
	// available, err := s.DoctorClient.CheckAvailabilityByDoctorId(context.Background(), &doctorpb.CheckAvailabilityByDoctorIdRequest{
	// 	DoctorId: doctorId,
	// })
	// if err != nil {
	// 	return "failed to call doctor service", err
	// }
	// for _, v := range available.DoctorAvailability {
	// 	if  v.IsAvailable =="unavailable" && reqTime == v.DateTime.  {
	// 		return "", errors.New("doctor is not available on this date")
	// 	}
	// }
	isAvailable, message, err := s.repo.IsDoctorAvailable(appointment.DoctorId, appointment.PatientId, appointment.AppointmentTime, time.Hour)
	if err != nil {
		return "", err
	}
	if !isAvailable {
		return "", errors.New(message)
	}

	// Retrieve the latest appointment ID and increment it for the new appointment
	latestAppointmentId, err := s.repo.GetLatestAppointmentId()
	if err != nil {
		return "", errors.New("failed to fetch latest appointment ID")
	}
	newAppointmentId := latestAppointmentId + 1

	appointment.AppointmentId = newAppointmentId
	appointment.Duration = time.Hour
	appointment.Status = "Confirmed"

	Resp, err := s.PaymentClient.CreateRazorOrderId(context.Background(), &paymentpb.CreateRazorOrderIdRequest{
		PatientId:     appointment.PatientId,
		Amount:        200,
		AppointmentId: int64(newAppointmentId),
		Type:          "appointment fee",
	})
	if err != nil {
		return "", errors.New("failed to call payment service")
	} else if Resp.Status != "success" {
		return "", errors.New(Resp.Message)
	}
	appointment.PaymentId = Resp.OrderId

	// Save the appointment
	if err := s.repo.ConfirmAppointment(appointment); err != nil {
		return "", err
	}

	return Resp.PaymentUrl, nil
}

// get upcoming appointment to show patients
func (s *appointmentService) GetUpcomingAppointments(patientId string) ([]domain.Appointment, error) {

	// Fetch appointments from the repository
	appointments, err := s.repo.FetchAppointmentsByPatient(patientId)
	if err != nil {
		return nil, err
	}

	return appointments, nil
}

func (d *appointmentService) CreateRoomForVideoTreatment(patientId, doctorId string, specializationId int64) (string, error) {
	check, resp, err := d.repo.CheckVideoAppoitment(patientId)
	if err != nil {
		return "", err
	}
	if !check {
		return "", errors.New("patient don't have appointment")
	}
	uuid := uuid.New()
	roomId := uuid.String()

	err = d.repo.SaveVideoAppointment(uuid.String(), resp.AppointmentId, int(specializationId))
	if err != nil {
		return "", err
	}
	roomURL := fmt.Sprintf("http://localhost:8080/api/v1/doctor/video-call?room=%s", roomId)

	profile, err := d.PatientClient.GetProfile(context.Background(), &patientpb.GetProfileRequest{
		PatientId: patientId,
	})
	if err != nil {
		return "", err
	}
	PatientRoomUrl := fmt.Sprintf("http://localhost:8080/api/v1/patient/video-call?room=%s", roomId)
	fmt.Printf("http://localhost:8080/api/v1/patient/video-call?room=%s", roomId)
	err = config.HandleAppointmentNotification(domain.AppointmentEvent{
		AppointmentId:   resp.AppointmentId,
		Email:           profile.Email,
		VideoURL:        PatientRoomUrl,
		DoctorId:        doctorId,
		AppointmentDate: resp.AppointmentTime.Format("2016-02-01"),
		Type:            resp.Type,
	})
	if err != nil {
		errors.New("failed to produce video appointment event")
	}
	return roomURL, nil
}

// // sendEmailToPatient sends an email to the patient with the video call room URL
// func (d *appointmentService) sendEmailToPatient(patientId string, roomURL string) error {
// 	// Fetch patient email from the repository
// 	patientEmail, err := d.repo.GetPatientEmail(patientId)
// 	if err != nil {
// 		return err
// 	}

// 	// Setup email details
// 	from := "your_email@gmail.com"
// 	password := "your_password"
// 	to := []string{patientEmail}
// 	smtpHost := "smtp.gmail.com"
// 	smtpPort := "587"

// 	// Email content
// 	subject := "Join Your Video Treatment"
// 	body := fmt.Sprintf("Hello, please join your video treatment by clicking the following link: %s", roomURL)
// 	message := fmt.Sprintf("From: %s\nTo: %s\nSubject: %s\n\n%s", from, patientEmail, subject, body)

// 	// Set up the email authentication and send the email
// 	auth := smtp.PlainAuth("", from, password, smtpHost)
// 	err = smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, []byte(message))
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// // notifyDoctor sends an email notification to the doctor (optional)
// func (d *appointmentService) notifyDoctor(doctorId string, roomURL string) error {
// 	// Fetch doctor email from the repository
// 	doctorEmail, err := d.repo.GetDoctorEmail(doctorId)
// 	if err != nil {
// 		return err
// 	}

// 	// Setup email details
// 	from := "your_email@gmail.com"
// 	password := "your_password"
// 	to := []string{doctorEmail}
// 	smtpHost := "smtp.gmail.com"
// 	smtpPort := "587"

// 	// Email content
// 	subject := "Join Your Patient Video Call"
// 	body := fmt.Sprintf("Hello Doctor, your patient is ready for a video treatment. Join here: %s", roomURL)
// 	message := fmt.Sprintf("From: %s\nTo: %s\nSubject: %s\n\n%s", from, doctorEmail, subject, body)

// 	// Set up the email authentication and send the email
// 	auth := smtp.PlainAuth("", from, password, smtpHost)
// 	err = smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, []byte(message))
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }
