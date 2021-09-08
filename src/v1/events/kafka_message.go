package events

type KafkaMessage struct {
	Body   interface{}        `json:"body"`
	Header KafkaMessageHeader `json:"header"`
}

type KafkaMessageHeader struct {
	Offset    int               `json:"offset"`
	Namespace string            `json:"namespace"`
	Command   string            `json:"command"`
	CommandId string            `json:"commandId"`
	CompanyId string            `json:"companyId"`
	Extras    map[string]string `json:"extras"`
}
