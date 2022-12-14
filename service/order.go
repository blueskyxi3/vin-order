package service

import (
	"citictel.com/vincentzou/vin-order/mongodb"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"k8s.io/klog/v2"
	"log"
	"time"
)

var ignoreKeys = []string{"_id", "createdBy", "createdTime", "updatedTime", "completedTime", "status", "wfType"}

func UpdateOrder(orderNo, status, reason, msg string) error {
	keyStr := "completedTime"
	if status == "3" {
		keyStr = "updatedTime"
	}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: status}, {Key: keyStr, Value: time.Now()},
		{Key: "reason", Value: reason}, {Key: "msg", Value: msg}}}}
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
func GetConfigInfo(wfType string) (data map[string]interface{}, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := mongodb.Client
	collection := client.Database(mongodb.DBName).Collection("conf")

	filter := bson.D{{"wfType", wfType}}
	cur, err := collection.Find(ctx, filter)
	defer cur.Close(ctx)
	if err != nil {
		log.Println("find wfType ", wfType, " with error ", err.Error())
		return nil, err
	}
	var result bson.M
	for cur.Next(context.TODO()) {
		if err := cur.Decode(&result); err != nil {
			panic(err)
		}
		fmt.Println(result)
	}
	if err = cur.Err(); err != nil {
		panic(err)
	}
	return result, nil
}
func GetOrderInfo(orderNo string, configmap primitive.M) (data map[string]interface{}, err error) {
	var ignoreFields []string
	ignoreFields = append(ignoreFields, ignoreKeys...)
	params := configmap["params"]
	for i, v := range params.(primitive.A) {
		fmt.Printf("i: %d v: %v \n ", i, v)
		m := v.(primitive.M)
		source, ok := m["source"]
		klog.Infof("source-%v\n", source)
		if ok {
			if source == "custom" {
				ignoreFields = append(ignoreFields, m["name"].(string))
			}
		}
	}
	klog.Infof("ignoreFileds %v \n", ignoreFields)

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
		for _, key := range ignoreFields {
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

func GetTaskInfo(orderNo, taskName string) (data primitive.M, err error) {
	client := mongodb.Client
	collection := client.Database(mongodb.DBName).Collection("order")

	validTaskFilter := bson.E{Key: "tasks", Value: bson.D{
		{Key: "$elemMatch", Value: bson.D{
			{Key: "taskName", Value: taskName},
			{Key: "valid", Value: 1},
		}},
	}}
	filter := bson.D{bson.E{Key: "orderNo", Value: orderNo}}
	filter = append(filter, validTaskFilter)

	result, err := collection.Distinct(context.TODO(), "tasks", filter)
	for _, item := range result {
		m := item.(primitive.D)
		m2 := m.Map()
		_taskName := m2["taskName"].(string)
		_valid := m2["valid"].(int32)
		if _taskName == _taskName && _valid == 1 {
			return m.Map(), nil
		}
	}
	return nil, nil
}
