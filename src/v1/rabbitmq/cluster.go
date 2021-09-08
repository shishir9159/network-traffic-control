package rabbitmq

import (
	"context"
	"errors"
	"github.com/go-bongo/bongo"
	"github.com/labstack/echo"
	"github.com/twinj/uuid"
	"gopkg.in/mgo.v2/bson"
	"klovercloud/queue/common"
	"klovercloud/queue/config"
	"klovercloud/queue/core"
	"klovercloud/queue/encryption"
	"klovercloud/queue/src/v1/adapter/enum/AgentCommand"
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
	"strings"
)

const (
	RabbitMQCollection       = "rabbitmq"
	UPDATE_SPEC              = "update_spec"
	UPDATE_RESOURCE          = "update_resource"
	UPDATE_REPLICA           = "update_replica"
	UPDATE_ADDITIONAL_PLUGIN = "update_additional_plugin"
	UPDATE_ADDITIONAL_CONFIG = "update_additional_config"
	UPDATE_TLS               = "update_tls"
	UPDATE_STORAGE           = "update_storage"
	TERMINATE_CLUSTER        = "terminate_cluster"
	VERSION                  = "3.8.8"
)

type Cluster struct {
	bongo.DocumentBase `bson:",inline"`
	VpcId              string   `bson:"vpcId" json:"vpcId" required:"true"`
	VpcName            string   `bson:"vpcName" json:"vpcName"`
	Id                 string   `bson:"id" json:"id"`
	Teams              []string `bson:"teams" json:"teams"`
	Team              string `bson:"team" json:"team"`
	Version            string   `bson:"version" json:"version"`
	Organization       string   `bson:"organization" json:"organization"`
	CompanyId          string   `bson:"companyId" json:"companyId"`
	ApiVersion         string   `bson:"-" json:"apiVersion"`
	Kind               string   `bson:"-" json:"kind"`
	Metadata           struct {
		Name      string `bson:"name" json:"name" required:"true"`
		Namespace string `bson:"namespace" json:"namespace"`
	} `bson:"metadata" json:"metadata" required:"true" type:"struct"`
	Spec struct {
		Replicas  int `bson:"replicas" json:"replicas" required:"true"`
		Resources struct {
			Requests struct {
				CPU    int `bson:"cpu" json:"cpu"`
				Memory int `bson:"memory" json:"memory" `
			} `bson:"requests" json:"requests"`
			Limits struct {
				CPU    int `bson:"cpu" json:"cpu" required:"true"`
				Memory int `bson:"memory" json:"memory" required:"true"`
			} `bson:"limits" json:"limits" required:"true" type:"struct"`
		} `bson:"resources" json:"resources" required:"true" type:"struct"`
		AdditionalPlugins []string `bson:"additionalPlugins" json:"additionalPlugins"`
		Tls               struct {
			SecretName   string `bson:"secretName" json:"secretName"`
			CaSecretName string `bson:"caSecretName" json:"caSecretName"`
			CaCertName   string `bson:"caCertName" json:"caCertName"`

			Secret struct {
				Crt string `bson:"crt" json:"crt"`
				Key string `bson:"key" json:"key"`
			} `bson:"secret" json:"secret" type:"struct"`

			CaSecret struct {
				Crt string `bson:"crt" json:"crt"`
				Key string `bson:"key" json:"key"`
			} `bson:"caSecret" json:"caSecret" type:"struct"`
		} `bson:"tls" json:"tls" type:"struct"`
		AdditionalConfigs []struct {
			Name  string `bson:"name" json:"name"`
			Value string `bson:"value" json:"value"`
		} `bson:"additionalConfigs" json:"additionalConfigs" `
		Storage struct {
			StorageClass string `bson:"storageClass" json:"storageClass"`
			Size         int    `bson:"size" json:"size" required:"true"`
		} `bson:"storage" json:"storage" required:"true" type:"struct"`
	} `bson:"spec" json:"spec" required:"true" type:"struct"`
	Status           Status.Status                             `bson:"status" json:"status"`
	DeploymentStatus Status.Rabbitmq_cluster_deployment_status `bson:"deployment_status" json:"deployment_status"`
	ClusterStatus    string                                    `bson:"clusterStatus" json:"clusterStatus"`
	Region           string                                    `bson:"region" json:"region" required:"true"`
	ClientService    struct {
		Name  string   `bson:"name" json:"name"`
		Ports []string `bson:"ports" json:"ports"`
	} `bson:"client_service" json:"client_service"`

	HeadlessService struct {
		Name  string   `bson:"name" json:"name"`
		Ports []string `bson:"ports" json:"ports"`
	} `bson:"headless_service" json:"headless_service"`

	Ingress  string `bson:"ingress" json:"ingress"`
	Username string `bson:"username" json:"username"`
	Password string `bson:"password" json:"password"`
}

func RabbitMQRouter(g *echo.Group) {
	rabbitMQ := Cluster{}
	g.POST("", rabbitMQ.Save)
	g.GET("", rabbitMQ.Find)
	g.GET("/:id", rabbitMQ.FindById)
	g.DELETE("/:id", rabbitMQ.Delete)
	g.PUT("", rabbitMQ.Update)
	g.PATCH("", rabbitMQ.UpdateDeploymentStatus)
}

func (rabbitMQ Cluster) UpdateDeploymentStatus(context echo.Context) error {

	namespace := context.QueryParam("namespace")
	name := context.QueryParam("name")

	username := context.QueryParam("username")
	password := context.QueryParam("password")
	log.Println("Updating queue..")
	if namespace == "" || name == "" {
		log.Println("[ERROR] No Namespace or name is provided")
		return common.GenerateErrorResponse(context, nil, "Operation failed!")
	}

	log.Println("Updating queue..")
	encrytptedUsername, err := core.Encrypt(username)
	if err != nil {
		return common.GenerateErrorResponse(context, "Failed to encrypt username", err.Error())
	}

	encrytptedPassword, err := core.Encrypt(password)
	if err != nil {
		return common.GenerateErrorResponse(context, "Failed to encrypt password", err.Error())
	}

	event := context.QueryParam("event")
	rm := Cluster{
		Metadata: struct {
			Name      string `bson:"name" json:"name" required:"true"`
			Namespace string `bson:"namespace" json:"namespace"`
		}{
			Name:      name,
			Namespace: namespace,
		},
	}.findByNamespaceAndName()
	log.Println("event: " + event)
	if event == string(Status.RABBITMQ_DEPLOYMENT_FAILED) {
		rm.DeploymentStatus = Status.RABBITMQ_DEPLOYMENT_FAILED
	} else if event == string(Status.RABBITMQ_DEPLOYED) {
		rm.DeploymentStatus = Status.RABBITMQ_DEPLOYED
		rm.Username = encrytptedUsername
		rm.Password = encrytptedPassword
		log.Println("RB deployed!")
	} else if event == string(Status.RABBITMQ_TERMINATED) {
		log.Println("RB terminated!")
			err = utility.UpdateResource(rm.VpcId, rm.Spec.Resources.Limits.CPU*rm.Spec.Replicas, rm.Spec.Resources.Limits.Memory*rm.Spec.Replicas, rm.Spec.Storage.Size, "add", config.GetSystemUser())
			if err!=nil{
				log.Println(err.Error())
			}
		rm.DeploymentStatus = Status.RABBITMQ_TERMINATED
		rm.Status = Status.D

	} else if event == string(Status.RABBITMQ_TERMINATION_FAILED) {
		log.Println("RB termination failed!")
		rm.DeploymentStatus = Status.RABBITMQ_TERMINATION_FAILED
	} else {
		return common.GenerateErrorResponse(context, nil, "No Status found!")
	}
	if err != nil {
		return common.GenerateErrorResponse(context, err.Error(), "Failed to update resource!")
	}
	log.Println(rm.DeploymentStatus)
	err = rm.update()
	if err != nil {

		return common.GenerateErrorResponse(context, err.Error(), "Failed to persist!")
	}
	return common.GenerateSuccessResponse(context, nil, "Operation Successful!")
}

func (rabbitMQ Cluster) Delete(context echo.Context) error {
	id := context.Param("id")

	if id == "" {
		log.Println("[ERROR] Please pass a cluster id")
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
		if authority == Authority.ROLE_SUPER_ADMIN || authority == Authority.ROLE_ADMIN {
			isSuperAdmin = true
			break
		}
	}
	rm := Cluster{Id: id}.findById()
	if isSuperAdmin {
		if rm.Id == "" {
			return common.GenerateErrorResponse(context, "No record found by id:"+id, "No VPC found by id:"+id)
		}
	}

	var isAuthorized bool


	if len(user.TeamIdList) > 0 {
		for _,each:=range user.TeamIdList{
			if each==bson.ObjectIdHex(rm.Team) {
				isAuthorized = true
				break
			}
		}
	} else {
		isAuthorized = true
	}

	rm.Kind = "RabbitmqCluster"
	rm.ApiVersion = "rabbitmq.com/v1beta1"
	if isSuperAdmin || isAuthorized {
		currDeploymentStatus := rm.DeploymentStatus
		currStatus := rm.Status
		rm.DeploymentStatus = Status.RABBITMQ_TERMINATION_INITIATED
		log.Println("Updating status", rm)
		err := rm.update()
		if err != nil {
			return common.GenerateErrorResponse(context, err.Error(), "Failed to delete cluster!")
		}
		err, _ = rm.publish(user, AgentCommand.GetAgentCommandString[AgentCommand.REMOVE_RABBITMQ_CLUSTER_DEPLOYMENT], "")
		if err != nil {
			rm.Status = currStatus
			rm.DeploymentStatus = currDeploymentStatus
			err := rm.update()
			return common.GenerateErrorResponse(context, err.Error(), "Cluster termination failed!")
		} else {
			return common.GenerateSuccessResponse(context, "Cluster termination initiated!", "Cluster termination initiated!")
		}
	}

	return common.GenerateErrorResponse(context, "[ERROR]: Unauthorized user!", "Operation failed!")

}

func (rabbitMQ Cluster) FindById(context echo.Context) error {

	id := context.Param("id")
	scope := context.QueryParam("scope")
	log.Println("scope is:", scope)
	if id == "" {
		log.Println("[ERROR] Please pass a cluster id")
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
		if authority == Authority.ROLE_SUPER_ADMIN || authority == Authority.ROLE_ADMIN {
			isSuperAdmin = true
			break
		}
	}

	if isSuperAdmin {
		rm := Cluster{Id: id}.findById()
		if rm.Id == "" {
			return common.GenerateErrorResponse(context, errors.New("No record found by id:"+id), "No VPC found by id:"+id)
		}
		if scope == "NAME" {
			log.Println("returning:", rm.Metadata.Name)
			return common.GenerateSuccessResponse(context, rm.Metadata.Name, "")
		} else if scope == "METADATA" {
			log.Println("returning:", rm.Metadata)
			return common.GenerateSuccessResponse(context, rm.Metadata, "")
		} else {

			if rm.Username != "" {
				username, err := core.Decrypt(rm.Username)
				if err != nil {
					log.Println("[ERROR]:", err.Error())
				}
				rm.Username = username
			}
			if rm.Password != "" {
				password, err := core.Decrypt(rm.Password)
				rm.Password = password
				if err != nil {
					log.Println("[ERROR]:", err.Error())
				}
			}
			return common.GenerateSuccessResponse(context, rm, "")
		}

	}

	rm := Cluster{VpcId: id}.findByVpcId()
	var isAuthorized bool
	log.Println("isAuthorized:", isAuthorized)
	if len(user.TeamIdList) > 0 {
		for _, each := range user.TeamIdList {
			if each==bson.ObjectIdHex(rm.Team){
				isAuthorized = true
				break
			}
		}
	} else {
		isAuthorized = true
	}
	log.Println("isAuthorized:", isAuthorized)
	if isAuthorized {
		if rm.VpcId == "" {
			return common.GenerateSuccessResponse(context, rm, "No rabbitmq cluster record found by id:"+id)
		}

		if scope == "NAME" {
			log.Println("returning:", rm.Metadata.Name)
			return common.GenerateSuccessResponse(context, rm.Metadata.Name, "")
		} else if scope == "METADATA" {
			log.Println("returning:", rm.Metadata)
			return common.GenerateSuccessResponse(context, rm.Metadata, "")
		} else {

			if rm.Username != "" {
				username, err := core.Decrypt(rm.Username)
				if err != nil {
					log.Println("[ERROR]:", err.Error())
				}
				rm.Username = username
			}
			if rm.Password != "" {
				password, err := core.Decrypt(rm.Password)
				rm.Password = password
				if err != nil {
					log.Println("[ERROR]:", err.Error())
				}
			}
			return common.GenerateSuccessResponse(context, rm, "")
		}

	}
	return common.GenerateErrorResponse(context, errors.New("Unauthorized user!"), "User is not authorized to access this resource")
}

func (rabbitMQ Cluster) Save(context echo.Context) error {
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

	sidecarToken := context.QueryParam("sidecarToken")
	if sidecarToken == "" {
		log.Println("[ERROR] sidecar token is Empty ")
		return common.GenerateErrorResponse(context, nil, "Sidecar token is empty!")
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

	isAuthorized:=false

	for _,each:=range user.TeamIdList{
		if each==bson.ObjectIdHex(formData.Team){
			isAuthorized=true
			break;
		}
	}
if isAuthorized==false {
	for _, authority := range user.Authorities {
		if authority == Authority.ROLE_SUPER_ADMIN {
			isAuthorized = true
			break
		} else if authority == Authority.ROLE_ADMIN {
			isAuthorized = true
			break
		}
	}
}
	if isAuthorized==false{
		log.Println("[ERROR] UnAuthorized, Team KubeCluster not matched!")
		return common.GenerateErrorResponse(context, "[ERROR] UnAuthorized, Team KubeCluster not matched!", "UnAuthorized, Team KubeCluster not matched!")

	}

	if !utility.IsLetter(formData.Metadata.Name) {
		log.Println("[ERROR] CLuster name can't contain any special character!")
		return common.GenerateErrorResponse(context, "[ERROR] CLuster name can't contain any special character!", "Please provide a valid name of your cluster!")
	}

	if formData.checkIfAnyClusterExists() {
		log.Println("[ERROR] Cluster already exists!")
		return common.GenerateErrorResponse(context, nil, "Cluster already exists! [ERROR]: Cluster:"+formData.Metadata.Name+", companyId:"+formData.CompanyId+", region:"+formData.VpcId)
	}

	vpc, err := utility.GetVPCNameByVPCId(formData.VpcId, user)
	if err != nil {
		log.Println("[ERROR] Failed to get vpcname from  management!")
		return common.GenerateErrorResponse(context, "[ERROR] Failed to get vpcname from  management!", "Operation failed")
	}
	if vpc.AvailableCPU>0 {
		if vpc.AvailableCPU < formData.Spec.Resources.Limits.CPU*formData.Spec.Replicas || vpc.AvailableMemory < formData.Spec.Resources.Limits.Memory*formData.Spec.Replicas || vpc.AvailableStorage < formData.Spec.Storage.Size {
			log.Println("[ERROR] Failed to create rabbitMQ! VpcName:" + vpc.Namespace + " Requested CPU:" + strconv.Itoa(formData.Spec.Resources.Limits.CPU*formData.Spec.Replicas) + "m" + ", Requested Memory: " + strconv.Itoa(formData.Spec.Resources.Limits.Memory*formData.Spec.Replicas) + "Mi , Requested Storage: " + strconv.Itoa(formData.Spec.Storage.Size) + "Mi")
			return common.GenerateErrorResponse(context, "[ERROR] Failed to create rabbitMQ! VpcName:"+vpc.Namespace+" Requested CPU:"+strconv.Itoa(formData.Spec.Resources.Limits.CPU*formData.Spec.Replicas)+"m"+", Requested Memory: "+strconv.Itoa(formData.Spec.Resources.Limits.Memory*formData.Spec.Replicas)+"Mi , Requested Storage: "+strconv.Itoa(formData.Spec.Storage.Size)+"Mi", "Operation failed")
		}
	}
	log.Println("[ERROR] invalid companyId!" + ": " + vpc.CompanyId + ", users company KubeCluster: " + user.CompanyId.String())
	if bson.ObjectIdHex(vpc.CompanyId) != user.CompanyId {
		log.Println("[ERROR] invalid companyId!" + ": " + vpc.CompanyId + ", users company KubeCluster: " + user.CompanyId.String())
		return common.GenerateErrorResponse(context, "[ERROR] Invalid company KubeCluster!", "Operation failed")
	}

	formData.CompanyId = user.CompanyId.Hex()

	var teams []string
	if len(vpc.TeamList) > 0 {
		for _, each := range vpc.TeamList {
			teams = append(teams, each.Id)
		}
	}

	formData.Metadata.Name = strings.Replace(formData.Metadata.Name, " ", "", -1)
	formData.Id = uuid.NewV4().String()
	formData.Version = VERSION
	formData.VpcName = vpc.Name

	formData.Teams = teams
	if vpc.Organization.Id != "" || vpc.Organization.Id != "null" {
		formData.Organization = vpc.Organization.Id
	}
	formData.Spec.Resources.Requests.Memory = formData.Spec.Resources.Limits.Memory
	formData.Spec.Resources.Requests.CPU = formData.Spec.Resources.Limits.CPU / 3
	formData.Metadata.Namespace = vpc.Namespace
	formData.Status = Status.V
	formData.ClientService.Name = formData.Metadata.Name + "-rabbitmq-client"
	formData.ClientService.Ports = []string{"5672", "15672"}
	formData.HeadlessService.Name = formData.Metadata.Name + "-rabbitmq-headless"
	formData.HeadlessService.Ports = []string{"4369", "25672"}
	formData.DeploymentStatus = Status.RABBITMQ_DEPLOYMENT_INITIATED

	kubeCluster, er := utility.GetKubeClusterAgentInfo(formData.VpcId, user)
	if er != nil || kubeCluster.AgentInfo.PublicKey == "" {
		log.Println("[ERROR] Unable to Get Agent Public Key !!")
		return common.GenerateErrorResponse(context, "[ERROR] Unable to Get Agent Public Key !!", "Failed to create cluster!")
	}

	formData.Ingress = formData.Metadata.Name + "-" + formData.VpcId + "." + kubeCluster.DefaultDomain
	err = formData.save()

	if err != nil {
		return common.GenerateErrorResponse(context, err.Error(), "Failed to persist!")
	}
	cluster := formData.findByVpcIdAndRegion()

	err, failed := formData.publish(user, AgentCommand.GetAgentCommandString[AgentCommand.ADD_RABBITMQ_CLUSTER_DEPLOYMENT], sidecarToken)
	if failed {
		cluster.DeploymentStatus = Status.RABBITMQ_DEPLOYMENT_FAILED
		cluster.update()
		log.Println("[ERROR] while publishing message", err)
		return common.GenerateErrorResponse(context, nil, err.Error())
	}
	log.Println("Requesting to update vpc!")
	err = utility.UpdateResource(formData.VpcId, formData.Spec.Resources.Limits.CPU*formData.Spec.Replicas, formData.Spec.Resources.Limits.Memory*formData.Spec.Replicas, formData.Spec.Storage.Size, "deduct", user)
	if err != nil {
		return common.GenerateErrorResponse(context, "[ERROR] Cluster creation initiated but failed to reduce resource from vpc! ", "Failed to reduce resource from vpc!")
	}
	return common.GenerateSuccessResponse(context, formData, "Cluster creation initiated!")
}

func (rabbitMQ Cluster) Update(context echo.Context) error {
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
	sidecarToken := context.QueryParam("sidecarToken")
	if sidecarToken == "" {
		log.Println("[ERROR] sidecar token is Empty ")
		return common.GenerateErrorResponse(context, nil, "Sidecar token is empty!")
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
	//vpc,err:= utility.GetVPCById(formData.VpcId,user)
	if err != nil {
		log.Println("[ERROR] Failed to get vpcname from  management!")
		return common.GenerateErrorResponse(context, "[ERROR] Failed to get vpcname from  management!", "Operation failed")
	}
	actions := strings.Split(context.QueryParam("actions"), ",")
	log.Println("companyId:", user.CompanyId.Hex(), " ", "id:", formData.Id)
	query := bson.M{
		"$and": []bson.M{{"companyId": user.CompanyId.Hex()}, {"id": formData.Id}, {"status": Status.V}},
	}
	clusters := rabbitMQ.FindByQuery(query)

	if len(clusters) == 0 {
		log.Println("[ERROR] No Cluster exists")
		return common.GenerateErrorResponse(context, nil, "No Cluster exists! [ERROR]: Cluster:"+formData.Metadata.Name+", companyId:"+formData.CompanyId+", region:"+formData.VpcId+", please create one first!")
	}
	cluster := clusters[0]

	cluster.ApiVersion = formData.ApiVersion
	cluster.Kind = formData.Kind
	formData.Status = Status.V

	var isSuperAdmin bool

	for _, authority := range user.Authorities {
		if authority == Authority.ROLE_SUPER_ADMIN || authority == Authority.ROLE_ADMIN {
			isSuperAdmin = true
			break
		}
	}

	//	var isUserAuthorized bool
	//for _, each := range user.TeamIdList {
	//	if each == bson.ObjectIdHex(formData.TeamId) {
	//		isUserAuthorized = true
	//		break
	//	}
	//}
	if isSuperAdmin {
		var terminate_cluster, update_spec, update_replica, update_resource, update_additional_config, update_additional_plugin, update_tls, update_storage bool

		for _, each := range actions {
			if each == UPDATE_SPEC {
				update_spec = true
			}
			if each == UPDATE_REPLICA {
				update_replica = true
			}
			if each == UPDATE_RESOURCE {
				update_resource = true
			}
			if each == UPDATE_ADDITIONAL_CONFIG {
				update_additional_config = true
			}
			if each == UPDATE_ADDITIONAL_PLUGIN {
				update_additional_plugin = true
			}
			if each == UPDATE_TLS {
				update_tls = true
			}
			if each == UPDATE_STORAGE {
				update_storage = true
			}
			if each == TERMINATE_CLUSTER {
				terminate_cluster = true
			}
		}
		log.Println(terminate_cluster, update_spec, update_replica, update_resource)
		log.Println("mem:", formData.Spec.Resources.Limits.Memory, "cpu", formData.Spec.Resources.Limits.CPU)
		if terminate_cluster {
			cluster.DeploymentStatus = Status.RABBITMQ_TERMINATION_INITIATED
		} else {
			if update_spec {
				cluster.Spec.Resources.Requests.Memory = formData.Spec.Resources.Limits.Memory
				cluster.Spec.Resources.Requests.CPU = formData.Spec.Resources.Limits.CPU / 3
				cluster.Spec.Resources.Limits.Memory = formData.Spec.Resources.Limits.Memory
				cluster.Spec.Resources.Limits.CPU = formData.Spec.Resources.Limits.CPU
				formData.Spec.Resources.Requests.Memory = cluster.Spec.Resources.Requests.Memory
				formData.Spec.Resources.Limits.CPU = cluster.Spec.Resources.Requests.CPU
				cluster.Spec.Replicas = formData.Spec.Replicas
				cluster.Spec.Storage = formData.Spec.Storage
				cluster.Spec.Tls = formData.Spec.Tls
				cluster.Spec.AdditionalPlugins = formData.Spec.AdditionalPlugins
				cluster.Spec.AdditionalConfigs = formData.Spec.AdditionalConfigs
			} else if update_replica {
				cluster.Spec.Replicas = formData.Spec.Replicas
			} else if update_resource {
				log.Println("mem:", formData.Spec.Resources.Limits.Memory, "cpu", formData.Spec.Resources.Limits.CPU)
				cluster.Spec.Resources.Requests.Memory = formData.Spec.Resources.Limits.Memory
				cluster.Spec.Resources.Requests.CPU = formData.Spec.Resources.Limits.CPU / 3
				cluster.Spec.Resources.Limits.Memory = formData.Spec.Resources.Limits.Memory
				cluster.Spec.Resources.Limits.CPU = formData.Spec.Resources.Limits.CPU
				formData.Spec.Resources.Requests.Memory = cluster.Spec.Resources.Requests.Memory
				formData.Spec.Resources.Limits.CPU = cluster.Spec.Resources.Requests.CPU
				//	cluster.Spec.Resources = formData.Spec.Resources
			} else if update_storage {
				cluster.Spec.Storage = formData.Spec.Storage
			} else if update_tls {
				cluster.Spec.Tls = formData.Spec.Tls
			} else if update_additional_plugin {
				cluster.Spec.AdditionalPlugins = formData.Spec.AdditionalPlugins
			} else if update_additional_config {
				cluster.Spec.AdditionalConfigs = formData.Spec.AdditionalConfigs
			}
		}
		err = cluster.update()
		if err != nil {
			return common.GenerateErrorResponse(context, err.Error(), "Failed to persist!")
		}

		if cluster.DeploymentStatus == Status.RABBITMQ_TERMINATION_INITIATED {
			err, _ := formData.publish(user, AgentCommand.GetAgentCommandString[AgentCommand.REMOVE_RABBITMQ_CLUSTER_DEPLOYMENT], sidecarToken)
			if err != nil {
				return common.GenerateSuccessResponse(context, err.Error(), "Cluster termination failed!")
			} else {
				return common.GenerateSuccessResponse(context, nil, "Cluster termination initiated!")
			}
		} else {
			err, _ := formData.publish(user, AgentCommand.GetAgentCommandString[AgentCommand.ADD_RABBITMQ_CLUSTER_DEPLOYMENT], sidecarToken)
			if err != nil {
				return common.GenerateSuccessResponse(context, err.Error(), "Cluster updating failed!")
			} else {
				return common.GenerateSuccessResponse(context, nil, "Cluster upgradation initiated!")
			}
		}
		//return common.GenerateSuccessResponse(context, nil, "Cluster updated successfully!")
	} else {
		return common.GenerateErrorResponse(context, "User not authorized!", "Failed to persist!")
	}
}

func (rabbitMQ Cluster) Find(context echo.Context) error {
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
    var IsAdmin bool
	for _, authority := range user.Authorities {
		if authority == Authority.ROLE_SUPER_ADMIN  {
			isSuperAdmin = true
			break
		}

		if authority==Authority.ROLE_ADMIN{
			IsAdmin=true
			break
		}
	}

	vpcId := context.QueryParam("vpcId")
	region := context.QueryParam("region")
	companId:= context.QueryParam("companyId")

	if isSuperAdmin && companId==""{
		return common.GenerateErrorResponse(context,"[ERROR] No CompanyId is provided!","Operation Failed!")
	}

	if companId==""{
		companId=user.CompanyId.Hex()
	}

	tempRmqs := []Cluster{}
	var query bson.M

	if isSuperAdmin || IsAdmin{
		if  vpcId != "null" && region != "" && region != "null" {
			log.Println("Querying with vpcId and region", "vpcId: "+vpcId, "region:"+region)
			query = bson.M{"$and": []bson.M{{"companyId": companId}, {"region": region}, {"vpcId": vpcId}, {"status": Status.V}}}
		} else if region != "" && region != "null" {
			log.Println("Querying with region", "region:"+region)
			query = bson.M{"$and": []bson.M{{"companyId": companId}, {"region": region}, {"status": Status.V}}}
		} else if vpcId != "null" {
			log.Println("Querying with vpcId ", "vpcId: "+vpcId)
			query = bson.M{"$and": []bson.M{{"vpcId": vpcId}, {"status": Status.V}}}
		} else {
			log.Println("Querying with companyId  ", "companyId: "+companId)
			query = bson.M{"$and": []bson.M{{"companyId": companId}, {"status": Status.V}}}
		}
		tempRmqs = rabbitMQ.FindByQuery(query)
		log.Println(len(tempRmqs))
		log.Println("query:", query)
	} else {
		log.Println("regular user!")
		if  vpcId != "null" && region != "" {
			log.Println("Querying with vpcId and region", "vpcId: "+vpcId, "region:"+region)
			query = bson.M{"$and": []bson.M{{"companyId":companId},{"region": region}, {"vpcId": vpcId}, {"status": Status.V}}}
		} else if region != "" && region != "null" {
			log.Println("Querying with region", "region:"+region)
			query = bson.M{"$and": []bson.M{{"companyId": companId},{"region": region}, {"status": Status.V}}}
		} else if  vpcId != "null" {
			log.Println("Querying with vpcId ", "vpcId: "+vpcId)
			query = bson.M{"$and": []bson.M{{"companyId": companId},{"vpcId": vpcId}, {"status": Status.V}}}
		}
		tempRmqs = rabbitMQ.FindByQuery(query)
	}
	log.Println(" teams:", user.TeamIdList)
	rmqs := []Cluster{}

	if IsAdmin || isSuperAdmin{
		rmqs=tempRmqs
	}else{
		for _, each := range tempRmqs {
			for _, userTeam := range user.TeamIdList {
				if userTeam==bson.ObjectIdHex(each.Team){
					rmqs = append(rmqs, each)
					break
				}
			}
		}
	}
	//	query = bson.M{}
	return common.GenerateSuccessResponse(context, rmqs, "")
}

func (rabbitMQ Cluster) Validate() []string {
	l := log.New(os.Stdout, "rabbitmq-struct validation", log.LstdFlags)
	validator := validator.NewValidatorWithLogger(l)
	errs := validator.Struct(rabbitMQ)
	errs = append(errs, validator.Struct(rabbitMQ.Metadata)...)
	errs = append(errs, validator.Struct(rabbitMQ.Spec)...)
	errs = append(errs, validator.Struct(rabbitMQ.Spec.Storage)...)
	errs = append(errs, validator.Struct(rabbitMQ.Spec.Tls)...)
	errs = append(errs, validator.Struct(rabbitMQ.Spec.Resources)...)
	errs = append(errs, validator.Struct(rabbitMQ.Spec.Resources.Limits)...)
	errs = append(errs, validator.Struct(rabbitMQ.Spec.Resources.Requests)...)
	for _, config := range rabbitMQ.Spec.AdditionalConfigs {
		errs = append(errs, validator.Struct(config)...)
	}
	return errs
}

func (rabbitMQ Cluster) checkIfAnyClusterExists() bool {
	query := bson.M{
		"$and": []bson.M{
			{"vpcId": rabbitMQ.VpcId},
			{"status": Status.V},
			//	{"region": rabbitMQ.Region},
		},
	}
	temp := new(Cluster)
	coll := db.GetDmManager().Db.Collection(RabbitMQCollection)
	result := coll.FindOne(db.GetDmManager().Ctx, query)

	err := result.Decode(temp)
	if err != nil {
		log.Println("[ERROR]", err)
	}
	if temp.Metadata.Name == rabbitMQ.Metadata.Name {
		return true
	}
	return false
}

func (rabbitMQ Cluster) save() error {
	coll := db.GetDmManager().Db.Collection(RabbitMQCollection)
	_, err := coll.InsertOne(db.GetDmManager().Ctx, rabbitMQ)
	if err != nil {
		log.Println("[ERROR] Insert document:", err.Error())
		return err
	}
	return nil
}

func (rabbitMQ Cluster) update() error {
	query := bson.M{
		"$and": []bson.M{
			{"id": rabbitMQ.Id},
		},
	}
	coll := db.GetDmManager().Db.Collection(RabbitMQCollection)
	_, err := coll.DeleteOne(db.GetDmManager().Ctx, query)
	if err != nil {
		return err
	}
	err = rabbitMQ.save()
	if err != nil {
		return errors.New("No document found!")
	}
	return nil

}

func (rabbitMQ Cluster) FindByQuery(query bson.M) []Cluster {
	clusters := []Cluster{}
	if query == nil {
		return clusters
	}
	coll := db.GetDmManager().Db.Collection(RabbitMQCollection)
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

func (rabbitmq Cluster) findByVpcId() Cluster {
	query := bson.M{
		"$and": []bson.M{
			{"vpcId": rabbitmq.VpcId},
			{"status": Status.V},
		},
	}

	clusters := []Cluster{}
	coll := db.GetDmManager().Db.Collection(RabbitMQCollection)
	curser, err := coll.Find(db.GetDmManager().Ctx, query)
	if err != nil {
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

	if len(clusters) > 0 {
		return clusters[0]
	}
	return Cluster{}

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
	coll := db.GetDmManager().Db.Collection(RabbitMQCollection)
	curser, err := coll.Find(db.GetDmManager().Ctx, query)
	if err != nil {
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

	if len(clusters) > 0 {
		return clusters[0]
	}
	return Cluster{}

}

func (rabbitMQ Cluster) publish(user universal.User, agentCommand string, sidecarToken string) (error, bool) {
	log.Println("cluster id", rabbitMQ.Id)
	var messageHeaders = make(map[string]string)
	messageHeaders["action"] = "create"
	response, er := utility.GetKubeClusterAgentInfo(rabbitMQ.VpcId, user)
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
	messageHeaders["vpcId"] = rabbitMQ.VpcId
	messageHeaders["applicationId"] = rabbitMQ.Id
	messageHeaders["sidecarToken"] = sidecarToken
	messageHeaders["companyId"] = rabbitMQ.CompanyId
	header := events.KafkaMessageHeader{0, rabbitMQ.Metadata.Namespace, agentCommand, uuid.NewV4().String(), rabbitMQ.CompanyId, messageHeaders}
	log.Println(response.KafkaTopic)

	err = events.PublishByKey(response.KafkaTopic, rabbitMQ, header, response.AgentInfo.PublicKey)
	if err != nil {
		log.Println("[ERROR]", err.Error())
		return errors.New("[ERROR] Unable to publish data!!"), true
	}
	log.Println("[INFO] Cluster deployment event published")
	return nil, false
}

func (rabbitMQ Cluster) findByNamespaceAndName() Cluster {
	query := bson.M{
		"$and": []bson.M{
			{"metadata": rabbitMQ.Metadata},
			{"status": Status.V},
		},
	}

	temp := new(Cluster)
	coll := db.GetDmManager().Db.Collection(RabbitMQCollection)
	result := coll.FindOne(db.GetDmManager().Ctx, query)

	err := result.Decode(temp)
	if err != nil {
		log.Println("[ERROR]", err)
	}
	return *temp
}

func (rabbitMQ Cluster) findById() Cluster {
	query := bson.M{
		"$and": []bson.M{
			{"id": rabbitMQ.Id},
			{"status": Status.V},
		},
	}

	temp := new(Cluster)
	coll := db.GetDmManager().Db.Collection(RabbitMQCollection)
	result := coll.FindOne(db.GetDmManager().Ctx, query)

	err := result.Decode(temp)
	if err != nil {
		log.Println("[ERROR]", err)
	}
	return *temp
}

func Test() {
	rs := Cluster{
		VpcId: "5f81a604c5fd070001456a25",
	}.findByVpcId()

	log.Println(rs)
}
