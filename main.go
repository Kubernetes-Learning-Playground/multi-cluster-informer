package main

import (

	"multi_cluster_informer/pkg"
	"time"

	//"time"
	"fmt"
)

func main() {

	r, err := pkg.NewMultiClusterInformer(
		5,
		pkg.Cluster{
			ConfigPath:"/Users/zhenyu.jiang/go/src/golanglearning/new_project/multi_cluster_informer/resource/config1",
			MetaData: pkg.MetaData{
				ClusterName: "cluster1",
				List: []pkg.ResourceAndNamespace{
					//{pkg.Services, "default"},
					//{pkg.ConfigMaps, "default"},
					{pkg.Pods, "default"}, // 支持使用all 来监听所有命名空间的资源
					//{pkg.Deployments, "default"},
				},

			},
		},
		//pkg.Cluster{
		//	ConfigPath:"/Users/zhenyu.jiang/go/src/golanglearning/new_project/multi_cluster_informer/resource/config",
		//	MetaData: pkg.MetaData{
		//		ClusterName: "cluster2",
		//		List: []pkg.ResourceAndNamespace{
		//			{pkg.Services, "default"},
		//			{pkg.ConfigMaps, "default"},
		//			{pkg.Pods, "default"},
		//		},
		//
		//	},
		//},
	)
	if err != nil {
		panic(err)
	}

	// 执行informer监听
	go r.Run()

	for {
		obj, _ := r.Pop()
		// 如果自己的业务逻辑发生问题，可以重新放回队列。
		if err := process(obj); err != nil {
			_ = r.ReQueue(obj) // 重新入列
		} else { // 完成就结束
			r.Finish(obj)
		}

	}

	go r.Stop()
}

// 执行自己的业务逻辑
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

