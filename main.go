package main

import (
	"klovercloud/queue/config"
	v1 "klovercloud/queue/src/v1"
	"klovercloud/queue/src/v1/db"
	"klovercloud/queue/src/v1/logs"
)

func main() {
	srv := config.New()
	db.GetDmManager()
	//data, err := json.Marshal(rabbitmq.Cluster{})
	//if err != nil {
	//	fmt.Println(err)
	//}
	//fmt.Println(string(data))
	go logs.ConsumeLog()
	v1.Routes(srv)
	srv.Logger.Fatal(srv.Start(":" + config.ServerPort))

}

