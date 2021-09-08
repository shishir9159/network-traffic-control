package cluster

import (
	"context"
	"errors"
	"github.com/go-bongo/bongo"
	"github.com/labstack/echo"
	"github.com/twinj/uuid"
	"gopkg.in/mgo.v2/bson"
	"klovercloud/queue/common"
	"klovercloud/queue/encryption"
	"klovercloud/queue/src/v1/adapter/enum/AgentCommand"
	_ "klovercloud/queue/src/v1/adapter/enum/AgentCommand"
	"klovercloud/queue/src/v1/adapter/enum/Authority"
	"klovercloud/queue/src/v1/adapter/enum/Status"
	"klovercloud/queue/src/v1/adapter/universal"
	"klovercloud/queue/src/v1/db"
	"klovercloud/queue/src/v1/events"
	"klovercloud/queue/src/v1/utility"
	"klovercloud/queue/src/v1/validator"
	"log"
	"os"
	"strconv"
)

const (
	KafkaClusterCollection       = "kafka_cluster"
	VERSION="2.5.0"
)
type Cluster struct {
	bongo.DocumentBase `bson:",inline"`
	Id                 string   `bson:"id" json:"id"`
	VpcId              string `bson:"vpcId" json:"vpcId" required:"true"`
	VpcName            string   `bson:"vpcName" json:"vpcName"`
	Version            string   `bson:"version" json:"version"`
	Organization       string   `bson:"organization" json:"organization"`
	Teams              []string `bson:"teams" json:"teams"`
	CompanyId          string `bson:"companyId" json:"companyId"`
	KafkaBrokerService struct{
		Ports [] string `bson:"ports" json:"ports"`
		Name  string `bson:"name" json:"name"`
	}`bson:"kafkaBroker" json:"kafkaBroker"`
	Metadata           struct {
		Name      string `bson:"name" json:"name" required:"true"`
		Namespace string `bson:"namespace" json:"namespace"`
	} `bson:"metadata" json:"metadata" required:"true" type:"struct"`
	Spec struct {
		Kafka struct {
			Version   string `bson:"version" json:"version" required:"true"`
			Replicas  int `bson:"replicas" json:"replicas" required:"true"`
			Resources struct {
				Requests struct {
					CPU    int   `bson:"cpu" json:"cpu"`
					Memory int `bson:"memory" json:"memory"`
				} `bson:"requests" json:"requests"`
				Limits struct {
					CPU    int   `bson:"cpu" json:"cpu"`
					Memory int `bson:"memory" json:"memory"`
				} `bson:"limits" json:"limits"`
			} `bson:"resources" json:"resources"`
            JvmOptions struct{
			    Xms int `bson:"-Xms" json:"-Xms"`
				Xmx int `bson:"-Xmx" json:"-Xmx"`
			}`bson:"jvmOptions" json:"jvmOptions"`
			Listeners struct {
				Plain struct {
				}`bson:"plain" json:"plain" type:"struct"`
				Tls struct {
				}`bson:"tls" json:"tls" type:"struct"`
			} `bson:"listeners" json:"listeners" required:"true" type:"struct"`
			Config map[string]string `bson:"config" json:"config" required:"true" type:"struct"`
            Storage struct{
            	Type string `bson:"type" json:"type"`
				Size int `bson:"size" json:"size"`
				DeleteClaim bool `bson:"deleteClaim" json:"deleteClaim"`
				Class string `bson:"class" json:"class"`
			}`bson:"storage" json:"storage" required:"true" type:"struct"`
		} `bson:"kafka" json:"kafka" required:"true" type:"struct"`
		Zookeeper struct{
			Replicas  int `bson:"replicas" json:"replicas" required:"true"`
			Resources struct {
				Requests struct {
					CPU    int   `bson:"cpu" json:"cpu"`
					Memory int `bson:"memory" json:"memory"`
				} `bson:"requests" json:"requests"`
				Limits struct {
					CPU    int   `bson:"cpu" json:"cpu"`
					Memory int `bson:"memory" json:"memory"`
				} `bson:"limits" json:"limits"`
			} `bson:"resources" json:"resources"`
			JvmOptions struct{
				Xms int `bson:"-Xms" json:"-Xms"`
				Xmx int `bson:"-Xmx" json:"-Xmx"`
			}`bson:"jvmOptions" json:"jvmOptions"`
			Storage struct{
				Type string `bson:"type" json:"type"`
				Size int `bson:"size" json:"size"`
				DeleteClaim bool `bson:"deleteClaim" json:"deleteClaim"`
				Class string `bson:"class" json:"class"`
			}`bson:"storage" json:"storage" required:"true" type:"struct"`
		} `bson:"zookeeper" json:"zookeeper" required:"true" type:"struct"`

	} `bson:"spec" json:"spec" required:"true" type:"struct"`
	Status           Status.Status                          `bson:"status" json:"status"`
	DeploymentStatus Status.Kafka_cluster_deployment_status `bson:"deployment_status" json:"deployment_status"`
	ClusterStatus    string                                 `bson:"clusterStatus" json:"clusterStatus"`
	Region           string                                 `bson:"region" json:"region" required:"true"`
}

func KafkaClusterRouter(g *echo.Group) {
	kafka := Cluster{}
	g.POST("", kafka.Save)
	g.GET("/:id",kafka.FindById)
//	g.PUT("", kafka.UpdateStatus)
	g.PATCH("",kafka.UpdateDeploymentStatus)
}


func (c Cluster) Save(context echo.Context) error {
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

	formData := new(Cluster)
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
		return common.GenerateErrorResponse(context, nil, "Cluster already exists! [ERROR]: Cluster:"+formData.Metadata.Name+", companyId:"+formData.CompanyId+", region:"+formData.VpcId)
	}

	vpc,err:= utility.GetVPCNameByVPCId(formData.VpcId,user)
	if err!=nil{
		log.Println("[ERROR] Failed to get vpcname from  management!")
		return common.GenerateErrorResponse(context,"[ERROR] Failed to get vpcname from  management!","Operation failed")
	}
	if vpc.AvailableCPU<(formData.Spec.Kafka.Resources.Requests.CPU*formData.Spec.Kafka.Replicas)+(formData.Spec.Zookeeper.Resources.Requests.CPU*formData.Spec.Zookeeper.Replicas)|| vpc.AvailableMemory<(formData.Spec.Kafka.Resources.Requests.Memory*formData.Spec.Kafka.Replicas)+(formData.Spec.Zookeeper.Resources.Requests.Memory*formData.Spec.Zookeeper.Replicas) ||vpc.AvailableStorage<(formData.Spec.Kafka.Storage.Size+formData.Spec.Zookeeper.Storage.Size){
		log.Println("[ERROR] Failed to create kafka cluster! VpcName:"+ vpc.Namespace+" Requested CPU:"+strconv.Itoa((formData.Spec.Kafka.Resources.Requests.CPU*formData.Spec.Kafka.Replicas)+(formData.Spec.Zookeeper.Resources.Requests.CPU*formData.Spec.Zookeeper.Replicas))+"m"+", Requested Memory: "+strconv.Itoa((formData.Spec.Kafka.Resources.Requests.Memory*formData.Spec.Kafka.Replicas)+(formData.Spec.Zookeeper.Resources.Requests.Memory*formData.Spec.Zookeeper.Replicas))+"Mi , Requested Storage: "+strconv.Itoa(formData.Spec.Kafka.Storage.Size+formData.Spec.Zookeeper.Storage.Size)+"Mi")
		return common.GenerateErrorResponse(context,"[ERROR] Failed to create kafka cluster! VpcName:"+ vpc.Namespace+" Requested CPU:"+strconv.Itoa((formData.Spec.Kafka.Resources.Requests.CPU*formData.Spec.Kafka.Replicas)+(formData.Spec.Zookeeper.Resources.Requests.CPU*formData.Spec.Zookeeper.Replicas))+"m"+", Requested Memory: "+strconv.Itoa((formData.Spec.Kafka.Resources.Requests.Memory*formData.Spec.Kafka.Replicas)+(formData.Spec.Zookeeper.Resources.Requests.Memory*formData.Spec.Zookeeper.Replicas))+"Mi , Requested Storage: "+strconv.Itoa(formData.Spec.Kafka.Storage.Size+formData.Spec.Zookeeper.Storage.Size)+"Mi","Operation failed")
	}
	formData.CompanyId = user.CompanyId.Hex()

	var teams []string
	if len(vpc.TeamList) > 0 {
		for _, each := range vpc.TeamList {
			teams = append(teams, each.Id)
		}
	}
	formData.Teams = teams
	if vpc.Organization.Id != "" || vpc.Organization.Id != "null" {
		formData.Organization = vpc.Organization.Id
	}
	formData.Metadata.Namespace=vpc.Namespace
	formData.Id = uuid.NewV4().String()
	formData.Status = Status.V
	formData.Version=VERSION
	formData.KafkaBrokerService.Ports=[] string{"9091","9092","9093"}
	formData.DeploymentStatus = Status.KAFKA_CLUSTER_DEPLOYMENT_INITIATED

	formData.Spec.Kafka.Resources.Requests.Memory = formData.Spec.Kafka.Resources.Limits.Memory
	formData.Spec.Kafka.Resources.Requests.CPU = formData.Spec.Kafka.Resources.Limits.CPU / 3

	formData.Spec.Zookeeper.Resources.Requests.Memory = formData.Spec.Zookeeper.Resources.Limits.Memory
	formData.Spec.Zookeeper.Resources.Requests.CPU = formData.Spec.Zookeeper.Resources.Limits.CPU / 3
	err = formData.save()
	if err != nil {
		return common.GenerateErrorResponse(context, err.Error(), "Failed to persist!")
	}
	cluster := formData.findByVpcIdAndRegion()
	err, failed := formData.publish(user, AgentCommand.GetAgentCommandString[AgentCommand.ADD_KAFKA_CLUSTER_DEPLOYMENT])
	if failed {
		cluster.DeploymentStatus = Status.KAFKA_CLUSTER_DEPLOYMENT_FAILED
		cluster.update()
		log.Println("[ERROR] while publishing message", err)
		return common.GenerateErrorResponse(context, nil, err.Error())
	}

	err = utility.UpdateResource(formData.VpcId, (formData.Spec.Kafka.Resources.Limits.CPU*formData.Spec.Kafka.Replicas)+(formData.Spec.Zookeeper.Resources.Limits.CPU*formData.Spec.Zookeeper.Replicas), (formData.Spec.Kafka.Resources.Limits.Memory*formData.Spec.Kafka.Replicas)+(formData.Spec.Zookeeper.Resources.Limits.Memory*formData.Spec.Zookeeper.Replicas), formData.Spec.Kafka.Storage.Size+formData.Spec.Zookeeper.Storage.Size, "deduct", user)
	if err != nil {
		return common.GenerateErrorResponse(context, "[ERROR] Cluster cluster creation initiated but failed to reduce resource from vpc! ", "Failed to reduce resource from vpc!")
	}
	return common.GenerateSuccessResponse(context, formData, "Kafka cluster creation initiated!")
}





func (cluster Cluster) FindById(context echo.Context) error {
	id := context.Param("id")
	scope := context.QueryParam("scope")
	log.Println("scope is:",scope)
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
		clstr := Cluster{VpcId: id}.findByVpcId()
		if clstr.VpcId==""{
			return common.GenerateErrorResponse(context, errors.New("No record found by id:"+id), "No VPC found by id:"+id)
		}
		if scope=="NAME"{
			log.Println("returning:",clstr.Metadata.Name)
			return common.GenerateSuccessResponse(context, clstr.Metadata.Name, "")
		}else if scope=="METADATA"{
			log.Println("returning:",clstr.Metadata)
			return common.GenerateSuccessResponse(context, clstr.Metadata, "")
		} else{
			return common.GenerateSuccessResponse(context, clstr, "")
		}
	}

	clstr:= Cluster{VpcId: id}.findByVpcId()
	var isAuthorized bool
	if len(user.TeamIdList) > 0 {
		for _, each := range user.TeamIdList {
			for _, autherizedTeam := range clstr.Teams {
				if each == bson.ObjectIdHex(autherizedTeam) {
					isAuthorized = true
					break
				}
			}

		}
	} else {
		isAuthorized = true
	}

	if isAuthorized{
		if clstr.VpcId==""{
			return common.GenerateErrorResponse(context, errors.New("No kafka cluster record"), "No kafka cluster record found by id:"+id)
		}

		if scope=="NAME"{
			log.Println("returning:",clstr.Metadata.Name)
			return common.GenerateSuccessResponse(context, clstr.Metadata.Name, "")
		}else if scope=="METADATA"{
			log.Println("returning:",clstr.Metadata)
			return common.GenerateSuccessResponse(context, clstr.Metadata, "")
		} else{
			return common.GenerateSuccessResponse(context, clstr, "")
		}
	}
	return common.GenerateErrorResponse(context, errors.New("Unauthorized user!"), "User is not authorized to access this resource")
}

func (cluster Cluster) save() error {
	coll := db.GetDmManager().Db.Collection(KafkaClusterCollection)
	_, err := coll.InsertOne(db.GetDmManager().Ctx, cluster)
	if err != nil {
		log.Println("[ERROR] Insert document:", err.Error())
		return err
	}
	return nil
}



func (cluster Cluster) UpdateDeploymentStatus(context echo.Context) error {
	namespace := context.QueryParam("namespace")
	name := context.QueryParam("name")
	if namespace=="" || name==""{
		log.Println("[ERROR] No Namespace or name is provided")
		return common.GenerateErrorResponse(context, nil, "Operation failed!")
	}
	user, _, err := utility.GetHeaderData(context)
	if err != nil {
		log.Println("[ERROR] while getting header data", err.Error())
		return common.GenerateErrorResponse(context, nil, "Operation failed!")
	}
	//permissionErr := utility.CheckPermission(user)
	//if permissionErr != nil {
	//	log.Println("[ERROR] while checking permission", permissionErr)
	//	return common.GenerateErrorResponse(context, nil, "Permission error!")
	//}
	event:=context.QueryParam("event")
	rm:= Cluster{
		Metadata: struct {
			Name      string `bson:"name" json:"name" required:"true"`
			Namespace string `bson:"namespace" json:"namespace"`
		}{
			Name:      name,
			Namespace: namespace,
		},
	}.findByNamespaceAndName()
	if event==string(Status.KAFKA_CLUSTER_DEPLOYMENT_FAILED) {
		rm.DeploymentStatus=Status.KAFKA_CLUSTER_DEPLOYMENT_FAILED
	}else if event==string(Status.KAFKA_CLUSTER_DEPLOYED){
		rm.DeploymentStatus=Status.KAFKA_CLUSTER_DEPLOYED
	} else if event==string(Status.KAFKA_CLUSTER_TERMINATED){
		rm.DeploymentStatus=Status.KAFKA_CLUSTER_TERMINATED
		err = utility.UpdateResource(rm.VpcId, (rm.Spec.Kafka.Resources.Limits.CPU*rm.Spec.Kafka.Replicas)+(rm.Spec.Zookeeper.Resources.Limits.CPU*rm.Spec.Zookeeper.Replicas), (rm.Spec.Kafka.Resources.Limits.Memory*rm.Spec.Kafka.Replicas)+(rm.Spec.Zookeeper.Resources.Limits.Memory*rm.Spec.Zookeeper.Replicas), rm.Spec.Kafka.Storage.Size+rm.Spec.Zookeeper.Storage.Size, "add", user)
	}else if event==string(Status.KAFKA_CLUSTER_TERMINATION_FAILED){
		rm.DeploymentStatus=Status.KAFKA_CLUSTER_TERMINATION_FAILED
	}else{
		return common.GenerateErrorResponse(context,nil,"No Status found!")
	}
	if err != nil {
		return common.GenerateErrorResponse(context, err.Error(), "Failed to update resource!")
	}
	log.Println(rm.DeploymentStatus)
	err=rm.update()
	if err!=nil{
		return common.GenerateErrorResponse(context,err.Error(),"Failed to persist!")
	}
	return common.GenerateSuccessResponse(context,nil,"Operation Successful!")
}


func (cluster Cluster) update() error {
	coll := db.GetDmManager().Db.Collection(KafkaClusterCollection)
	res := coll.FindOneAndUpdate(db.GetDmManager().Ctx,nil, cluster,nil)
	if res != nil {
		return nil
	}
	log.Println("[ERROR] No document found!")
	return errors.New("No document found!")
}

func (cluster Cluster) Validate() []string {
	l := log.New(os.Stdout, "kafka cluster-struct validation", log.LstdFlags)
	validator := validator.NewValidatorWithLogger(l)
	errs := validator.Struct(cluster)
	errs = append(errs, validator.Struct(cluster.Metadata)...)
	errs = append(errs, validator.Struct(cluster.Spec)...)
	errs = append(errs, validator.Struct(cluster.Spec.Kafka)...)
	errs = append(errs, validator.Struct(cluster.Spec.Kafka.Storage)...)
	errs = append(errs, validator.Struct(cluster.Spec.Kafka.Listeners)...)
	errs = append(errs, validator.Struct(cluster.Spec.Zookeeper)...)
	errs = append(errs, validator.Struct(cluster.Spec.Zookeeper.Storage)...)
	return errs
}

func (cluster Cluster) checkIfAnyClusterExists() bool {
	query := bson.M{
		"$and": []bson.M{
			{"vpcId": cluster.VpcId},
			{"status": Status.V},
			//{"region": cluster.Region},
		},
	}
	temp := new(Cluster)
	coll := db.GetDmManager().Db.Collection(KafkaClusterCollection)
	result := coll.FindOne(db.GetDmManager().Ctx, query)

	err := result.Decode(temp)
	if err != nil {
		log.Println("[ERROR]", err)
	}
	if temp.Metadata.Name == cluster.Metadata.Name {
		return true
	}
	return false
}


func (rabbitmq Cluster) findByVpcIdAndRegion() Cluster {
	query := bson.M{
		"$and": []bson.M{
			{"vpcId": rabbitmq.VpcId},
			{"status": Status.V},
			{"region": rabbitmq.Region},
		},
	}
	clusters := []Cluster{}
	coll := db.GetDmManager().Db.Collection(KafkaClusterCollection)
	curser, err := coll.Find(db.GetDmManager().Ctx, query)
	if err!=nil{
		return Cluster{}
	}
	for curser.Next(context.TODO()) {
		elemValue := new(Cluster)
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
	return Cluster{}
}

func (cluster Cluster) publish(user universal.User, agentCommand string) (error, bool) {
	var messageHeaders = make(map[string]string)
	messageHeaders["action"] = "create"
	response, er := utility.GetKubeClusterAgentInfo(cluster.VpcId, user)
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
	messageHeaders["vpcId"] = cluster.VpcId
	messageHeaders["applicationId"] = cluster.Id
	header := events.KafkaMessageHeader{0, cluster.Metadata.Namespace, agentCommand, uuid.NewV4().String(), cluster.CompanyId, messageHeaders}
	log.Println(response.KafkaTopic)

	err = events.PublishByKey(response.KafkaTopic, cluster, header, response.AgentInfo.PublicKey)
	if err != nil {
		log.Println("[ERROR]", err.Error())
		return errors.New("[ERROR] Unable to publish data!!"), true
	}
	log.Println("[INFO] Kafka cluster deployment event published")
	return nil, false
}


func (rabbitmq Cluster) findByVpcId() Cluster {
	query := bson.M{
		"$and": []bson.M{
			{"vpcId": rabbitmq.VpcId},
			{"status": Status.V},
		},
	}

	clusters := []Cluster{}
	coll := db.GetDmManager().Db.Collection(KafkaClusterCollection)
	curser, err := coll.Find(db.GetDmManager().Ctx, query)
	if err!=nil{
		return Cluster{}
	}
	for curser.Next(context.TODO()) {
		elemValue := new(Cluster)
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
	return Cluster{}
}


func (cluster Cluster) findByNamespaceAndName() Cluster {
	query := bson.M{
		"$and": []bson.M{
			{"metadata": cluster.Metadata},
		},
	}

	temp := new(Cluster)
	coll := db.GetDmManager().Db.Collection(KafkaClusterCollection)
	result := coll.FindOne(db.GetDmManager().Ctx, query)

	err := result.Decode(temp)
	if err != nil {
		log.Println("[ERROR]", err)
	}
	return *temp
}

func (c Cluster) FindByQuery(query bson.M) []Cluster {
	clusters := []Cluster{}
	if query == nil {
		return clusters
	}

	coll := db.GetDmManager().Db.Collection(KafkaClusterCollection)
	curser, _ := coll.Find(db.GetDmManager().Ctx, query)
	for curser.Next(context.TODO()) {
		elemValue := new(Cluster)
		err := curser.Decode(elemValue)
		if err != nil {
			log.Println("[ERROR]", err)
			break
		}
		clusters = append(clusters, *elemValue)
	}

	return clusters

}