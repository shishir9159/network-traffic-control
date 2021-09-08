package dto

type KafkaMessageInput struct {
	Topic     string `json:"topic"`
	Message   string `json:"message"`
	CryptoKey string `json:"cryptoKey"`
}
