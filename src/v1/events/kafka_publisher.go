package events

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Shopify/sarama"
	"klovercloud/queue/config"
	"klovercloud/queue/src/v1/adapter/dto"
	"log"
	"net/http"
)

func PublishByKey(kafkaTopic string, body interface{}, header KafkaMessageHeader, publicKey string) error {
	message := KafkaMessage{
		Body:   body,
		Header: header,
	}

	err := KafkaPublisher(kafkaTopic, message, publicKey)
	if err != nil {
		return errors.New("Unable to publish data!")
	}
	return nil
}

func KafkaPublisher(kafkaTopic string, message KafkaMessage, publicKey string) error {
	//addresses of available kafka brokers
	brokers := []string{config.KafkaBroker}
	producer, err := sarama.NewSyncProducer(brokers, nil)

	if err != nil {
		fmt.Println("Could not Fetch Producer")
	}

	defer func() {
		err := producer.Close()
		if err != nil {
			fmt.Println("Could Not close the producer")
		}
	}()

	messageJson, err := json.Marshal(message)
	if err != nil {
		return err
	}
	kafkaMessageInput := dto.KafkaMessageInput{
		Topic:     kafkaTopic,
		Message:   string(messageJson),
		CryptoKey: publicKey,
	}
	buf := new(bytes.Buffer)
	err = json.NewEncoder(buf).Encode(kafkaMessageInput)
	if err != nil {
		return errors.New("Invalid Input")
	}
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, config.MessagePublisherServiceUrl+"/api/kafka/publish", buf)
	if req != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, er := client.Do(req)
	if er != nil {
		log.Print("[ERROR] Error in connecting User Message publisher Service: ", er)
		return errors.New("Something wrong with connection")
	}

	defer resp.Body.Close()

	fmt.Printf("Message is stored in topic(%s)", kafkaTopic)
	return nil
}
