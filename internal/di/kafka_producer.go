package di

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/domain"
	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	writer *kafka.Writer
}

func NewKafkaProducer(broker string) (*KafkaProducer, error) {
	// Create a new Kafka writer (producer)
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      []string{broker},    // Kafka brokers
		Topic:        "appointment_topic", // The topic to which events will be written
		Balancer:     &kafka.LeastBytes{}, // Load balancing strategy (can also use other strategies like Random)
		RequiredAcks: int(kafka.RequireOne),    // Acknowledgment setting
	})

	return &KafkaProducer{writer: writer}, nil
}

func (kp *KafkaProducer) AppointmentEvent(topic string, event domain.AppointmentEvent) error {
	// Marshal the event into JSON
	message, err := json.Marshal(event)
	if err != nil {
		log.Println("failed to marshal event:", err)
		return err
	}

	// Create a Kafka message
	msg := kafka.Message{
		Key:   []byte(strconv.Itoa(event.AppointmentId)), // Use AppointmentId as the key (helps with partitioning)
		Value: message,                     // The actual event data
	}

	// Send the message to Kafka
	err = kp.writer.WriteMessages(context.Background(), msg)
	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	log.Printf("Message delivered to topic %s\n", topic)
	return nil
}

func HandleAppointmentNotification(topic string, appevent domain.AppointmentEvent) error {
	kafkaProducer, err := NewKafkaProducer("localhost:9092") // Use your Kafka broker address
	if err != nil {
		return fmt.Errorf("failed to create Kafka producer: %w", err)
	}
	defer kafkaProducer.writer.Close()

	// Create the event that will be sent
	event := domain.AppointmentEvent{
		AppointmentId:   appevent.AppointmentId,
		Email:           appevent.Email,
		DoctorId:        appevent.DoctorId,
		AppointmentDate: appevent.AppointmentDate,
		Type:            appevent.Type,
		VideoURL:        appevent.VideoURL,
	}

	// Produce the event
	err = kafkaProducer.AppointmentEvent(topic, event)
	if err != nil {
		return fmt.Errorf("failed to produce appointment event: %w", err)
	}

	log.Println("Appointment alert event produced successfully to email:", appevent.Email)
	return nil
}
