package handler

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/NUHMANUDHEENT/hosp-connect-pb/proto/appointment"
	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/domain"
	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/service"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AppoinmentServiceClient struct {
	pb.UnimplementedAppointmentServiceServer
	service service.AppointmentService
}

func NewAppoinmentClient(service service.AppointmentService) *AppoinmentServiceClient {
	return &AppoinmentServiceClient{
		service: service,
	}
}

// Handler function for checking availability
func (a *AppoinmentServiceClient) CheckAvailability(ctx context.Context, req *pb.GetAvailabilityRequest) (*pb.GetAvailabilityResponse, error) {
	// Convert Protobuf Timestamp to Go's time.Time
	requestedTime := req.RequestedDateTime.AsTime()
	fmt.Println("category", req.CategoryId, req.RequestedDateTime)
	resp, err := a.service.CheckAvailability(req.CategoryId, requestedTime)
	if err != nil {
		return nil, err
	}

	availabilityResponse := &pb.GetAvailabilityResponse{
		AvailableSlots: []*pb.AvailabilitySlot{},
	}

	for _, v := range resp {
		availabilityResponse.AvailableSlots = append(availabilityResponse.AvailableSlots, &pb.AvailabilitySlot{
			DoctorId:   v.DoctorId,
			DoctorName: v.DoctorName,
		})
	}
	// fmt.Println(availabilityResponse)
	return availabilityResponse, nil
}
func (h *AppoinmentServiceClient) CheckAvailabilityByDoctorId(ctx context.Context, req *pb.CheckAvailabilityByDoctorIdRequest) (*pb.CheckAvailabilityByDoctorIdResponse, error) {
	// Call the appointment service to check doctor's availability

	available, err := h.service.CheckAvailabilityByDoctorId(req.DoctorId)
	if err != nil {
		return &pb.CheckAvailabilityByDoctorIdResponse{
			Status: "error",
		}, nil
	}
	fmt.Println("availability======", available)
	// Return the gRPC response
	return &pb.CheckAvailabilityByDoctorIdResponse{
		Status:             "available",
		DoctorId:           available.DoctorId,
		DoctorAvailability: available.DoctorAvailability,
	}, nil
}
func (h *AppoinmentServiceClient) ConfirmAppointment(ctx context.Context, req *pb.ConfirmAppointmentRequest) (*pb.ConfirmAppointmentResponse, error) {
	// Convert the protobuf Timestamp to Go's time.Time
	appointmentTime := req.GetConfirmedDateTime().AsTime()

	// Log the received time to ensure it's correct
	fmt.Println("Received appointment time:", appointmentTime, "for doctor:", req.DoctorId)
	fmt.Println("type", req.Type)
	appointment := domain.Appointment{
		DoctorId:         req.DoctorId,
		PatientId:        req.PatientId,
		AppointmentTime:  appointmentTime,
		SpecializationId: req.SpecializationId,
		Type:             req.Type,
	}
	// Continue the rest of the logic
	url, message, err := h.service.ConfirmAppointment(appointment)
	if err != nil {
		return &pb.ConfirmAppointmentResponse{
			Status:     "fail",
			Message:    err.Error(),
			StatusCode: 400,
		}, nil
	}

	return &pb.ConfirmAppointmentResponse{
		Message:    message,
		StatusCode: 200,
		Status:     "success",
		PaymentUrl: url,
	}, nil
}
func (h *AppoinmentServiceClient) GetUpcomingAppointments(ctx context.Context, req *pb.GetAppointmentsRequest) (*pb.GetAppointmentsResponse, error) {
	appointments, err := h.service.GetUpcomingAppointments(req.PatientId)
	if err != nil {
		return &pb.GetAppointmentsResponse{
			StatusCode: 400,
			Status:     "fail",
		}, nil
	}
	currentTime := time.Now()
	var upcomingAppointments []*pb.Appointment
	for _, appointment := range appointments {
		if appointment.AppointmentTime.After(currentTime) {
			upcomingAppointments = append(upcomingAppointments, &pb.Appointment{
				AppointmentId:   int64(appointment.AppointmentId),
				AppointmentType: appointment.Type,
				DoctorId:        appointment.DoctorId,
				AppointmentTime: appointment.AppointmentTime.Format(time.ANSIC),
				Specialization:  int64(appointment.SpecializationId),
			})
		}
	}
	return &pb.GetAppointmentsResponse{
		StatusCode:   200,
		Status:       "success",
		Appointments: upcomingAppointments,
	}, nil
}
func (d *AppoinmentServiceClient) CreateRoomForVideoTreatment(ctx context.Context, req *pb.VideoRoomRequest) (*pb.VideoRoomResponse, error) {
	room, err := d.service.CreateRoomForVideoTreatment(req.PatientId, req.DoctorId, req.SpecializationId)
	if err != nil {
		return &pb.VideoRoomResponse{
			StatusCode: "400",
			Status:     "fail",
			Message:    err.Error(),
		}, nil
	}
	return &pb.VideoRoomResponse{
		StatusCode: "200",
		Status:     "success",
		RoomUrl:    room,
	}, nil
}
func (d *AppoinmentServiceClient) GetAppointmentDetails(ctx context.Context, req *pb.GetAppointmentDetailsRequest) (*pb.GetAppointmentDetailsResponse, error) {
	appointment, err := d.service.GetAppointmentDetails(req.OrderId)
	if err != nil {
		return &pb.GetAppointmentDetailsResponse{
			StatusCode: 400,
			Status:     "fail",
			Message:    err.Error(),
		}, nil
	}
	return &pb.GetAppointmentDetailsResponse{
		StatusCode:       200,
		Status:           "success",
		AppointmentId:    int64(appointment.AppointmentId),
		AppointmentType:  appointment.Type,
		DoctorId:         appointment.DoctorId,
		AppointmentTime:  timestamppb.New(appointment.AppointmentTime),
		SpecializationId: int64(appointment.SpecializationId),
	}, nil
}
func (a *AppoinmentServiceClient) AddSpecialization(ctx context.Context, req *pb.AddSpecializationRequest) (*pb.StandardResponse, error) {
	log.Println("Adding specialization with name: ", req.Name)
	resp, err := a.service.AddSpecialization(req.Name, req.Description)
	if err != nil {
		return &pb.StandardResponse{
			Status:     "fail",
			Error:      err.Error(),
			StatusCode: 400,
		}, nil
	}
	return &pb.StandardResponse{
		Status:     "success",
		Message:    resp,
		StatusCode: 200,
	}, nil
}
func (a *AppoinmentServiceClient) FetchStatisticsDetails(ctx context.Context, req *pb.StatisticsRequest) (*pb.StatisticsResponse, error) {
	special, statics, err := a.service.FetchStatisticsDetails(req.Param)
	if err != nil {
		return &pb.StatisticsResponse{}, err
	}
	fmt.Println("speccc", special)
	var specializ []*pb.SpecializationStats
	for _, s := range special {
		specializ = append(specializ, &pb.SpecializationStats{
			AppointmentCount:   int32(s.Count),
			SpecializationName: s.Name,
		})
	}
	return &pb.StatisticsResponse{
		TotalPatients:       int32(statics.TotalPatients),
		TotalDoctors:        int32(statics.TotalDoctors),
		TotalAppointments:   int32(statics.TotalAppointments),
		TotalRevenue:        float32(statics.TotalRevenue),
		SpecializationStats: specializ,
	}, nil
}
func (a *AppoinmentServiceClient) CancelAppointment(ctx context.Context, req *pb.CancelAppointmentRequest) (*pb.CancelAppointmentResponse, error) {
	resp, err := a.service.CancelAppointment(domain.Appointment{AppointmentId: int(req.AppointmentId), PatientId: req.PatientId}, req.Reason)
	if err != nil {
		return &pb.CancelAppointmentResponse{
			Status:     "fail",
			StatusCode: "400",
			Message:    resp,
		}, nil
	}
	return &pb.CancelAppointmentResponse{
		Status:     "succes",
		StatusCode: "200",
		Message:    resp,
	}, nil
}
