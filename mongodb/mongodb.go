package mongodb

import (
	"citictel.com/vincentzou/vin-order/settings"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

var Client *mongo.Client
var DBName string

func Init(cfg *settings.MongodbConfig) error {
	ctx := context.TODO()
	dsn := fmt.Sprintf("mongodb://%s:%s@%s:%s", cfg.Username, cfg.Password, cfg.Url, cfg.Port)
	//mongodb://root:root123@localhost:27017
	log.Println("dsn:", dsn)
	// 设置客户端连接配置
	clientOptions := options.Client().ApplyURI(dsn)
	// 连接到MongoDB
	var err error
	Client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Println(err)
		return err
	}
	// 检查连接
	err = Client.Ping(ctx, nil)
	if err != nil {
		log.Println(err)
		return err
	}
	DBName = cfg.Db
	log.Println("Connected to MongoDB successfully !")
	return nil
}

func Close() {
	ctx := context.TODO()
	err := Client.Disconnect(ctx)
	if err != nil {
		log.Fatal("closed mongodb exception with error ", err.Error())
	}
}
