package logs
type KafkaStruct struct {
	Header interface{}                 `json:"header"`
	Body   Log `json:"body"`

}
type Log struct {
	AppId  string     `json:"appId" bson:"appId"`
	Order int  `json:"order" bson:"order"`
	Log string `json:"log" bson:"log"`
	Type string `json:"type" bson:"type"`
}