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
	"github.com/sirupsen/logrus"

	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/di"
	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/domain"
	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/repository"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AppointmentService interface {
	CheckAvailability(CategoryId int32, reqtime time.Time) ([]domain.Availability, error)
	CheckAvailabilityByDoctorId(doctorID string) (*appointment.CheckAvailabilityByDoctorIdResponse, error)
	ConfirmAppointment(appointment domain.Appointment) (string, string, error)
	CancelAppointment(appointment domain.Appointment, reason string) (string, error)
	CreateRoomForVideoTreatment(patientId, doctorId string, specializationId int64) (string, error)
	GetUpcomingAppointments(patientId string) ([]domain.Appointment, error)
	GetAppointmentDetails(orderid string) (domain.Appointment, error)
	SendDialyReminders()
	AddSpecialization(name, Description string) (string, error)
	FetchStatisticsDetails(param string) ([]domain.SpecializationStats, domain.StatisticsData, error)
}

type appointmentService struct {
	repo          repository.AppointmentRepository
	DoctorClient  doctorpb.DoctorServiceClient
	PaymentClient paymentpb.PaymentServiceClient
	PatientClient patientpb.PatientServiceClient
	Logger        *logrus.Logger
}

func NewAppoinmentService(repo repository.AppointmentRepository, DoctorClient doctorpb.DoctorServiceClient, paymentClient paymentpb.PaymentServiceClient, patientClient patientpb.PatientServiceClient, logger *logrus.Logger) AppointmentService {
	return &appointmentService{
		repo:          repo,
		DoctorClient:  DoctorClient,
		PaymentClient: paymentClient,
		PatientClient: patientClient,
		Logger:        logger,
	}
}

func (a *appointmentService) CheckAvailability(CategoryId int32, reqtime time.Time) ([]domain.Availability, error) {
	a.Logger.WithFields(logrus.Fields{
		"Function":   "CheckAvailability",
		"CategoryId": CategoryId,
	}).Info("Checking availability for category")

	availability := []domain.Availability{}
	reqTimestamp := timestamppb.New(reqtime)

	resp, err := a.DoctorClient.GetAvailability(context.Background(), &doctorpb.GetAvailabilityRequest{
		CategoryId:        CategoryId,
		RequestedDateTime: reqTimestamp,
	})
	if err != nil {
		a.Logger.WithFields(logrus.Fields{
			"Function":   "CheckAvailability",
			"CategoryId": CategoryId,
			"Error":      err,
		}).Error("Failed to get availability")
		return availability, err
	}

	for _, slot := range resp.AvailableSlots {
		availability = append(availability, domain.Availability{
			DoctorId:   slot.DoctorId,
			DoctorName: slot.DoctorName,
		})
	}

	a.Logger.WithFields(logrus.Fields{
		"Function":   "CheckAvailability",
		"CategoryId": CategoryId,
	}).Info("Availability checked successfully")

	return availability, nil
}

func (s *appointmentService) CheckAvailabilityByDoctorId(doctorID string) (*appointment.CheckAvailabilityByDoctorIdResponse, error) {
	s.Logger.WithFields(logrus.Fields{
		"Function": "CheckAvailabilityByDoctorId",
		"DoctorId": doctorID,
	}).Info("Checking availability by doctor ID")

	available, err := s.DoctorClient.CheckAvailabilityByDoctorId(context.Background(), &doctorpb.CheckAvailabilityByDoctorIdRequest{
		DoctorId: doctorID,
	})
	if err != nil {
		s.Logger.WithFields(logrus.Fields{
			"Function": "CheckAvailabilityByDoctorId",
			"DoctorId": doctorID,
			"Error":    err,
		}).Error("Failed to check doctor availability")
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

	s.Logger.WithFields(logrus.Fields{
		"Function": "CheckAvailabilityByDoctorId",
		"DoctorId": doctorID,
	}).Info("Doctor availability checked successfully")

	return availability, nil
}

func (s *appointmentService) ConfirmAppointment(appointment domain.Appointment) (string, string, error) {
	s.Logger.WithFields(logrus.Fields{
		"Function":        "ConfirmAppointment",
		"AppointmentID":   appointment.AppointmentId,
		"DoctorID":        appointment.DoctorId,
		"PatientID":       appointment.PatientId,
		"AppointmentTime": appointment.AppointmentTime,
	}).Info("Starting appointment confirmation")

	available, err := s.DoctorClient.CheckAvailabilityByDoctorId(context.Background(), &doctorpb.CheckAvailabilityByDoctorIdRequest{
		DoctorId: appointment.DoctorId,
	})
	if err != nil {
		s.Logger.WithFields(logrus.Fields{
			"Function": "ConfirmAppointment",
			"DoctorID": appointment.DoctorId,
			"Error":    err,
		}).Error("Failed to call doctor service")
		return "", "failed to call doctor service", err
	}

	for _, v := range available.DoctorAvailability {
		if v.IsAvailable == "unavailable" {
			doctorUnavailableDate, err := time.Parse("Mon Jan 2 15:04:05 2006", v.DateTime)
			if err != nil {
				return "", "", errors.New("invalid doctor availability date format")
			}

			if appointment.AppointmentTime.Year() == doctorUnavailableDate.Year() &&
				appointment.AppointmentTime.Month() == doctorUnavailableDate.Month() &&
				appointment.AppointmentTime.Day() == doctorUnavailableDate.Day() {
				return "", "", errors.New("doctor is not available on this date")
			}
		}
	}

	isAvailable, url, message, err := s.repo.IsDoctorAvailable(appointment.DoctorId, appointment.PatientId, appointment.AppointmentTime, time.Hour)
	if err != nil {
		s.Logger.WithFields(logrus.Fields{
			"Function": "ConfirmAppointment",
			"DoctorID": appointment.DoctorId,
			"Error":    err,
		}).Error("Error in checking doctor availability from repository")
		return "", "", err
	}
	if !isAvailable {
		s.Logger.WithFields(logrus.Fields{
			"Function": "ConfirmAppointment",
			"Message":  message,
		}).Info("Doctor is not available at requested time")
		return url, message, nil
	}

	latestAppointmentId, err := s.repo.GetLatestAppointmentId()
	if err != nil {
		s.Logger.WithFields(logrus.Fields{
			"Function": "ConfirmAppointment",
			"Error":    err,
		}).Error("Failed to fetch latest appointment ID")
		return "", "", errors.New("failed to fetch latest appointment ID")
	}
	newAppointmentId := latestAppointmentId + 1
	appointment.AppointmentId = newAppointmentId
	appointment.Duration = time.Hour
	appointment.Status = "Pending"

	Resp, err := s.PaymentClient.CreateRazorOrderId(context.Background(), &paymentpb.CreateRazorOrderIdRequest{
		PatientId:     appointment.PatientId,
		Amount:        200,
		AppointmentId: int64(newAppointmentId),
		Type:          "appointment fee",
	})
	if err != nil {
		s.Logger.WithFields(logrus.Fields{
			"Function": "ConfirmAppointment",
			"Error":    err,
		}).Error("Failed to call payment service")
		return "", "", errors.New("failed to call payment service")
	} else if Resp.Status != "success" {
		s.Logger.WithFields(logrus.Fields{
			"Function": "ConfirmAppointment",
			"Message":  Resp.Message,
		}).Info("Payment service responded with failure")
		return "", "", errors.New(Resp.Message)
	}
	appointment.PaymentId = Resp.OrderId

	if err := s.repo.ConfirmAppointment(appointment); err != nil {
		s.Logger.WithFields(logrus.Fields{
			"Function":      "ConfirmAppointment",
			"AppointmentID": newAppointmentId,
			"Error":         err,
		}).Error("Failed to save appointment")
		return "", "", err
	}

	s.Logger.WithFields(logrus.Fields{
		"Function":      "ConfirmAppointment",
		"AppointmentID": newAppointmentId,
		"PaymentURL":    Resp.PaymentUrl,
	}).Info("Appointment confirmed successfully")

	return Resp.PaymentUrl, "Appointment successfully confirmed", nil
}

// Cancel an appointment
func (s *appointmentService) CancelAppointment(appointment domain.Appointment, reason string) (string, error) {
	s.Logger.WithFields(logrus.Fields{
		"Function":      "CancelAppointment",
		"AppointmentId": appointment.AppointmentId,
		"Reason":        reason,
	}).Info("Attempting to cancel appointment")

	resp, err := s.repo.CancelAppointment(appointment, reason)
	if err != nil {
		s.Logger.WithError(err).Error("Failed to cancel appointment")
		return "", err
	}
	s.Logger.Info("Appointment cancelled successfully")
	return resp, nil
}

// Get upcoming appointments for a patient
func (s *appointmentService) GetUpcomingAppointments(patientId string) ([]domain.Appointment, error) {
	s.Logger.WithFields(logrus.Fields{
		"Function":  "GetUpcomingAppointments",
		"PatientId": patientId,
	}).Info("Fetching upcoming appointments for patient")

	appointments, err := s.repo.FetchAppointmentsByPatient(patientId)
	if err != nil {
		s.Logger.WithError(err).Error("Failed to fetch upcoming appointments")
		return nil, err
	}

	s.Logger.Info("Upcoming appointments fetched successfully")
	return appointments, nil
}

// Create a room for video treatment
func (d *appointmentService) CreateRoomForVideoTreatment(patientId, doctorId string, specializationId int64) (string, error) {
	d.Logger.WithFields(logrus.Fields{
		"Function":  "CreateRoomForVideoTreatment",
		"PatientId": patientId,
		"DoctorId":  doctorId,
	}).Info("Creating video treatment room")

	check, resp, err := d.repo.CheckVideoAppoitment(patientId)
	if err != nil {
		d.Logger.WithError(err).Error("Failed to check video appointment")
		return "", err
	}
	if !check {
		return "", errors.New("patient doesn't have an appointment")
	}

	roomId := uuid.New().String()
	err = d.repo.SaveVideoAppointment(roomId, resp.AppointmentId, int(specializationId))
	if err != nil {
		d.Logger.WithError(err).Error("Failed to save video appointment")
		return "", err
	}

	roomURL := fmt.Sprintf("http://localhost:8080/api/v1/doctor/video-call/%s", roomId)
	profile, err := d.PatientClient.GetProfile(context.Background(), &patientpb.GetProfileRequest{PatientId: patientId})
	if err != nil {
		d.Logger.WithError(err).Error("Failed to fetch patient profile")
		return "", err
	}

	PatientRoomUrl := fmt.Sprintf("http://localhost:8080/api/v1/patient/video-call/%s", roomId)
	err = di.HandleAppointmentNotification("appointment_topic", domain.AppointmentEvent{
		AppointmentId:   resp.AppointmentId,
		Email:           profile.Email,
		VideoURL:        PatientRoomUrl,
		DoctorId:        doctorId,
		AppointmentDate: resp.AppointmentTime.Format("2016-02-01"),
		Type:            resp.Type,
	})
	if err != nil {
		d.Logger.WithError(err).Error("Failed to produce video appointment event")
		return "", errors.New("failed to produce video appointment event")
	}

	d.Logger.Info("Video treatment room created successfully")
	return roomURL, nil
}

// Get details of an appointment
func (d *appointmentService) GetAppointmentDetails(orderid string) (domain.Appointment, error) {
	d.Logger.WithFields(logrus.Fields{
		"Function": "GetAppointmentDetails",
		"OrderId":  orderid,
	}).Info("Fetching appointment details")

	appointment, err := d.repo.GetAppointmentDetails(orderid)
	if err != nil {
		d.Logger.WithError(err).Error("Failed to fetch appointment details")
		return domain.Appointment{}, err
	}

	d.Logger.Info("Appointment details fetched successfully")
	return appointment, nil
}

// Send daily reminders
func (d *appointmentService) SendDialyReminders() {
	d.Logger.Info("Sending daily appointment alerts")

	appointments, err := d.repo.FetchTodayAppointments()
	if err != nil {
		d.Logger.WithError(err).Error("Error fetching today's appointments")
		return
	}

	for _, appointment := range appointments {
		profile, err := d.PatientClient.GetProfile(context.Background(), &patientpb.GetProfileRequest{
			PatientId: appointment.PatientId,
		})
		if err != nil {
			d.Logger.WithError(err).Warn("Failed to fetch patient profile, skipping reminder")
			continue
		}

		err = di.HandleAppointmentNotification("alert_topic", domain.AppointmentEvent{
			AppointmentId:   appointment.AppointmentId,
			Email:           profile.Email,
			DoctorId:        appointment.DoctorId,
			AppointmentDate: appointment.AppointmentTime.Format("2016-02-01"),
			Type:            appointment.Type,
		})
		if err != nil {
			d.Logger.WithError(err).Warn("Failed to send daily reminder")
			continue
		}
	}

	d.Logger.Info("Daily appointment alerts sent")
}

// Add a new specialization
func (a *appointmentService) AddSpecialization(name, description string) (string, error) {
	a.Logger.WithFields(logrus.Fields{
		"Function": "AddSpecialization",
		"Name":     name,
	}).Info("Adding a new specialization")

	resp, err := a.repo.CreateSpecialization(domain.Specialization{Name: name, Description: description})
	if err != nil {
		a.Logger.WithError(err).Error("Failed to add specialization")
		return resp, err
	}

	a.Logger.Info("Specialization added successfully")
	return resp, nil
}

// Fetch statistics details
func (a *appointmentService) FetchStatisticsDetails(param string) ([]domain.SpecializationStats, domain.StatisticsData, error) {
	a.Logger.WithFields(logrus.Fields{
		"Function": "FetchStatisticsDetails",
		"Param":    param,
	}).Info("Fetching statistics details")

	special, err := a.repo.GetSpecializationStats(param)
	if err != nil {
		a.Logger.WithError(err).Error("Failed to fetch specialization stats")
		return nil, domain.StatisticsData{}, err
	}

	appointmentCount, err := a.repo.GetTotalAppointment(param)
	if err != nil {
		a.Logger.WithError(err).Error("Failed to fetch total appointment count")
		return nil, domain.StatisticsData{}, err
	}

	patientCount, err := a.PatientClient.GetTotalPatient(context.Background(), &patientpb.GetTotalPatientCountRequest{})
	if err != nil {
		a.Logger.WithError(err).Error("Failed to fetch total patient count")
		return nil, domain.StatisticsData{}, err
	}

	doctorCount, err := a.DoctorClient.GetTotalDoctor(context.Background(), &doctorpb.GetTotalDoctorCountRequest{Param: param})
	if err != nil {
		a.Logger.WithError(err).Error("Failed to fetch total doctor count")
		return nil, domain.StatisticsData{}, err
	}

	revenue, err := a.PaymentClient.GetTotalRevenue(context.Background(), &paymentpb.GetTotalRevenueRequest{Param: param})
	if err != nil {
		a.Logger.WithError(err).Error("Failed to fetch total revenue")
		return nil, domain.StatisticsData{}, err
	}

	a.Logger.Info("Statistics details fetched successfully")
	return special, domain.StatisticsData{
		TotalAppointments: appointmentCount,
		TotalPatients:     int(patientCount.PatientCount),
		TotalDoctors:      int(doctorCount.DoctorCount),
		TotalRevenue:      revenue.TotalRevenue,
	}, nil
}
