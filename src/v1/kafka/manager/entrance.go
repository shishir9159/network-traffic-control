package manager

import (
	"context"
	"errors"
	"github.com/go-bongo/bongo"
	"github.com/labstack/echo"
	"github.com/twinj/uuid"
	"klovercloud/queue/common"
	"klovercloud/queue/encryption"
	"klovercloud/queue/src/v1/adapter/enum/AgentCommand"
	"klovercloud/queue/src/v1/adapter/enum/Authority"
	"klovercloud/queue/src/v1/adapter/enum/Status"
	"klovercloud/queue/src/v1/adapter/universal"
	"klovercloud/queue/src/v1/db"
	"klovercloud/queue/src/v1/events"
	"klovercloud/queue/src/v1/utility"
	"klovercloud/queue/src/v1/validator"
	"gopkg.in/mgo.v2/bson"
	"log"
	"os"
	"strconv"
)

const (
	ZookeeperEntranceCollection       = "zookeeper_entrance"
)

type Entrance struct {
	bongo.DocumentBase `bson:",inline"`
	VpcId              string `bson:"vpcId" json:"vpcId" required:"true"`
	TeamId             string `bson:"teamId" json:"teamId" required:"true" `
	CompanyId          string `bson:"companyId" json:"companyId" required:"true"`
	Namespace string `bson:"namespace" json:"namespace"`
	Replicas  int `bson:"replicas" json:"replicas" required:"true"`
	Resources struct {
		Requests struct {
			CPU    int   `bson:"cpu" json:"cpu"`
			Memory int `bson:"memory" json:"memory"`
		} `bson:"requests" json:"requests"`
		Limits struct {
			CPU    int   `bson:"cpu" json:"cpu" required:"true"`
			Memory int `bson:"memory" json:"memory" required:"true"`
		} `bson:"limits" json:"limits" required:"true" type:"struct"`
	} `bson:"resources" json:"resources" required:"true" type:"struct"`

	Manager struct{
		Resources struct {
			Requests struct {
				CPU    int   `bson:"cpu" json:"cpu"`
				Memory int `bson:"memory" json:"memory"`
			} `bson:"requests" json:"requests"`
			Limits struct {
				CPU    int   `bson:"cpu" json:"cpu" required:"true"`
				Memory int `bson:"memory" json:"memory" required:"true"`
			} `bson:"limits" json:"limits" required:"true" type:"struct"`
		} `bson:"resources" json:"resources" required:"true" type:"struct"`
	}`bson:"manager" json:"manager"`
	Status           Status.Status                          `bson:"status" json:"status"`
	DeploymentStatus Status.Zookeeper_entrance_deployment_status `bson:"deployment_status" json:"deployment_status"`
	ClusterStatus    string                                 `bson:"clusterStatus" json:"clusterStatus"`
	Region           string                                 `bson:"region" json:"region" required:"true"`
}


func KafkaManagerRouter(g *echo.Group) {
	entrance := Entrance{}
	g.POST("", entrance.Save)
	g.GET("/:id",entrance.FindById)
	//	g.PUT("", kafka.UpdateStatus)
	g.PATCH("",entrance.UpdateDeploymentStatus)
}

func (entrance Entrance) UpdateDeploymentStatus(context echo.Context) error {
	namespace := context.QueryParam("namespace")
	region := context.QueryParam("region")
	if namespace=="" || region==""{
		log.Println("[ERROR] No Namespace or region is provided")
		return common.GenerateErrorResponse(context, nil, "Operation failed!")
	}
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
	event:=context.QueryParam("event")
	rm:= Entrance{
			Region:      region,
			Namespace: namespace,
	}.findByNamespaceAndRegion()
	if event==string(Status.ZOOKEEEPER_DEPLOYMENT_FAILED) {
		rm.DeploymentStatus=Status.ZOOKEEEPER_DEPLOYMENT_FAILED
	}else if event!=string(Status.ZOOKEEEPER_DEPLOYED){
		rm.DeploymentStatus=Status.ZOOKEEEPER_DEPLOYED
	} else if event!=string(Status.ZOOKEEEPER_TERMINATED){
		rm.DeploymentStatus=Status.ZOOKEEEPER_TERMINATED
	}else if event!=string(Status.ZOOKEEEPER_TERMINATION_FAILED){
		rm.DeploymentStatus=Status.ZOOKEEEPER_TERMINATION_FAILED
	}else{
		return common.GenerateErrorResponse(context,nil,"No Status found!")
	}
	err=rm.update()
	if err!=nil{
		return common.GenerateErrorResponse(context,err.Error(),"Failed to persist!")
	}
	return common.GenerateSuccessResponse(context,nil,"Operation Successful!")
}


func (entrance Entrance) FindById(context echo.Context) error {
	id := context.Param("id")

	if id==""{
		log.Println("[ERROR] Please pass a vpc id")
		return common.GenerateErrorResponse(context, errors.New("id is empty!"), "Operation failed!")
	}
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
		if authority == Authority.ROLE_SUPER_ADMIN {
			isSuperAdmin = true
			break
		}
	}

	if isSuperAdmin{
		clstr := Entrance{VpcId: id}.findByVpcId()
		if clstr.VpcId==""{
			return common.GenerateErrorResponse(context, errors.New("No record found by id:"+id), "No VPC found by id:"+id)
		}
		return common.GenerateSuccessResponse(context, clstr,"")
	}
	teamId := context.QueryParam("teamId")
	if teamId==""{
		return common.GenerateErrorResponse(context, errors.New("No teamId is found!"), "Please provide a teamId")
	}
	team := bson.ObjectIdHex(teamId)
	var isAuthorized bool
	for _, each := range user.TeamIdList {
		if team == each {
			isAuthorized = true
			break
		}
	}

	if isAuthorized{
		rm:= Entrance{VpcId: id}.findByVpcId()
		if rm.VpcId==""{
			return common.GenerateErrorResponse(context, errors.New("No zookeeper Entrance record found"), "No kafka cluster record found by id:"+id)
		}
		return common.GenerateSuccessResponse(context,rm,"")
	}
	return common.GenerateErrorResponse(context, errors.New("Unauthorized user!"), "User is not authorized to access this resource")
}


func (entrance Entrance) Save(context echo.Context) error {
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

	formData := new(Entrance)
	if err := context.Bind(formData); err != nil {
		log.Println("[ERROR] while binding form data", err.Error())
		return common.GenerateErrorResponse(context, nil, "Failed to parse playload")
	}

	errs := formData.Validate()

	if len(errs) > 0 {
		return common.GenerateErrorResponse(context, errs, "Failed to parse playload")
	}
	if formData.checkIfAnyClusterExists() {
		log.Println("[ERROR] Cluster already exists!")
		return common.GenerateErrorResponse(context, "[ERROR]:Zookeeper Entrance already exists in this namespace!", "Operation Failed!")
	}

	vpc,err:= utility.GetVPCNameByVPCId(formData.VpcId,user)
	if err!=nil{
		log.Println("[ERROR] Failed to get vpcname from  management!")
		return common.GenerateErrorResponse(context,"[ERROR] Failed to get vpcname from  management!","Operation failed")
	}

	if vpc.AvailableCPU<(formData.Resources.Limits.CPU*formData.Replicas)+(formData.Manager.Resources.Limits.CPU*formData.Replicas)|| vpc.AvailableMemory<(formData.Resources.Limits.Memory*formData.Replicas)+(formData.Manager.Resources.Limits.Memory*formData.Replicas){
		log.Println("[ERROR] Failed to create zookeeper entrance! VpcName:"+ vpc.Namespace+" Requested CPU:"+strconv.Itoa(formData.Resources.Limits.CPU*formData.Replicas+formData.Manager.Resources.Limits.CPU*formData.Replicas)+"m"+", Requested Memory: "+strconv.Itoa(formData.Resources.Limits.Memory*formData.Replicas+formData.Manager.Resources.Limits.Memory*formData.Replicas)+"Mi")
		return common.GenerateErrorResponse(context,"[ERROR] Failed to create zookeeper entrance! VpcName:"+ vpc.Namespace+" Requested CPU:"+strconv.Itoa(formData.Resources.Limits.CPU*formData.Replicas+formData.Manager.Resources.Limits.CPU*formData.Replicas)+"m"+", Requested Memory: "+strconv.Itoa(formData.Resources.Limits.Memory*formData.Replicas+formData.Manager.Resources.Limits.Memory*formData.Replicas)+"Mi","Operation failed")
	}
	formData.Resources.Requests.Memory=formData.Resources.Limits.Memory
	formData.Resources.Requests.CPU=formData.Resources.Limits.CPU/3
	formData.Manager.Resources.Requests.Memory=formData.Manager.Resources.Limits.Memory
	formData.Manager.Resources.Requests.CPU=formData.Manager.Resources.Limits.CPU/3
	formData.Namespace=vpc.Namespace
	formData.Status = Status.V
	formData.DeploymentStatus = Status.ZOOKEEEPER_DEPLOYMENT_INITIATED
	err = formData.save()
	if err != nil {
		return common.GenerateErrorResponse(context, err.Error(), "Failed to persist!")
	}

	cluster := formData.findByVpcIdAndRegion()
	err, failed := formData.publish(user, AgentCommand.GetAgentCommandString[AgentCommand.ADD_KAFKA_MANAGER_DEPLOYMENT])
	if failed {
		cluster.DeploymentStatus = Status.ZOOKEEEPER_DEPLOYMENT_FAILED
		cluster.update()
		log.Println("[ERROR] while publishing message", err)
		return common.GenerateErrorResponse(context, nil, err.Error())
	}
	err = utility.UpdateResource(formData.VpcId, (formData.Resources.Limits.CPU*formData.Replicas)+(formData.Manager.Resources.Limits.CPU*formData.Replicas), (formData.Resources.Limits.Memory*formData.Replicas), 0, "deduct", user)
	if err != nil {
		return common.GenerateErrorResponse(context, "[ERROR] Cluster cluster creation initiated but failed to reduce resource from vpc! ", "Failed to reduce resource from vpc!")
	}
	return common.GenerateSuccessResponse(context, formData, "Zookeeper entrance creation initiated!")
}

func (entrance Entrance) publish(user universal.User, agentCommand string) (error, bool) {
	var messageHeaders = make(map[string]string)
	messageHeaders["action"] = "create"
	response, er := utility.GetKubeClusterAgentInfo(entrance.VpcId, user)
	if er != nil || response.AgentInfo.PublicKey == "" {
		log.Println("[ERROR] Unable to Get Agent Public Key !!")
		return errors.New("[ERROR] Unable to Get Agent Public Key !!"), true
	}

	encryptedPublicKey, err := encryption.AES256().Encrypt(response.AgentInfo.PublicKey)
	if err != nil {
		log.Println("Could not encrypt agent public key: ", err.Error())
		return errors.New("[ERROR] Unable to Get Agent Public Key !!"), true
	}

	messageHeaders["encryptedAgentPublicKey"] = encryptedPublicKey
	messageHeaders["vpcId"] = entrance.VpcId

	header := events.KafkaMessageHeader{0, entrance.Namespace, agentCommand, uuid.NewV4().String(), entrance.CompanyId, messageHeaders}
	log.Println(response.KafkaTopic)

	err = events.PublishByKey(response.KafkaTopic, entrance, header, response.AgentInfo.PublicKey)
	if err != nil {
		log.Println("[ERROR]", err.Error())
		return errors.New("[ERROR] Unable to publish data!!"), true
	}
	log.Println("[INFO] Zookeeper Entrance deployment event published")
	return nil, false
}


func (entrance Entrance) Validate() []string{
	l := log.New(os.Stdout, "kafka cluster-struct validation", log.LstdFlags)
	validator := validator.NewValidatorWithLogger(l)
	errs := validator.Struct(entrance)
	return errs
}

func (entrance Entrance) checkIfAnyClusterExists() bool {
	query := bson.M{
		"$and": []bson.M{
			{"vpcId": entrance.VpcId},
			{"status": Status.V},
			{"region": entrance.Region},
		},
	}
	temp := new(Entrance)
	coll := db.GetDmManager().Db.Collection(ZookeeperEntranceCollection)
	result := coll.FindOne(db.GetDmManager().Ctx, query)

	err := result.Decode(temp)
	if err != nil {
		log.Println("[ERROR]", err)
	}
	if temp.Region == entrance.Region && temp.VpcId==entrance.VpcId {
		return true
	}
	return false
}
func (entrance Entrance) update() error {
	coll := db.GetDmManager().Db.Collection(ZookeeperEntranceCollection)
	res := coll.FindOneAndUpdate(db.GetDmManager().Ctx,nil, entrance,nil)
	if res != nil {
		return nil
	}
	log.Println("[ERROR] No document found!")
	return errors.New("No document found!")
}


func (entrance Entrance) save() error {
	coll := db.GetDmManager().Db.Collection(ZookeeperEntranceCollection)
	_, err := coll.InsertOne(db.GetDmManager().Ctx, entrance)
	if err != nil {
		log.Println("[ERROR] Insert document:", err.Error())
		return err
	}
	return nil
}

func (entrance Entrance) findByVpcId() Entrance {
	query := bson.M{
		"$and": []bson.M{
			{"vpcId": entrance.VpcId},
			{"status": Status.V},
		},
	}

	clusters := []Entrance{}
	coll := db.GetDmManager().Db.Collection(ZookeeperEntranceCollection)
	curser, err := coll.Find(db.GetDmManager().Ctx, query)
	if err!=nil{
		return Entrance{}
	}
	for curser.Next(context.TODO()) {
		elemValue := new(Entrance)
		err := curser.Decode(elemValue)
		if err != nil {
			log.Println("[ERROR] Failed to decode data", err)
			break
		}
		clusters = append(clusters, *elemValue)
	}

	if len(clusters)>0{
		return clusters[0]
	}
	return Entrance{}
}
func (entrance Entrance) findByVpcIdAndRegion() Entrance {
	query := bson.M{
		"$and": []bson.M{
			{"vpcId": entrance.VpcId},
			{"status": Status.V},
			{"region": entrance.Region},
		},
	}
	clusters := []Entrance{}
	coll := db.GetDmManager().Db.Collection(ZookeeperEntranceCollection)
	curser, err := coll.Find(db.GetDmManager().Ctx, query)
	if err!=nil{
		return Entrance{}
	}
	for curser.Next(context.TODO()) {
		elemValue := new(Entrance)
		err := curser.Decode(elemValue)
		if err != nil {
			log.Println("[ERROR] Failed to decode data", err)
			break
		}
		clusters = append(clusters, *elemValue)
	}

	if len(clusters)>0{
		return clusters[0]
	}
	return Entrance{}
}

func (entrance Entrance) findByNamespaceAndRegion() Entrance {
	query := bson.M{
		"$and": []bson.M{
			{"namespace": entrance.Namespace},
			{"region": entrance.Region},
		},
	}

	temp := new(Entrance)
	coll := db.GetDmManager().Db.Collection(ZookeeperEntranceCollection)
	result := coll.FindOne(db.GetDmManager().Ctx, query)

	err := result.Decode(temp)
	if err != nil {
		log.Println("[ERROR]", err)
	}
	return *temp
}