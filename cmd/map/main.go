package main

import "k8s.io/klog/v2"

func main() {
	maptest := map[string]interface{}{}
	maptest["k1"] = "v1"
	maptest["k2"] = "v2"
	klog.Info(maptest)
	delete(maptest, "k1")
	klog.Info(maptest)
	delete(maptest, "k3")
	klog.Info(maptest)
	for k, v := range maptest {
		klog.Info(k, v)
	}

}
