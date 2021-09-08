package utility



type KubeCluster struct {
	Name          string             `json:"name"`
	DefaultDomain string             `json:"defaultDomain"`
	KafkaTopic    string             `json:"kafkaTopic"`
	AgentInfo     KubeAgentCryptoKey `json:"agentInfo"`
}

type KubeAgentCryptoKey struct {
	PublicKey string `json:"publicKey"`
	KafkaTopic string `json:"kafkaTopic"`
}


type VPCResponseDtoForManagement struct {
	Namespace string  `json:"namespace"`
	AvailableCPU int  `json:"availableCPU"`
	AvailableMemory int  `json:"availableMemory"`
	AvailableStorage int  `json:"availableStorage"`
	TeamList [] Team `json:"teamlist"`
	Organization Organization `json:"organization"`
	CompanyId string `json:"companyId"`
	Name string  `json:"name"`
}

type Team struct {
	CompanyId string `json:"companyId"`
	Id string `json:"id"`
	Name string  `json:"name"`
}

type Organization struct {
	Id string `json:"id"`
	Company Company `json:"company"`
}

type Company struct {
	Id string `json:"id"`
}