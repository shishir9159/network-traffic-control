package utility

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/labstack/echo"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"klovercloud/queue/common"
	"klovercloud/queue/config"
	"klovercloud/queue/src/v1/adapter/enum/Authority"
	"klovercloud/queue/src/v1/adapter/universal"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const (
	CURRENT_USER   = "current-user"
	SUCCESS_STATUS = "success"
)

func GetHeaderData(c echo.Context) (universal.User, bson.ObjectId, error) {
	currentUserString := c.Request().Header.Get(CURRENT_USER)
	log.Println("current-user:",currentUserString)
	user, err := universal.GenerateUserFromJsonStr(currentUserString)
	if err != nil {
		return user, "", err
	}
	return user, "", nil
}


func GetKubeClusterAgentInfo(vpcId string, requester universal.User) (*KubeCluster, error) {
	var userAsString = universal.GenerateJsonStringFromUser(requester)
	client := &http.Client{}
	log.Println(vpcId)
	req, err := http.NewRequest(http.MethodGet, config.ManagementServiceUrl+"/api/vpc/"+vpcId+"/kube-cluster", nil)
	if req != nil {
		req.Header.Set(CURRENT_USER, userAsString)
		req.Header.Set("Content-Type", "application/json")
	} else {
		log.Print("Error:", err.Error())
		return nil, errors.New("error creating request for fetching agent info")
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Print("[ERROR] something wrong with api connection. ", err.Error())
		return nil, errors.New("Something wrong with API connection")
	} else if strings.TrimSpace(resp.Status) != strconv.Itoa(http.StatusOK) {
		log.Print("[ERROR] something wrong with api connection. ", resp.Status)
		return nil, errors.New("Something wrong with API connection")
	}

	defer resp.Body.Close()
	jsonDataFromHttp, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print("Error in  getting response body: ", err.Error())
		return nil, errors.New("Something wrong with API connection")
	}

	log.Println("[DEBUG] kube cluster", string(jsonDataFromHttp))

	responseDto := common.ResponseDTO{}
	err = json.Unmarshal([]byte(jsonDataFromHttp), &responseDto) // here!
	if err != nil {
		log.Print("Error in un-marshalling response dto: ", err.Error())
		return nil, errors.New("Something wrong with connection")
	} else if responseDto.Status != SUCCESS_STATUS {
		return nil, errors.New(responseDto.Message)
	}
	dataByte, _ := json.Marshal(responseDto.Data)
	response := KubeCluster{}
	err = json.Unmarshal(dataByte, &response)
	if err != nil {
		log.Print("Error in un-marshalling data dto: ", err.Error())
		return nil, errors.New("Something wrong with connection")
	}
	kubeInfoInfoResponse := &response
	return kubeInfoInfoResponse, nil
}

func GetVPCNameByVPCId(vpcId string,requester universal.User) (VPCResponseDtoForManagement, error) {
	var userAsString = universal.GenerateJsonStringFromUser(requester)
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, config.ManagementServiceUrl+"/api/vpc/get/"+vpcId, nil)
	if req != nil {
		req.Header.Set(CURRENT_USER, userAsString)
		req.Header.Set("Content-Type", "application/json")
	} else {
		log.Print("Error:", err.Error())
		return VPCResponseDtoForManagement{}, errors.New("error creating request for fetching vpc info")
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Print("[ERROR] something wrong with api connection. ", err.Error())
		return VPCResponseDtoForManagement{}, errors.New("Something wrong with API connection")
	} else if strings.TrimSpace(resp.Status) != strconv.Itoa(http.StatusOK) {
		log.Print("[ERROR] something wrong with api connection. ", resp.Status)
		return VPCResponseDtoForManagement{}, errors.New("Something wrong with API connection")
	}

	defer resp.Body.Close()
	jsonDataFromHttp, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print("Error in  getting response body: ", err.Error())
		return VPCResponseDtoForManagement{}, errors.New("Something wrong with API connection")
	}

	log.Println("[DEBUG] kube cluster", string(jsonDataFromHttp))

	responseDto := common.ResponseDTO{}
	err = json.Unmarshal([]byte(jsonDataFromHttp), &responseDto) // here!
	if err != nil {
		log.Print("Error in un-marshalling response dto: ", err.Error())
		return VPCResponseDtoForManagement{}, errors.New("Something wrong with connection")
	} else if responseDto.Status != SUCCESS_STATUS {
		return VPCResponseDtoForManagement{}, errors.New(responseDto.Message)
	}
	dataByte, _ := json.Marshal(responseDto.Data)

	response :=VPCResponseDtoForManagement{}
	err = json.Unmarshal(dataByte, &response)
	if err != nil {
		log.Print("Error in un-marshalling data dto: ", err.Error())
		return VPCResponseDtoForManagement{}, errors.New("Something wrong with connection")
	}
	return response, nil
}

func UpdateResource(vpcId string,cpu int, memory int,storage int,action string,requester universal.User) error{
	var userAsString = universal.GenerateJsonStringFromUser(requester)

	type resource struct {
		Cpu int `json:"cpu"`
		Memory int `json:"memory"`
		Storage int `json:"storage"`
	}

	data:=resource{
		Cpu:     cpu,
		Memory:  memory,
		Storage: storage,
	}
	client := http.Client{}
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(data)
	req, err := http.NewRequest("PUT", config.ManagementServiceUrl+"/api/vpc/"+vpcId+"/resources?action="+action, buf)
	if req != nil {
		req.Header.Set(CURRENT_USER, userAsString)
		req.Header.Set("action-id",bson.NewObjectId().Hex())
		req.Header.Set("Content-Type", "application/json")
	} else {
		log.Print("Error:", err.Error())
		return  errors.New("error creating request for updating vpc resource!")
	}
	_, err = client.Do(req)
	log.Println(req)
	log.Println("Requesting to update vpc!")
	if err != nil {
		log.Println("Failed - update vpc resource.....")
		log.Println(err.Error(), "\n")
		return err
	}
	log.Println("Requesting to update vpc!","cpu:",cpu,", memory:",memory," ,storage:",storage)
	return nil
}


func CheckPermission(user universal.User) error {
	if user.HasAnyAuthority(Authority.ROLE_AGENT_SERVICE, Authority.ROLE_GIT_AGENT, Authority.ANONYMOUS,Authority.ROLE_SUPER_ADMIN,Authority.ROLE_ADMIN,Authority.ROLE_SYSTEM) {
		return nil
	}
	return errors.New("permission denied")
}
//

var IsLetter = regexp.MustCompile(`^[a-zA-Z0-9 ]+$`).MatchString
