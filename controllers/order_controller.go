/*
Copyright 2022 vicentzou.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	webappv1 "citictel.com/vincentzou/vin-order/api/v1"
	"citictel.com/vincentzou/vin-order/service"
	"context"
	"fmt"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
	"time"
)

// OrderReconciler reconciles a Order object
type OrderReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmap,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=webapp.citictel.com,resources=orders,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=webapp.citictel.com,resources=orders/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=webapp.citictel.com,resources=orders/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Order object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *OrderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("start to reconcile ----->", req.Name, req.NamespacedName)
	if strings.HasSuffix(req.Name, "-pod") {
		go r.TaskHandler(ctx, req)
		return ctrl.Result{}, nil
	}
	klog.Info("-------------------")
	var order webappv1.Order
	err := r.Get(ctx, req.NamespacedName, &order)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	klog.Info("fetch Order objects ", "Order ", order.Name, " ", order.Namespace)
	switch order.Status.Phase {
	case webappv1.OrderPhaseFailed, webappv1.OrderPhaseCompleted:
		klog.Infof("completed time is %v, it will remove order [%s] at %v. \n", order.Status.CompletionTime.Format("2006-01-02 15:04:05"), order.Name, order.Status.CompletionTime.Add(24*time.Hour).Format("2006-01-02 15:04:05"))
		r.Recorder.Event(&order, corev1.EventTypeNormal, "StatusChange", "start to set order status to in-progress")
		if order.Status.CompletionTime.Add(24 * time.Hour).Before(time.Now()) {
			r.Delete(ctx, &order)
			klog.Infof("remove order[%s] successfully in k8s !\n", req.Name)
		}
		return ctrl.Result{}, nil
	case webappv1.OrderPhaseInProgress:
		klog.Infof("enter order[%s] in-progress \n", req.Name)
		// check pr object
		// if exist return
		err, returnFlag := r.pipelineRunHandle(ctx, &order, req)
		if err != nil {
			klog.Info("check pipeline run object exception with error :", err.Error())
			return ctrl.Result{}, nil
		}
		if returnFlag {
			return ctrl.Result{}, nil
		}
		klog.Info("there is no pr object generated.")
		// else pass , ready to create
	}

	//--start---
	var action Action

	switch {
	case !order.DeletionTimestamp.IsZero():
		klog.Info("Order Object has been deleted. Ignoring.")
	case order.Status.Phase == "":
		klog.Info("Order Create Starting. Updating status.")
		newBackup := order.DeepCopy()
		newBackup.Status.Phase = webappv1.OrderPhaseInProgress
		now := metav1.Time{Time: time.Now()}
		newBackup.Status.StartTime = &now
		action = &PatchStatus{client: r.Client, original: &order, new: newBackup, status: "3"} // 下一步要执行的动作
		r.Recorder.Event(&order, corev1.EventTypeNormal, "StatusChange", "start to set order status to in-progress")
	case order.Status.Phase == webappv1.OrderPhaseInProgress: // 进行中
		klog.Info("enter into in-progress status...")
	}

	// 执行动作
	if action != nil {
		if err = action.Execute(ctx); err != nil {
			klog.Errorf("executing action error: %s\n", err)
			//	return ctrl.Result{}, fmt.Errorf("executing action error: %s", err)
		}
		//执行action，需返回
		return ctrl.Result{}, nil
	}
	//--end---

	// 检查DB是否已经有该表订单,如果已经存在，则直接返回
	klog.Info("start to compose pipelineRun!")
	pr, err := r.buildPipelineRun(ctx, order, req.Namespace)
	if err != nil {
		klog.Errorf("build PipelineRun Object error: %s\n", err)
		//return ctrl.Result{}, fmt.Errorf("build PipelineRun Object error: %s", err)
		return ctrl.Result{}, nil
	}
	//enc := json.NewEncoder(os.Stdout)
	//enc.SetIndent("", "    ")
	//enc.Encode(pr)
	klog.Info("start to create pipelineRun!")
	err = r.Client.Create(ctx, pr)
	if err != nil {
		klog.Info(err.Error())
		r.Recorder.Event(&order, corev1.EventTypeWarning, "CreateFailed", fmt.Sprintf("create pipelineRun[%s] failed ", pr.Name))
		return ctrl.Result{}, nil
	}
	r.Recorder.Event(&order, corev1.EventTypeNormal, "CreateSuccessful", fmt.Sprintf("create pipelineRun[%s] successfully ", pr.Name))
	klog.Info("create pipelineRun successfully!")
	return ctrl.Result{}, nil
}

// pipelineRunHandle  0: go to next code; 1: stop reconcile this time;
func (r *OrderReconciler) pipelineRunHandle(ctx context.Context, order *webappv1.Order, req ctrl.Request) (error, bool) {
	var prActual tektonv1beta1.PipelineRun
	objKey := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      fmt.Sprintf("%s-run", req.Name),
	}
	//err = r.Get(ctx, req.NamespacedName, &prActual)
	err := r.Get(ctx, objKey, &prActual)
	if err != nil {
		klog.Info("pipeline query with error :", err.Error())
		if client.IgnoreNotFound(err) != nil {
			return err, false
		}
	}
	klog.Info("fetch PipelineRun objects ", "PipelineRun ", prActual.Name, " ", prActual.Namespace)
	if !reflect.DeepEqual(prActual, tektonv1beta1.PipelineRun{}) {
		// encode back to JSON
		//enc := json.NewEncoder(os.Stdout)
		//enc.SetIndent("", "    ")
		//enc.Encode(prActual)
		if prActual.Status.Conditions == nil {
			klog.Info("current pr status is null, please waiting for a while to retry.. ")
			return nil, true
		}
		conditions := prActual.Status.Conditions
		condition := conditions[0]
		klog.Infof("condition : %v \n", condition.Reason)
		klog.Infof("pipeline [%s] in status of %s. ready to further check for pr status.", prActual.Name, condition.Reason)
		newBackup := order.DeepCopy()
		switch condition.Status {
		case corev1.ConditionTrue:
			newBackup.Status.Phase = webappv1.OrderPhaseCompleted
			newBackup.Status.CompletionTime = &condition.LastTransitionTime.Inner
			action := &PatchStatus{client: r.Client, original: order, new: newBackup, status: "1"} // 下一步要执行的动作
			err = action.Execute(ctx)
			//TODO need to update task's status into completed 1
			r.Recorder.Event(order, corev1.EventTypeNormal, condition.Reason, condition.Message)
			klog.Infof("the order %s is %s with message: %s", order.Name, condition.Reason, condition.Message)
			return err, true
		case corev1.ConditionFalse:
			newBackup.Status.Phase = webappv1.OrderPhaseFailed
			newBackup.Status.CompletionTime = &condition.LastTransitionTime.Inner
			action := &PatchStatus{client: r.Client, original: order, new: newBackup, status: "2", reason: condition.Reason, msg: condition.Message} // 下一步要执行的动作
			err = action.Execute(ctx)
			//TODO need to update task's status into failed 2
			r.Recorder.Event(order, corev1.EventTypeWarning, condition.Reason, condition.Message)
			klog.Infof("the order %s is %s with error: %s", order.Name, condition.Reason, condition.Message)
			return err, true
		case corev1.ConditionUnknown:
			klog.Infof("the order %s is %s with message: %s", order.Name, condition.Reason, condition.Message)
			return nil, true
		default:
			return nil, false
		}
	}
	return nil, false
}

//通过orderNo去取请求单的内容，并组装pipeline
func (r *OrderReconciler) buildPipelineRun(ctx context.Context, order webappv1.Order, namespace string) (*tektonv1beta1.PipelineRun, error) {
	// fetch config map from database
	//cmdb := &corev1.ConfigMap{}
	//nameSpacedName := types.NamespacedName{
	//	Name:      "cmdb",
	//	Namespace: namespace,
	//}
	//
	//if err := r.Client.Get(ctx, nameSpacedName, cmdb); err != nil && client.IgnoreNotFound(err) == nil {
	//	klog.Info("configmap[cmdb] can't be found in namespace ", namespace)
	//} else {
	//	klog.Info("configmap[cmdb] is found in namespace ", namespace)
	//	klog.Info("the cmdb data is ", cmdb.Data)
	//}

	var params []tektonv1beta1.Param
	orderInfo, err := service.GetOrderInfo(order.Spec.OrderNo)
	if err != nil {
		klog.Infof("get order %s info error with %s \n", order.Spec.OrderNo, err.Error())
		return nil, err
	}
	for k, v := range orderInfo {
		switch v.(type) {
		case int:
			param := tektonv1beta1.Param{
				Name: k,
				Value: tektonv1beta1.ArrayOrString{
					Type:      tektonv1beta1.ParamTypeString,
					StringVal: v.(string),
				},
			}
			params = append(params, param)
		case string:
			param := tektonv1beta1.Param{
				Name: k,
				Value: tektonv1beta1.ArrayOrString{
					Type:      tektonv1beta1.ParamTypeString,
					StringVal: v.(string),
				},
			}
			params = append(params, param)
		default:
			klog.Infof("ignore current key:%s value:%v\n", k, v)
		}
	}

	pipelineRun := &tektonv1beta1.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PipelineRun",
			APIVersion: "tekton.dev/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-run", order.Name),
			Namespace: order.Namespace,
		},
		Spec: tektonv1beta1.PipelineRunSpec{
			PipelineRef:        &tektonv1beta1.PipelineRef{Name: order.Spec.Type},
			ServiceAccountName: "tekton-build-sa",
			Params:             params,
			PodTemplate:        &tektonv1beta1.PodTemplate{SchedulerName: "tekton-scheduler"},
		},
	}
	klog.Infof("pipeline Run --> \n %#v \n", pipelineRun)
	// 配置 controller reference
	if err = controllerutil.SetControllerReference(&order, pipelineRun, r.Scheme); err != nil {
		return nil, fmt.Errorf("setting order controller reference error : %s", err)
	}

	return pipelineRun, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OrderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.Order{}).
		Owns(&tektonv1beta1.PipelineRun{}).
		Watches(&source.Kind{Type: &corev1.Pod{}},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForPods),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		Complete(r)
}

func (r *OrderReconciler) findObjectsForPods(pod client.Object) []reconcile.Request {
	klog.Infof("find pod-> \n %v-%v \n", pod.GetNamespace(), pod.GetName())
	labels := pod.GetLabels()
	klog.Infof("labels:%v\n", labels)
	_, ok := labels["tekton.dev/pipelineTask"]
	if !ok {
		klog.Infof("it can't find the task name, please check if it is in tekton pipeline \n")
		return nil
	}
	requests := make([]reconcile.Request, 1)

	requests[0] = reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      pod.GetName(),
			Namespace: pod.GetNamespace(),
		},
	}

	return requests
}
