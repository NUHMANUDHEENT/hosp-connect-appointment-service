package config

import (
	"log"
	"net"
	"os"

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

	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}
	logger := logs.NewLogger()
	db := InitDatabase()

	appointmentRepo := repository.NewAppoinmentRepository(db)

	userconn, err := grpc.NewClient(os.Getenv("USER_GRPC_SERVER"), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to doctor service: %v", err)
	}
	PaymentConn, err := grpc.NewClient(os.Getenv("PAYMENT_GRPC_SERVER"), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to payment service: %v", err)
	}

	doctorClient := doctorpb.NewDoctorServiceClient(userconn)
	paymentClient := paymentpb.NewPaymentServiceClient(PaymentConn)
	patientClient := patientpb.NewPatientServiceClient(userconn)

	appointmentService := service.NewAppoinmentService(appointmentRepo, doctorClient, paymentClient, patientClient, logger)

	appointmentHandler := handler.NewAppoinmentClient(appointmentService)
	go utils.StartCroneSheduler(appointmentService)
	
	server := grpc.NewServer()

	appointmentpb.RegisterAppointmentServiceServer(server, appointmentHandler)

	reflection.Register(server)

	log.Printf("============== gRPC server is running on port %s ===============", port)
	return listener, server
}
