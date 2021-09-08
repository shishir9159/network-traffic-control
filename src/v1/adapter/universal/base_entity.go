package universal

import (
	"gopkg.in/mgo.v2/bson"
	"klovercloud/queue/src/v1/adapter/enum/Status"
)

type BaseEntity struct {
	Id                 bson.ObjectId  `bson:"_id,omitempty" json:"_id"`
	Status             Status.Status  `default:"V"  json:"status"`
	CreateDate         string         `json:"createDate"`
	UpdateDate         string         `json:"updateDate"`
	CreatedBy          string         `json:"createdBy"`
	UpdatedBy          string         `json:"updatedBy"`
	CreateActionId     *bson.ObjectId `json:"createActionId"`
	LastUpdateActionId *bson.ObjectId `json:"lastUpdateActionId"`
}
