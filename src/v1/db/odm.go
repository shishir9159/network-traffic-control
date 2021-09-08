package db

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"klovercloud/queue/config"
	"log"
	"sync"
)

type SortParam struct {
	SortBy string // bson field name
	Type   int    // -1 or 1
}

type Data interface{}


type dmManager struct {
	Ctx context.Context
	Db  *mongo.Database
}

var singletonDmManager *dmManager
var onceDmManager sync.Once

func GetDmManager() *dmManager {
	onceDmManager.Do(func() {
		log.Println("[INFO] Starting Initializing Singleton DB Manager")
		singletonDmManager = &dmManager{}
		singletonDmManager.initConnection()
	})
	return singletonDmManager
}

func (dm *dmManager) initConnection() {
	// Base context.
	ctx := context.Background()
	dm.Ctx = ctx
	clientOpts := options.Client().ApplyURI(config.DatabaseConnectionString)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		log.Println("[ERROR] DB Connection error:", err.Error())
		return
	}

	db := client.Database(config.DatabaseName)
	dm.Db = db

	log.Println("[INFO] Initialized Singleton DB Manager")
}


