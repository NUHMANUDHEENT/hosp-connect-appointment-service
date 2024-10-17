package handler

import (
	"context"
	"fmt"
	"time"

	pb "github.com/NUHMANUDHEENT/hosp-connect-pb/proto/appointment"
	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/service"
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

	// Continue the rest of the logic
	appointment, err := h.service.ConfirmAppointment(req.PatientId, req.DoctorId, appointmentTime, 1)
	if err != nil {
		return &pb.ConfirmAppointmentResponse{
			Status:     "fail",
			Message:    err.Error(),
			StatusCode: 400,
		}, nil
	}

	return &pb.ConfirmAppointmentResponse{
		Message:    "Appointment successfully confirmed",
		StatusCode: 200,
		Status:     "success",
		PaymentUrl: appointment,
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
				PatientId:       appointment.PatientId,
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
func (d *AppoinmentServiceClient) CreateRoomForVideoTreatment(ctx context.Context, req *pb.VideoRoomRequest) (*pb.VideoRoomResponse,error){
	room, err := d.service.CreateRoomForVideoTreatment(req)
	
}