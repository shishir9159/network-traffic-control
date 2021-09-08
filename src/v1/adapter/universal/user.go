package universal

import (
	"encoding/json"
	"errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"gopkg.in/mgo.v2/bson"
	"klovercloud/queue/src/v1/adapter/dto"
	"klovercloud/queue/src/v1/adapter/enum/AccountType"
	"klovercloud/queue/src/v1/adapter/enum/Authority"
	"klovercloud/queue/src/v1/adapter/enum/InitialSettingStatus"
	"klovercloud/queue/src/v1/adapter/enum/UserType"

	"log"
	"strconv"
	"strings"
)

type User struct {
	BaseEntity            `bson:",inline"`
	UserName              string                                      `json:"username"`
	Password              string                                      `json:"password"`
	VerificationCode      string                                      `json:"verificationCode"`
	AccountNonExpired     bool                                        `json:"accountNonExpired"`
	AccountNonLocked      bool                                        `json:"accountNonLocked"`
	CredentialsNonExpired bool                                        `json:"credentialsNonExpired"`
	IsEnabled             bool                                        `json:"isEnabled"`
	NeedToResetPassword   bool                                        `json:"needToResetPassword"`
	UserInfo              UserInfo                                    `json:"userInfo"`
	CompanyId             bson.ObjectId                               `json:"companyId"`
	OrganizationId        bson.ObjectId                               `json:"organizationId"`
	TeamIdList            []bson.ObjectId                             `json:"teamIds"`
	AdminTeamIdList       []primitive.ObjectID                        `json:"adminTeamIdList"`
	Authorities           []Authority.Authority                       `json:"authorities"`
	UserType              UserType.UserType                           `json:"userType"`
	InitialSettingStatus  []InitialSettingStatus.InitialSettingStatus `json:"initialSettingStatus"`
}

func (user User) HasTeam(teamId bson.ObjectId) bool {
	if len(user.TeamIdList) == 0 {
		return false
	}
	for _, teamObjectId := range user.TeamIdList {
		if teamObjectId.Hex() == teamId.Hex() {
			return true
		}
	}
	return false
}

func (user User) HasAllAuthority(authority ...Authority.Authority) bool {
	if len(user.Authorities) == 0 {
		return false
	}
	for _, auth := range authority {
		if !Authority.ContainsAuthority(user.Authorities, auth) {
			return false
		}
	}
	return true
}

func (user User) HasAuthority(authority Authority.Authority) bool {
	if len(user.Authorities) == 0 {
		return false
	}
	if Authority.ContainsAuthority(user.Authorities, authority) {
		return true
	}
	return false
}

func (user User) HasAnyAuthority(authority ...Authority.Authority) bool {
	if len(user.Authorities) == 0 {
		return false
	}
	for _, auth := range authority {
		if Authority.ContainsAuthority(user.Authorities, auth) {
			return true
		}
	}
	return false
}

func GenerateUserFromJsonStr(userString string) (User, error) {
	user := User{}
	if userString == "" {
		log.Println("Error parsing template to json (empty user info string)")
		return user, errors.New("Error parsing template to json: User Info Not Found")
	}
	userReq := dto.UserHeaderRequestDTO{}
	err := json.Unmarshal([]byte(userString), &userReq)
	if err != nil {
		log.Println("userString:",userString)
		log.Println("Error parsing template to json (create resource-quota config): ", err)
		return user, err
	}
	userObjectId := bson.ObjectIdHex(userReq.Id)

	if userReq.CompanyId != "" {
		user.CompanyId = bson.ObjectIdHex(userReq.CompanyId)
	}

	if userReq.Authorities == "" {
		log.Println("Error parsing Authorities of User")
		return user, errors.New("User do not have authority to access")
	}


	// Authorities
	authorityStringList := strings.Split(userReq.Authorities, ",")
	var authorities []Authority.Authority
	for i := 0; i < len(authorityStringList); i++ {
		var authority Authority.Authority
		if !Authority.IsAuthority[authorityStringList[i]] {
			log.Println("Error parsing Authority")
			return user, errors.New("Invalid Authority for user")
		}
		authority = Authority.GetAuthority(authorityStringList[i])
		authorities = append(authorities, authority)
	}

	// Get Team KubeCluster List
	if userReq.TeamIdList != "" {
		teamIdStringList := strings.Split(userReq.TeamIdList, ",")
		var teamIdList []bson.ObjectId
		for i := 0; i < len(teamIdStringList); i++ {
			teamId := bson.ObjectIdHex(teamIdStringList[i])
			teamIdList = append(teamIdList, teamId)
		}
		user.TeamIdList = teamIdList
	}

	// If User Belongs To Any Organization Then User will have Organization KubeCluster
	if userReq.OrganizationId != "" {
		user.OrganizationId = bson.ObjectIdHex(userReq.OrganizationId)
	}

	if AccountType.IsAccountType[userReq.AccountType] {
		user.UserInfo.Company.AccountType = AccountType.GetAccountType(userReq.AccountType)
	}

	if userReq.AdminTeamList != "" {
		teamAdminIdStringList := strings.Split(userReq.AdminTeamList, ",")
		var adminTeamIdList []primitive.ObjectID
		for i := 0; i < len(teamAdminIdStringList); i++ {
			adminTeamId, err := primitive.ObjectIDFromHex(teamAdminIdStringList[i])
			if err == nil {
				adminTeamIdList = append(adminTeamIdList, adminTeamId)
			}

		}
		user.AdminTeamIdList = adminTeamIdList
	}

	user.BaseEntity.Id = userObjectId
	user.UserName = userReq.UserName
	user.Authorities = authorities
	user.IsEnabled = userReq.IsEnabled
	user.UserType = UserType.GetUserType(userReq.UserType)
	return user, nil
}

func GenerateJsonStringFromUser(user User) string {
	var jsonString string = "{" +
		"\"id\": \"" + user.Id.Hex() + "\"" +
		", \"username\": \"" + user.UserName + "\"" +
		", \"isEnabled\": " + strconv.FormatBool(user.IsEnabled) +
		", \"authorities\": " + "\"" + getAuthoritiesAsString(user) + "\"" +
		", \"companyId\": " + "\"" + getCompanyIdString(user) + "\"" +
		", \"organizationId\": " + "\"" + getOrganizationIdFromUserInfo(user) + "\"" +
		", \"teamIds\": " + "\"" + getTeamIdsFromUserInfo(user) + "\"" +
		", \"userType\": " + "\"" + getUserTypeAsString(user) + "\"" +
		"}"
	return jsonString
}

func getUserTypeAsString(user User) string {
	return UserType.GetUserTypeAsString[user.UserType]
}

func getCompanyIdString(user User) string {
	companyId := ""
	if user.CompanyId != "" {
		companyId = user.CompanyId.Hex()
	}
	return companyId
}

func getOrganizationIdFromUserInfo(user User) string {
	if user.OrganizationId != "" {
		return user.OrganizationId.Hex()
	}
	return ""
}

func getTeamIdsFromUserInfo(user User) string {
	var teamIdStr string = ""
	for i, teamId := range user.TeamIdList {
		if i < len(user.Authorities)-1 {
			teamIdStr = teamIdStr + teamId.Hex() + ","
		} else {
			teamIdStr = teamIdStr + teamId.Hex()
		}
	}
	return teamIdStr
}

func getAuthoritiesAsString(user User) string {
	var authoritiesStr string = ""
	for i, auth := range user.Authorities {
		if i < len(user.Authorities)-1 {
			authoritiesStr = authoritiesStr + Authority.GetAuthorityString[auth] + ","
		} else {
			authoritiesStr = authoritiesStr + Authority.GetAuthorityString[auth]
		}
	}
	return authoritiesStr
}
