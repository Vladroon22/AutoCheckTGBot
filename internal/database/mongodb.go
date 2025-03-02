package database

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/Vladroon22/TG-Bot/internal/entity"
	"github.com/Vladroon22/TG-Bot/internal/utils"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

func ConnectToMongo(c context.Context) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(c, 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://" + os.Getenv("mongo")))
	if err != nil {
		return nil, errors.New("ошибка подключения к базе данных")
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, errors.New("ошибка пинга базы данных")
	}

	return client, nil
}

func Insert(c context.Context, stud *entity.Student) error {
	ctx, cancel := context.WithTimeout(c, 5*time.Second)
	defer cancel()

	client, err := ConnectToMongo(ctx)
	if err != nil {
		return err
	}
	defer client.Disconnect(ctx)

	filter := bson.D{{Key: "login", Value: stud.Login}}

	var existingDoc bson.M
	collection := client.Database("education").Collection("students")
	errFind := collection.FindOne(ctx, filter).Decode(&existingDoc)
	switch {
	case errFind != mongo.ErrNoDocuments:
		return errors.New("попробуйте ввести другие данные")
	case errFind != mongo.ErrNoDocuments && err != nil:
		return errors.New("ошибка базы данных")
	}

	if _, err := collection.InsertOne(ctx, stud); err != nil {
		return errors.New("ошибка загрузки данных. Попробуйте еще раз")
	}

	return nil
}

func UpdateStudentSub(c context.Context, st *entity.Student, status bool) error {
	ctx, cancel := context.WithTimeout(c, 5*time.Second)
	defer cancel()

	client, err := ConnectToMongo(ctx)
	if err != nil {
		return err
	}
	defer client.Disconnect(ctx)

	filter := bson.D{{Key: "_id", Value: st.ID}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "subscription", Value: status}}}}

	collection := client.Database("education").Collection("students")
	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return errors.New("ошибка обновления данных")
	}

	if result.MatchedCount == 0 {
		return errors.New("студент не найден")
	}

	if result.ModifiedCount == 0 {
		return errors.New("данные не были изменены")
	}

	return nil
}

func CheckEquillity(c context.Context, stud *entity.Student) (string, error) {
	ctx, cancel := context.WithTimeout(c, 7*time.Second)
	defer cancel()

	client, err := ConnectToMongo(ctx)
	if err != nil {
		return "", err
	}
	defer client.Disconnect(ctx)

	collection := client.Database("education").Collection("students")
	filter := bson.D{
		{Key: "groupname", Value: stud.GroupName},
		{Key: "login", Value: stud.Login},
	}

	var result bson.M
	if err := collection.FindOne(ctx, filter).Decode(&result); err != nil {
		if err == mongo.ErrNoDocuments {
			return "", errors.New("студент не найден")
		}
		return "", errors.New("ошибка поиска в базе данных")
	}

	Hash, ok := result["hash"]
	if !ok {
		return "", errors.New("неправильный пароль")
	}

	if !utils.CmpHashAndPass(Hash.(string), stud.Password) {
		return "", errors.New("неправильный пароль")
	}

	id, exist := result["_id"]
	if !exist {
		return "", errors.New("неизвестный id пользователя")
	}
	return id.(string), nil
}
