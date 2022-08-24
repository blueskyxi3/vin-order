package main

import (
	webappv1 "citictel.com/vincentzou/vin-order/api/v1"
	"context"
	"flag"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog"
	"path/filepath"
)

const gbManifest = `
apiVersion: webapp.github.com/v1
kind: Guestbook
metadata:
  name: guestbook-pc-003
spec:
  # TODO(user): Add fields here
  type: pipeline-pc-run
  orderNo: PC003
`

func main() {

	dynamicClient, _, err := initClient()
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	//设置要请求的 GVR
	gvr := schema.GroupVersionResource{
		Group:    "webapp.github.com",
		Version:  "v1",
		Resource: "guestbooks",
	}

	obj := &unstructured.Unstructured{}

	// decode YAML into unstructured.Unstructured
	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, gvk, err := dec.Decode([]byte(gbManifest), nil, obj)

	// Get the common metadata, and show GVK
	fmt.Println(obj.GetName(), gvk.String())

	// encode back to JSON
	//enc := json.NewEncoder(os.Stdout)
	//enc.SetIndent("", "    ")
	//enc.Encode(obj)
	unStructObj, err := dynamicClient.Resource(gvr).Namespace("default").Create(context.TODO(), obj, metav1.CreateOptions{})
	if err != nil {
		panic(any(err.Error()))
	}
	order := &webappv1.Order{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(
		unStructObj.UnstructuredContent(),
		order,
	)
	if err != nil {
		panic(any(err.Error()))
	}
	fmt.Printf("namespace: %v, name: %v\n", order.Namespace, order.Name)
	fmt.Println("create successfully!")

	// 发送请求，并得到返回结果
	unStructData, err := dynamicClient.Resource(gvr).Namespace("default").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(any(err.Error()))
	}

	// 使用反射将 unStructData 的数据转成对应的结构体类型，例如这是是转成 v1.PodList 类型
	// podList := &corev1.PodList{}
	guestbookList := &webappv1.OrderList{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(
		unStructData.UnstructuredContent(),
		guestbookList,
	)
	if err != nil {
		panic(any(err.Error()))
	}

	// 输出 guestbook 资源信息
	for _, item := range guestbookList.Items {
		fmt.Printf("namespace: %v, name: %v\n", item.Namespace, item.Name)
	}

}

func initClient() (dynamic.Interface, *rest.Config, error) {
	var err error
	var config *rest.Config
	// inCluster（Pod）、KubeConfig（kubectl）
	var kubeconfig *string

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(可选) kubeconfig 文件的绝对路径")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "kubeconfig 文件的绝对路径")
	}
	flag.Parse()

	// 首先使用 inCluster 模式(需要去配置对应的 RBAC 权限，默认的sa是default->是没有获取deployments的List权限)
	if config, err = rest.InClusterConfig(); err != nil {
		// 使用 KubeConfig 文件创建集群配置 Config 对象
		if config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig); err != nil {
			panic(any(err.Error()))
		}
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, config, err
	}

	return dynamicClient, config, nil
}
