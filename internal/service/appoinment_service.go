package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/NUHMANUDHEENT/hosp-connect-pb/proto/appointment"
	doctorpb "github.com/NUHMANUDHEENT/hosp-connect-pb/proto/doctor"
	paymentpb "github.com/NUHMANUDHEENT/hosp-connect-pb/proto/payment"

	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/domain"
	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/repository"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AppointmentService interface {
	CheckAvailability(CategoryId int32, reqtime time.Time) ([]domain.Availability, error)
	CheckAvailabilityByDoctorId(doctorID string) (*appointment.CheckAvailabilityByDoctorIdResponse, error)
	ConfirmAppointment(patientId string, doctorId string, reqTime time.Time, specializationId int32) (string, error)
	GetUpcomingAppointments(patientId string) ([]domain.Appointment, error)
}

type appointmentService struct {
	repo          repository.AppointmentRepository
	DoctorClient  doctorpb.DoctorServiceClient
	PaymentClient paymentpb.PaymentServiceClient
}

// Initialize appointment service with gRPC client
func NewAppoinmentService(repo repository.AppointmentRepository, DoctorClient doctorpb.DoctorServiceClient, paymentClient paymentpb.PaymentServiceClient) AppointmentService {

	return &appointmentService{
		repo:          repo,
		DoctorClient:  DoctorClient,
		PaymentClient: paymentClient,
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
func (s *appointmentService) ConfirmAppointment(patientId string, doctorId string, reqTime time.Time, specializationId int32) (string, error) {
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
	isAvailable, message, err := s.repo.IsDoctorAvailable(doctorId, patientId, reqTime, time.Hour)
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

	// Confirm the appointment and generate payment details
	appointment := &domain.Appointment{
		AppointmentId:    newAppointmentId, // Set the new incremented appointment ID
		PatientId:        patientId,
		DoctorId:         doctorId,
		AppointmentTime:  reqTime,
		SpecializationId: specializationId,
		Duration:         time.Hour, // 1-hour slot
		Status:           "Confirmed",
	}

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
     resp,err := d.repo.CheckVideoAppoitment(patientId)
	 if err != nil {
		return "",err
	 }
	 
	 
}
