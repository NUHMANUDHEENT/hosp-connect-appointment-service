package config

import (
	"log"
	"net"

	appointmentpb "github.com/NUHMANUDHEENT/hosp-connect-pb/proto/appointment"
	doctorpb "github.com/NUHMANUDHEENT/hosp-connect-pb/proto/doctor"
	patientpb "github.com/NUHMANUDHEENT/hosp-connect-pb/proto/patient"
	paymentpb "github.com/NUHMANUDHEENT/hosp-connect-pb/proto/payment"
	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/handler"
	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/repository"
	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/service"
	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/utils"
	"github.com/nuhmanudheent/hosp-connect-appointment-service/logs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// GRPCSetup initializes the gRPC server and registers the services
func GRPCSetup(port string) (net.Listener, *grpc.Server) {
	// Create a TCP listener
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}
	logger := logs.NewLogger()
	// Initialize the database
	db := InitDatabase()

	// Initialize the appointment repository
	appointmentRepo := repository.NewAppoinmentRepository(db)

	// Initialize the gRPC client for the Doctor Service
	userconn, err := grpc.NewClient("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to doctor service: %v", err)
	}
	PaymentConn, err := grpc.NewClient("localhost:50054", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to payment service: %v", err)
	}

	doctorClient := doctorpb.NewDoctorServiceClient(userconn)
	paymentClient := paymentpb.NewPaymentServiceClient(PaymentConn)
	patientClient := patientpb.NewPatientServiceClient(userconn)

	appointmentService := service.NewAppoinmentService(appointmentRepo, doctorClient, paymentClient, patientClient, logger)

	appointmentHandler := handler.NewAppoinmentClient(appointmentService)
	go utils.StartCroneSheduler(appointmentService)
	// Create a new gRPC server
	server := grpc.NewServer()

	// Register the AppointmentService with the gRPC server
	appointmentpb.RegisterAppointmentServiceServer(server, appointmentHandler)

	// Enable server reflection (optional, useful for testing with tools like grpcurl)
	reflection.Register(server)

	log.Printf("gRPC server is running on port %s", port)
	return listener, server
}
