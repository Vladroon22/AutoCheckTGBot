package entity

type Student struct {
	ID           string `bson:"_id"`
	GroupName    string `bson:"groupname"`
	Login        string `bson:"login"`
	Password     string `bson:"hash"`
	Subscription bool   `bson:"subscription"`
}
