
package rabbitmq
import (
	"gopkg.in/mgo.v2/bson"
	"klovercloud/queue/src/v1/adapter/universal"
)

type RabbitMQResponseDTO struct {
	Name        string `json:"name"`
	CompanyId   string `json:"companyId"`
	CompanyName string `json:"companyName"`
	Namespace   string `json:"namespace"`
	KafkaTopic  string `json:"kafkaTopic"`
}
type CreateRabbitMQClusterDto struct {
	requester               universal.User
	actionId                *bson.ObjectId
	input                   *Cluster
	output                  interface{}
	rabbitMQClusterResponse *RabbitMQResponseDTO
	vpcId                   bson.ObjectId
	companyId               bson.ObjectId
}

func (dto CreateRabbitMQClusterDto) init(companyId string, actionId bson.ObjectId, requester universal.User, input *Cluster) CreateRabbitMQClusterDto {
	dto.companyId= bson.ObjectIdHex(companyId)
	dto.actionId=&actionId
	dto.requester=requester
	dto.input=input

	return dto

}
