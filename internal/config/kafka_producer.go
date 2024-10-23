package config

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/nuhmanudheent/hosp-connect-appointment-service/internal/domain"
)

type KafkaProducer struct {
	Producer *kafka.Producer
}

func NewKafkaProducer(broker string) (*KafkaProducer, error) {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": broker})
	if err != nil {
		return nil, err
	}
	return &KafkaProducer{Producer: producer}, nil
}
func (kp *KafkaProducer) AppointmentEvent(topic string, event domain.AppointmentEvent) error {
	message, err := json.Marshal(event)
	if err != nil {
		fmt.Errorf("failed to marshal event: %w", err)
	}
	deliveryChan := make(chan kafka.Event)
	err = kp.Producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          message,
	}, deliveryChan)
	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}
	e := <-deliveryChan
	msg := (e.(*kafka.Message))
	if msg.TopicPartition.Error != nil {
		return fmt.Errorf("delivery failed: %w", msg.TopicPartition.Error)
	}

	log.Printf("Message delivered to topic %s [%d] at offset %v\n", *msg.TopicPartition.Topic, msg.TopicPartition.Partition, msg.TopicPartition.Offset)
	close(deliveryChan)

	return nil
}
func HandleAppointmentNotification(appevent domain.AppointmentEvent) error {
	kafkaProducer, err := NewKafkaProducer("localhost:9092") // Use your Kafka broker address
	if err != nil {
		return fmt.Errorf("failed to create Kafka producer: %w", err)
	}
	defer kafkaProducer.Producer.Close()
	event := domain.AppointmentEvent{
		AppointmentId:   appevent.AppointmentId,
		Email:           appevent.Email,
		DoctorId:        appevent.DoctorId,
		AppointmentDate: appevent.AppointmentDate,
		Type:            appevent.Type,
		VideoURL:        appevent.VideoURL,
	}
	err = kafkaProducer.AppointmentEvent("appointment_topic", event)
	if err != nil {
		return fmt.Errorf("failed to produce appointment event: %w", err)
	}
	log.Println("Appointment event produced successfully")
	return nil

}
