package universal

import (
	"gopkg.in/mgo.v2/bson"
)

type Team struct {
	BaseEntity
	CompanyId    bson.ObjectId `json:"companyId"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Organization Organization  `json:"organization"`
}
