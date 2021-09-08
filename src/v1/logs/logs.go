package logs

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-bongo/bongo"
	"github.com/labstack/echo"
	"github.com/segmentio/kafka-go"
	"gopkg.in/mgo.v2/bson"
	"klovercloud/queue/common"
	"klovercloud/queue/config"
	"klovercloud/queue/src/v1/adapter/enum/Authority"
	"klovercloud/queue/src/v1/db"
	"klovercloud/queue/src/v1/rabbitmq"
	"klovercloud/queue/src/v1/utility"
	"log"
)

const(
	LogDtoCollection="LogDtoCollection"
)
func LogRouter(g *echo.Group) {
	logDto := LogDto{}
	g.GET("/app/:id", logDto.Find)
	g.GET("/app/:id/revision/:revision", logDto.FindByRevision)
	//g.GET("/app/{id}/revision/{revisionId}", logDto.Find)
}

func (logDto LogDto) FindByRevision(context echo.Context) error {

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

	id := context.Param("id")
	revision := context.Param("revision")
	query := bson.M{"$and": []bson.M{{"id":id },{"revision":revision}} }
	rmqs := rabbitmq.Cluster{}.FindByQuery(query)

	if len(rmqs)==0{
		return common.GenerateErrorResponse(context,"[ERROR] No resource found by id:"+id,"Operation failed!")
	}
	var isAuthorized bool
	if len(user.TeamIdList) > 0 {
		for _, each := range user.TeamIdList {
			for _, autherizedTeam := range rmqs[0].Teams {
				if each == bson.ObjectIdHex(autherizedTeam) {
					isAuthorized = true
					break
				}
			}

		}
	} else {
		isAuthorized = true
	}
	if isAuthorized || isSuperAdmin{
		logDtos:=LogDto{
			AppId: id,
		}.find()

		if len(logDtos)>0{
			return common.GenerateSuccessResponse(context,logDtos[0],"")
		}
		return common.GenerateSuccessResponse(context,LogDto{},"")
	}
	return common.GenerateErrorResponse(context,"[ERROR]: Unauthorized!","Operation failed!")

}



func (logDto LogDto) Find(context echo.Context) error {
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

	id := context.Param("id")
	query := bson.M{"$and": []bson.M{{"id":id }}}
	rmqs := rabbitmq.Cluster{}.FindByQuery(query)

	if len(rmqs)==0{
		return common.GenerateErrorResponse(context,"[ERROR] No resource found by id:"+id,"Operation failed!")
	}
	var isAuthorized bool
	if len(user.TeamIdList) > 0 {
		for _, each := range user.TeamIdList {
			for _, autherizedTeam := range rmqs[0].Teams {
				if each == bson.ObjectIdHex(autherizedTeam) {
					isAuthorized = true
					break
				}
			}

		}
	} else {
		isAuthorized = true
	}

	if isAuthorized || isSuperAdmin{
		logDtos:=LogDto{
			AppId: id,
		}.find()
		return common.GenerateSuccessResponse(context,logDtos,"")
	}
return common.GenerateErrorResponse(context,"[ERROR]: Unauthorized!","Operation failed!")

}
type LogDto struct {
	bongo.DocumentBase `json:"_" bson:",inline"`
	AppId  string     `json:"appId" bson:"appId"`
	Revision int `json:"revision" bson:"revision"`
	Log    [] string `json:"log" bson:"log"`
}

type LogResponseDTO struct {
	Logs []string `json:"logs"`
}

func (logDto LogDto) find() []LogDto {
	logs := []LogDto{}
	query := bson.M{
		"$and": []bson.M{
			{"appId": logDto.AppId},
		},
	}
	coll := db.GetDmManager().Db.Collection(LogDtoCollection)
	curser, _ := coll.Find(db.GetDmManager().Ctx, query)
	for curser.Next(context.TODO()) {
		elemValue := new(LogDto)
		err := curser.Decode(elemValue)
		if err != nil {
			log.Println("[ERROR]", err)
			break
		}
		logs = append(logs, *elemValue)
	}

	return logs

}

func (logDto LogDto) save() error {
	coll := db.GetDmManager().Db.Collection(LogDtoCollection)
	_, err := coll.InsertOne(db.GetDmManager().Ctx, logDto)
	if err != nil {
		log.Println("[ERROR] Insert document:", err.Error())
		return err
	}
	return nil
}

func (logDto LogDto) findByAppId() LogDto {
	query := bson.M{
		"$and": []bson.M{
			{"appId": logDto.AppId},
		},
	}
	temp := new(LogDto)
	coll := db.GetDmManager().Db.Collection(LogDtoCollection)
	result := coll.FindOne(db.GetDmManager().Ctx, query)

	err := result.Decode(temp)
	if err != nil {
		log.Println("[ERROR]", err)
	}
	return *temp
}

func (logDto LogDto) Update() error{
	query := bson.M{
		"$and": []bson.M{
			{"appId": logDto.AppId},
			{"appId": logDto.AppId},

		},
	}
	coll := db.GetDmManager().Db.Collection(LogDtoCollection)
	_, err:=coll.DeleteOne(db.GetDmManager().Ctx,query)
	if err != nil {
		return err
	}
	err=logDto.save()
	if err != nil {
		return errors.New("Failed to persist!")
	}
	return nil
}




func ConsumeLog() {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{config.KafkaBroker},
		GroupID:  config.QueueDeploymentEventConsumerGroup,
		Topic:    config.TopicQueueDeploymentEvent,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})
	for {
		log.Println("consuming events log ... ")
		var consumedData KafkaStruct
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Println(err.Error())
			continue
		}
		err = json.Unmarshal([]byte(string(m.Value)), &consumedData)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		data:=LogDto{
			AppId: consumedData.Body.AppId,
		}.findByAppId()

		if data.AppId==""{
			log.Println("New build ... ")
			dto:= LogDto{
				AppId:        consumedData.Body.AppId,
				Revision:     0,
			}
			dto.Log=append(dto.Log,consumedData.Body.Log )
			dto.save()
		}else{
			if consumedData.Body.Order==0{
				log.Println("Old app, new build ... ")
				data.Revision=data.Revision+1
				data.Log=append(data.Log,consumedData.Body.Log )
				data.Update()
				log.Println("Event persisted ... ")
			}else{
				data.Log=append(data.Log,consumedData.Body.Log )
				data.Update()
				log.Println("Event persisted ... ")
			}

		}

	}
	r.Close()
}