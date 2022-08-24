package service

import (
	"citictel.com/vincentzou/vin-order/mongodb"
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"k8s.io/klog/v2"
	"log"
	"time"
)

var ignoreKeys = []string{"_id", "createdBy", "createdTime", "updatedTime", "completedTime", "status", "wfType"}

func UpdateOrder(orderNo, status string) error {
	keyStr := "completedTime"
	if status == "3" {
		keyStr = "updatedTime"
	}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: status}, {Key: keyStr, Value: time.Now()}}}}
	filter := bson.D{{Key: "orderNo", Value: orderNo}}
	return updateOrder(context.TODO(), filter, update)
}

func updateOrder(ctx context.Context, filter bson.D, update bson.D) error {
	client := mongodb.Client
	c := client.Database(mongodb.DBName).Collection("order")
	ur, err := c.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Println(err)
		return err
	}
	log.Printf("ur.ModifiedCount: %v\n", ur.ModifiedCount)
	return nil
}

func GetOrderInfo(orderNo string) (data map[string]interface{}, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := mongodb.Client
	collection := client.Database(mongodb.DBName).Collection("order")

	filter := bson.D{{Key: "orderNo", Value: orderNo}}
	cur, err := collection.Find(ctx, filter)
	if err != nil {
		klog.Infof("find order %s with error %s ", orderNo, err.Error())
		return nil, err
	}
	for cur.Next(ctx) {
		var result bson.D
		err = cur.Decode(&result)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		data = result.Map()
		//log.Printf("remove sid result: %v\n", res)
		for _, key := range ignoreKeys {
			delete(data, key)
		}
		klog.Info(data)
		break
	}
	if err = cur.Err(); err != nil {
		klog.Info(err)
		return nil, err
	}
	return data, nil
}
