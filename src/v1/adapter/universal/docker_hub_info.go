package universal

import (
	"klovercloud/queue/src/v1/adapter/enum/DockerRegistryType"
)

type DockerHubInfo struct {
	DockerRegistryType   DockerRegistryType.DockerRegistryType `json:"dockerRegistryType"`
	DockerRegistryServer string                                `json:"dockerRegistryServer"`
	DockerhubId          string                                `json:"dockerhubId"`
	DockerhubPassword    string                                `json:"dockerhubPassword"`
	DockerhubEmail       string                                `json:"dockerhubEmail"`
	DockerConfigData     string                                `json:"dockerConfigData"`
}
