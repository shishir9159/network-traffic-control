package dto

type UserHeaderRequestDTO struct {
	Id             string `json:"id"`
	UserName       string `json:"username"`
	CompanyId      string `json:"companyId"`
	OrganizationId string `json:"organizationId"`
	TeamIdList     string `json:"teamIds"`
	IsEnabled      bool   `json:"isEnabled"`
	Authorities    string `json:"authorities"`
	UserType       string `json:"userType"`
	AdminTeamList  string `json:"adminTeamList"`
	AccountType    string `json:"accountType"`
}
