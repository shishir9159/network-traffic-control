package config

import (
	"encoding/base64"
	"encoding/json"
	"github.com/joho/godotenv"
	"klovercloud/queue/src/v1/adapter/universal"
	"log"
	"os"
)

var RunMode string
var ServerPort string
var MongoServer string
var MongoPort string
var MongoUsername string
var MongoPassword string

var DatabaseConnectionString string
var DatabaseName string

var KafkaBroker string
var ManagementServiceUrl string
var MessagePublisherServiceUrl string

var TopicQueueDeploymentEvent string
var QueueDeploymentEventConsumerGroup string
var AES256_CIPHER_KEY string
var systemUserData string
var systemUser universal.User
var systemUserAsJsonStr string
func InitEnvironmentVariables() {
	RunMode = os.Getenv("RUN_MODE")
	if RunMode == "" {
		RunMode = DEVELOP
	}

	log.Println("RUN MODE:", RunMode)

	if RunMode != PRODUCTION {
		//Load .env file
		err := godotenv.Load()
		if err != nil {
			log.Println("ERROR:", err.Error())
			return
		}
	}

	ServerPort = os.Getenv("SERVER_PORT")

	MongoServer = os.Getenv("MONGO_SERVER")
	MongoPort = os.Getenv("MONGO_PORT")
	MongoUsername = os.Getenv("MONGO_USERNAME")
	MongoPassword = os.Getenv("MONGO_PASSWORD")

	DatabaseConnectionString = "mongodb://" + MongoUsername + ":" + MongoPassword + "@" + MongoServer + ":" + MongoPort
	DatabaseName = os.Getenv("DATABASE_NAME")
	KafkaBroker = os.Getenv("KAFKA_BROKER")
	ManagementServiceUrl = os.Getenv("MANAGEMENT_SERVICE_URL")
	MessagePublisherServiceUrl = os.Getenv("MESSAGE_PUBLISHER_SERVICE_URL")

	TopicQueueDeploymentEvent = os.Getenv("TOPIC_QUEUE_DEPLOYMENT_EVENT")
	AES256_CIPHER_KEY =os.Getenv("AES256_CIPHER_KEY")
	QueueDeploymentEventConsumerGroup = os.Getenv("CONSUMER_GROUP_QUEUE_DEPLOYMENT_EVENT")
	systemUserData = os.Getenv("KLOVERCLOUD_SYSTEM_USER")
	generateSystemUserAsJsonStr()
	generateSystemUser()
}

func GetAES256CipherKey() string {
	return AES256_CIPHER_KEY
}

func generateSystemUserAsJsonStr() {
	systemUserAsByte, err := base64.StdEncoding.DecodeString(systemUserData)
	if err != nil {
		return
	}
	systemUserAsJsonStr = string(systemUserAsByte)
}

func generateSystemUser() {
	if systemUserData == "" {
		log.Println("[ERROR] system user data doesn't exists")
		return
	}
	err := json.Unmarshal([]byte(systemUserAsJsonStr), &systemUser)
	if err != nil {
		log.Println(err.Error())
	}
}

func GetSystemUser() universal.User {
	return systemUser
}

func GetSystemUserAsJsonStr() string {
	return systemUserAsJsonStr
}

func GetSystemUserData() string {
	return systemUserData
}
