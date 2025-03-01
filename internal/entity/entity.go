package entity

import "go.mongodb.org/mongo-driver/v2/bson"

type Student struct {
	ID           bson.ObjectID `bson:"_id"`
	GroupName    string        `bson:"groupname"`
	Login        string        `bson:"login"`
	Password     string        `bson:"hash"`
	Subscription bool          `bson:"subscription"`
}
