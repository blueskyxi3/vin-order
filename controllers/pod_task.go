package controllers

import (
	"citictel.com/vincentzou/vin-order/mongodb"
	"citictel.com/vincentzou/vin-order/service"
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

// TaskHandler such as task completed
func (r *OrderReconciler) TaskHandler(ctx context.Context, req ctrl.Request) error {
	var pod corev1.Pod
	err := r.Get(ctx, req.NamespacedName, &pod)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		return nil
	}
	index := strings.Index(pod.Name, "-")
	orderNo := pod.Name[0:index]
	klog.Infof("orderNo %s ", orderNo)
	labels := pod.ObjectMeta.Labels
	taskName, ok := labels["tekton.dev/pipelineTask"]
	if !ok {
		klog.Info("pipeline task is ", taskName)
		return nil
	}
	task, err := service.GetTaskInfo(orderNo, taskName)
	if err != nil {
		klog.Errorf("%v\n", err)
		return err
	}
	klog.Infof("task \n%v \n", task)
	if pod.Status.Phase == corev1.PodFailed {
		if task["taskStatus"] != 2 {
			UpdateTaskStatus(orderNo, taskName, 2)
		}
	} else if pod.Status.Phase == corev1.PodSucceeded {
		if task["taskStatus"] != 1 {
			UpdateTaskStatus(orderNo, taskName, 1)
		}
	}
	return nil
}

func UpdateTaskStatus(orderNo, taskName string, taskStatus int) (bool, error) {
	arrayFilters := options.ArrayFilters{Filters: bson.A{bson.M{"x.valid": 1, "x.taskName": taskName}}}
	filter := bson.M{"orderNo": orderNo}
	upsert := false
	opts := options.UpdateOptions{
		ArrayFilters: &arrayFilters,
		Upsert:       &upsert,
	}
	update := bson.M{
		"$set": bson.M{
			"tasks.$[x].completedTime": time.Now(),
			"tasks.$[x].taskStatus":    taskStatus,
		},
	}
	client := mongodb.Client
	collection := client.Database(mongodb.DBName).Collection("order")
	ret, err := collection.UpdateOne(context.TODO(), filter, update, &opts)
	if err != nil {
		klog.Infof("update order %s failed with error %v \n", orderNo, err)
		return false, err
	}
	klog.Infof("%#v\n", ret)
	if ret.ModifiedCount == 0 {
		return true, nil
	}
	return false, nil
}
