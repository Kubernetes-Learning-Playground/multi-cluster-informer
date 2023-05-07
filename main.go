package main

import (
	"k8s.io/klog/v2"
	"multi_cluster_informer/pkg"
	"multi_cluster_informer/pkg/config"
	"time"

	//"time"
	"fmt"
)

func main() {
	// 1. 项目配置
	sysConfig, err := config.LoadConfig("./pkg/config/config.yaml")
	if err != nil {
		klog.Error("load config error: ", err)
		return
	}

	// 2. 启动多集群informer
	r, err := pkg.NewMultiClusterInformer(
		sysConfig.MaxReQueueTime,
		sysConfig.Clusters,
	)
	if err != nil {
		klog.Fatal("multi cluster informer err: ", err)
	}

	// 3. 执行informer监听
	go r.Run()
	defer r.Stop()

	// 4. 不断从队列取出资源对象
	for {
		obj, _ := r.Pop()
		// 如果自己的业务逻辑发生问题，可以重新放回队列。
		if err = process(obj); err != nil {
			_ = r.ReQueue(obj) // 重新入列
		} else { // 完成就结束
			r.Finish(obj)
		}
	}

}

// process 执行自己的业务逻辑
func process(obj pkg.QueueObject) error {

	// 判断只有add事件
	if obj.Event == pkg.EventAdd {
		fmt.Println("目前监听到事件为add的资源对象", obj.ResourceType)
	}
	//fmt.Println("目前监听到的资源对象", obj.ResourceType, obj.Event)
	//// 判断只有cluster2集群的事件
	if obj.ClusterName == "cluster2" {
		fmt.Println("目前监听到集群为cluster2的资源对象", obj.ResourceType)
	}

	fmt.Println(time.Now(), obj.Event, obj.ResourceType, obj.Key, obj.ClusterName)
	return nil
}
