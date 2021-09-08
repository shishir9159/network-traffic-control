package universal

import (
	"klovercloud/queue/src/v1/adapter/enum/AccountType"
)

type Company struct {
	BaseEntity    `bson:",inline"`
	Name          string                  `json:"name"`
	BusinessEmail string                  `json:"businessEmail"`
	AccountType   AccountType.AccountType `json:"accountType"`
	Website       string                  `json:"website"`
	Address       string                  `json:"address"`
	PhoneNo       string                  `json:"phoneNo"`
	Enabled       bool                    `json:"enabled"`
	GitInfo       GitInfo                 `json:"gitInfo"`
	DockerHubInfo DockerHubInfo           `json:"dockerHubInfo"`
}
