// controllers/action.go

package controllers

import (
	webappv1 "citictel.com/vincentzou/vin-order/api/v1"
	"citictel.com/vincentzou/vin-order/service"
	"context"
	"fmt"
	"k8s.io/klog/v2"
	"log"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//Action 定义的执行动作接口
type Action interface {
	Execute(context.Context) error
}

//PatchStatus 用户更新对象 status 状态
type PatchStatus struct {
	client client.Client
	//	original runtime.Object
	//	new      runtime.Object
	original client.Object
	new      client.Object
	status   string
	reason   string
	msg      string
}

func (o *PatchStatus) Execute(ctx context.Context) error {
	if reflect.DeepEqual(o.original, o.new) {
		return nil
	}
	// update database order status
	order := o.original.(*webappv1.Order)
	log.Println("orderNo:", order.Spec.OrderNo)
	fmt.Println(order)

	err := service.UpdateOrder(order.Spec.OrderNo, o.status, o.reason, o.msg)
	if err != nil {
		klog.Info("update order status into mongodb with error ", err)
		return err
	}
	// 更新状态
	if err := o.client.Status().Patch(ctx, o.new, client.MergeFrom(o.original)); err != nil {
		return fmt.Errorf("while patching status error %q", err)
	}

	return nil
}

// CreateObject 创建一个新的资源对象
//type CreateObject struct {
//	client client.Client
//	obj    client.Object
//}
//
//func (o *CreateObject) Execute(ctx context.Context) error {
//	if err := o.client.Create(ctx, o.obj); err != nil {
//		return fmt.Errorf("error %q while creating object ", err)
//	}
//	return nil
//}
