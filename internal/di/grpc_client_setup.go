package di

import (
	"log"
	"net"

	appointmentpb "github.com/NUHMANUDHEENT/hosp-connect-pb/proto/appointment"
	doctorpb "github.com/NUHMANUDHEENT/hosp-connect-pb/proto/doctor"
	paymentpb "github.com/NUHMANUDHEENT/hosp-connect-pb/proto/payment"
	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/config"
	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/handler"
	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/repository"
	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/service"
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

	// Initialize the database
	db := config.InitDatabase()

	// Initialize the appointment repository
	appointmentRepo := repository.NewAppoinmentRepository(db)

	// Initialize the gRPC client for the Doctor Service
	DoctorConn, err := grpc.Dial("localhost:50051", grpc.WithInsecure()) // Doctor client calling
	if err != nil {
		log.Fatalf("Failed to connect to doctor service: %v", err)
	}
	PaymentConn, err := grpc.Dial("localhost:50054", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to payment service: %v", err)
	}


	doctorClient := doctorpb.NewDoctorServiceClient(DoctorConn)
	paymentClient := paymentpb.NewPaymentServiceClient(PaymentConn)

	// Initialize the appointment service with repository and doctor client
	appointmentService := service.NewAppoinmentService(appointmentRepo, doctorClient, paymentClient)

	// Initialize the appointment handler
	appointmentHandler := handler.NewAppoinmentClient(appointmentService)

	// Create a new gRPC server
	server := grpc.NewServer()

	// Register the AppointmentService with the gRPC server
	appointmentpb.RegisterAppointmentServiceServer(server, appointmentHandler)

	// Enable server reflection (optional, useful for testing with tools like grpcurl)
	reflection.Register(server)

	log.Printf("gRPC server is running on port %s", port)
	return listener, server
}
