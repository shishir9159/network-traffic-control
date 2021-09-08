package DockerRegistryType

type DockerRegistryType string

const (
	DOCKER_HUB = DockerRegistryType("DOCKER_HUB")
	CUSTOM     = DockerRegistryType("CUSTOM")
)

var IsDockerRegistryType = map[string]bool{
	"DOCKER_HUB": true,
	"CUSTOM":     true,
}
