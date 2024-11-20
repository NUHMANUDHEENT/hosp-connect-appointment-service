package di

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/domain"
	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	writer *kafka.Writer
}

func NewKafkaProducer(broker string) (*KafkaProducer, error) {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      []string{broker},   
		Topic:        "appointment_topic", 
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: int(kafka.RequireOne),  
	})

	return &KafkaProducer{writer: writer}, nil
}

func (kp *KafkaProducer) AppointmentEvent(topic string, event domain.AppointmentEvent) error {
	message, err := json.Marshal(event)
	if err != nil {
		log.Println("failed to marshal event:", err)
		return err
	}

	msg := kafka.Message{
		Key:   []byte(strconv.Itoa(event.AppointmentId)), 
		Value: message,      
	}

	err = kp.writer.WriteMessages(context.Background(), msg)
	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	log.Printf("Message delivered to topic %s\n", topic)
	return nil
}

func HandleAppointmentNotification(topic string, appevent domain.AppointmentEvent) error {
	kafkaProducer, err := NewKafkaProducer(os.Getenv("KAFKA_BROKER")) 
	if err != nil {
		return fmt.Errorf("failed to create Kafka producer: %w", err)
	}
	defer kafkaProducer.writer.Close()

	event := domain.AppointmentEvent{
		AppointmentId:   appevent.AppointmentId,
		Email:           appevent.Email,
		DoctorId:        appevent.DoctorId,
		AppointmentDate: appevent.AppointmentDate,
		Type:            appevent.Type,
		VideoURL:        appevent.VideoURL,
	}

	err = kafkaProducer.AppointmentEvent(topic, event)
	if err != nil {
		return fmt.Errorf("failed to produce appointment event: %w", err)
	}

	log.Println("Appointment alert event produced successfully to email:", appevent.Email)
	return nil
}
