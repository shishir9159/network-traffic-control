package v1

import (
	"github.com/labstack/echo"
	"gopkg.in/mgo.v2/bson"
	"klovercloud/queue/common"
	"klovercloud/queue/src/v1/adapter/enum/Authority"
	"klovercloud/queue/src/v1/adapter/enum/Status"
	kafka "klovercloud/queue/src/v1/kafka/cluster"
	"klovercloud/queue/src/v1/kafka/manager"
	"klovercloud/queue/src/v1/logs"
	"klovercloud/queue/src/v1/rabbitmq"
	"klovercloud/queue/src/v1/utility"
	"log"
	"net/http"
)

func Routes(e *echo.Echo) {
	e.GET("/", index)

	// Health Page
	e.GET("/health", health)

	v1RabbitMQMonitor := e.Group("/api/v1/rabbitmqs")
	rabbitmq.RabbitMQRouter(v1RabbitMQMonitor)

	v1KafkaMonitor := e.Group("/api/v1/kafkas")
	kafka.KafkaClusterRouter(v1KafkaMonitor)

	v1QueueMonitor := e.Group("/api/v1/queues")
	QueueRouter(v1QueueMonitor)

	v1KafkaManagerMonitor := e.Group("/api/v1/kafka_managers")
	manager.KafkaManagerRouter(v1KafkaManagerMonitor)

	v1LogMonitor := e.Group("/api/v1/logs")
	logs.LogRouter(v1LogMonitor)

}


func QueueRouter(g *echo.Group) {

	g.GET("",Find)
}

func Find(context echo.Context) error {
	user, _, err := utility.GetHeaderData(context)
	if err != nil {
		log.Println("[ERROR] while getting header data", err.Error())
		return common.GenerateErrorResponse(context, nil, "Operation failed!")
	}
	permissionErr := utility.CheckPermission(user)
	if permissionErr != nil {
		log.Println("[ERROR] while checking permission", permissionErr)
		return common.GenerateErrorResponse(context, nil, "Permission error!")
	}

	var isSuperAdmin bool

	for _, authority := range user.Authorities {
		if authority == Authority.ROLE_SUPER_ADMIN || authority == Authority.ROLE_ADMIN {
			isSuperAdmin = true
			break
		}
	}

	vpcId := context.QueryParam("vpcId")
	region := context.QueryParam("region")
	tempRmqs := []rabbitmq.Cluster{}
	tempkafkas:= []kafka.Cluster{}
	var query bson.M
	if isSuperAdmin {
		if vpcId != "" && vpcId != "null" && region != "" && region != "null" {
			log.Println("Querying with vpcId and region", "vpcId: "+vpcId, "region:"+region)
			query = bson.M{"$and": []bson.M{{"companyId": user.CompanyId.Hex()}, {"region": region}, {"vpcId": vpcId}, {"status": Status.V}}}
		} else if region != "" && region != "null" {
			log.Println("Querying with region", "region:"+region)
			query = bson.M{"$and": []bson.M{{"companyId": user.CompanyId.Hex()}, {"region": region}, {"status": Status.V}}}
		} else if vpcId != "" && vpcId != "null" {
			log.Println("Querying with vpcId ", "vpcId: "+vpcId)
			query = bson.M{"$and": []bson.M{{"vpcId": vpcId}, {"status": Status.V}}}
		} else {
			log.Println("Querying with companyId  ", "companyId: "+user.CompanyId.Hex())
			query = bson.M{"$and": []bson.M{{"companyId": user.CompanyId.Hex()}, {"status": Status.V}}}
		}
		tempRmqs = rabbitmq.Cluster{}.FindByQuery(query)
		tempkafkas=kafka.Cluster{}.FindByQuery(query)
	} else {
		log.Println("regular user!")
		if vpcId != "" && vpcId != "null" && region != ""  {
			log.Println("Querying with vpcId and region", "vpcId: "+vpcId, "region:"+region)
			query = bson.M{"$and": []bson.M{{"region": region}, {"vpcId": vpcId}, {"status": Status.V}}}
		} else if region != "" && region != "null" {
			log.Println("Querying with region", "region:"+region)
			query = bson.M{"$and": []bson.M{{"region": region}, {"status": Status.V}}}
		} else if vpcId != "" && vpcId != "null" {
			log.Println("Querying with vpcId ", "vpcId: "+vpcId)
			query = bson.M{"$and": []bson.M{{"vpcId": vpcId}, {"status": Status.V}}}
		}
		tempRmqs = rabbitmq.Cluster{}.FindByQuery(query)
		tempkafkas=kafka.Cluster{}.FindByQuery(query)
	}
	log.Println(" teams:", user.TeamIdList)

	rmqs := []rabbitmq.Cluster{}
	kafkas:=[] kafka.Cluster{}

	if len(user.TeamIdList) > 0 {
		for _, each := range tempRmqs {
			if len(each.Teams) > 0 {
				for _, team := range each.Teams {
					for _, userTeam := range user.TeamIdList {
						if bson.ObjectIdHex(team) == userTeam {
							rmqs = append(rmqs, each)
							break
						}
					}
				}
			} else {
				rmqs = append(rmqs, each)
			}
		}
	} else {
		rmqs = tempRmqs
	}

	if len(user.TeamIdList) > 0 {
		for _, each := range tempkafkas {
			if len(each.Teams) > 0 {
				for _, team := range each.Teams {
					for _, userTeam := range user.TeamIdList {
						if bson.ObjectIdHex(team) == userTeam {
							kafkas = append(kafkas, each)
							break
						}
					}
				}
			} else {
				kafkas = append(kafkas, each)
			}
		}
	} else {
		kafkas = tempkafkas
	}

	type Queues struct {
		Kafkas [] kafka.Cluster `json:"kafkas"`
		RabbitMQs [] rabbitmq.Cluster `json:"rabbitmqs"`
	}

	data:= Queues{
		Kafkas:    kafkas,
		RabbitMQs: rmqs,
	}

	return common.GenerateSuccessResponse(context, data, "")
}

func index(c echo.Context) error {
	return c.String(http.StatusOK, "This is KloverCloud queue service")
}

func health(c echo.Context) error {
	return c.String(http.StatusOK, "I am live!")
}
