package Status

type Status string
type Rabbitmq_cluster_deployment_status string
type Kafka_cluster_deployment_status string
type Zookeeper_entrance_deployment_status string
const (
	V = Status("V")
	D = Status("D")
	RABBITMQ_DEPLOYED= Rabbitmq_cluster_deployment_status("DEPLOYED")
	RABBITMQ_DEPLOYMENT_FAILED= Rabbitmq_cluster_deployment_status("DEPLOYMENT_FAILED")
	RABBITMQ_TERMINATED= Rabbitmq_cluster_deployment_status("DEPLOYMENT_TERMINATED")
	RABBITMQ_TERMINATION_INITIATED= Rabbitmq_cluster_deployment_status("TERMINATION_INITIATED")
	RABBITMQ_DEPLOYMENT_INITIATED=Rabbitmq_cluster_deployment_status("DEPLOYMENT_INITIATED")
	RABBITMQ_TERMINATION_FAILED=Rabbitmq_cluster_deployment_status("DEPLOYMENT_TERMINATION_FAILED")


	KAFKA_CLUSTER_DEPLOYED= Kafka_cluster_deployment_status("DEPLOYED")
	KAFKA_CLUSTER_DEPLOYMENT_FAILED= Kafka_cluster_deployment_status("DEPLOYMENT_FAILED")
	KAFKA_CLUSTER_TERMINATED= Kafka_cluster_deployment_status("DEPLOYMENT_TERMINATED")
	KAFKA_CLUSTER_TERMINATION_INITIATED= Kafka_cluster_deployment_status("TERMINATION_INITIATED")
	KAFKA_CLUSTER_DEPLOYMENT_INITIATED=Kafka_cluster_deployment_status("DEPLOYMENT_INITIATED")
	KAFKA_CLUSTER_TERMINATION_FAILED=Kafka_cluster_deployment_status("DEPLOYMENT_TERMINATION_FAILED")

	ZOOKEEEPER_DEPLOYED= Zookeeper_entrance_deployment_status("DEPLOYED")
	ZOOKEEEPER_DEPLOYMENT_FAILED= Zookeeper_entrance_deployment_status("DEPLOYMENT_FAILED")
	ZOOKEEEPER_TERMINATED= Zookeeper_entrance_deployment_status("DEPLOYMENT_TERMINATED")
	ZOOKEEEPER_TERMINATION_INITIATED= Zookeeper_entrance_deployment_status("TERMINATION_INITIATED")
	ZOOKEEEPER_DEPLOYMENT_INITIATED=Zookeeper_entrance_deployment_status("DEPLOYMENT_INITIATED")
	ZOOKEEEPER_TERMINATION_FAILED=Zookeeper_entrance_deployment_status("DEPLOYMENT_TERMINATION_FAILED")
)

var IsStatus = map[string]bool{
	"V": true,
	"D": true,
}
