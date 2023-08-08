package main

import (
	"fmt"
	"github.com/practice/multi_cluster_informer/pkg"
	"github.com/practice/multi_cluster_informer/pkg/queue"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"time"
)

var (
	cfg string // 配置文件
)

func main() {

	//flag.StringVar(&cfg, "config", "./config.yaml", "config yaml file")
	//flag.Parse()
	//
	//// 1. 项目配置
	//sysConfig, err := config.LoadConfig(cfg)
	//if err != nil {
	// klog.Error("load config error: ", err)
	// return
	//}
	//
	//// 2. 启动多集群informer
	//r, err := multi_informer.NewMultiClusterInformer(
	// sysConfig.MaxReQueueTime,
	// sysConfig.Clusters,
	//)
	//if err != nil {
	// klog.Fatal("multi cluster informer err: ", err)
	//}

	// 推荐如下方式，调用者只需要关心配置文件中的设置即可
	// 1. 获取控制器对象
	r, err := multi_informer.NewMultiClusterInformerFromConfig("./config.yaml")
	if err != nil {
		klog.Fatal("multi cluster informer err: ", err)
	}
	// 2. 加入handler
	r.AddEventHandler(func(object queue.QueueObject) error {
		// 判断只有add事件
		if object.Event == queue.EventAdd {
			fmt.Println("目前监听到事件为add的资源对象", object.ResourceType)
		}
		//fmt.Println("目前监听到的资源对象", obj.ResourceType, obj.Event)
		//// 判断只有cluster2集群的事件
		if object.ClusterName == "cluster1" {
			fmt.Println("目前监听到集群为cluster1的资源对象", object.ResourceType)
		}

		if object.ClusterName == "cluster2" {
			fmt.Println("目前监听到集群为cluster2的资源对象", object.ResourceType)
		}

		fmt.Println(time.Now(), object.Event, object.ResourceType, object.Key, object.ClusterName)
		if object.ResourceType == queue.Pods {
			if object.Obj != nil {
				pp := object.Obj.(*v1.Pod)
				fmt.Println("名字！！", pp.Name)
			}
		}
		return nil
	})

	// 3. 执行informer监听
	go r.Run()
	defer r.Stop()

	// 4. 不断从队列取出资源对象
	for {
		obj, _ := r.Pop()
		// 方法一：使用handler
		// 如果自己的业务逻辑发生问题，可以重新放回队列。
		if err = r.HandleObject(obj); err != nil {
			_ = r.ReQueue(obj) // 重新入列
		} else { // 完成就结束
			r.Finish(obj)
		}

		// 方法二：自行处理
		//if err = process(obj); err != nil {
		// _ = r.ReQueue(obj) // 重新入列
		//} else { // 完成就结束
		// r.Finish(obj)
		//}
	}

}

// process 执行自己的业务逻辑
func process(obj queue.QueueObject) error {

	// 判断只有add事件
	if obj.Event == queue.EventAdd {
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
