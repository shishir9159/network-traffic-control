package universal

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserInfo struct {
	BaseEntity
	UserId       primitive.ObjectID `json:"userId"`
	Email        string             `json:"email"`
	FirstName    string             `json:"firstName"`
	LastName     string             `json:"lastName"`
	Company      Company            `json:"company"`
	Organization Organization       `json:"organization"`
	Team         []Team             `json:"teams"`
}
