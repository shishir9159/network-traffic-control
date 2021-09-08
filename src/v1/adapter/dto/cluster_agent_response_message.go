package dto

type ClusterAgentResponseMessage struct {
	AgentKafkaTopic string                   `json:"agentKafkaTopic"`
	Namespace       string                   `json:"namespace"`
	Command         string                   `json:"command"`
	CompanyId       string                   `json:"companyId"`
	Extras          map[string]string        `json:"extras"`
	TaskStatusList  []ClusterAgentTaskStatus `json:"taskStatusList"`
	KafkaOffset     int                      `json:"kafkaOffset"`
}

type ClusterAgentTaskStatus struct {
	TaskType string `json:"taskType"`
	Status   string `json:"status"`
	Message  string `json:"message"`
	Log      string `json:"log"`
}
