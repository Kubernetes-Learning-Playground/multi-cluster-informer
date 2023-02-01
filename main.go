package main

import (
	"fmt"
	"golanglearning/new_project/multi_cluster_informer/pkg"
	"time"
)

func main() {
	r, err := pkg.NewMultiClusterInformer(
		5,
		pkg.Cluster{
			ConfigPath:"/Users/zhenyu.jiang/go/src/golanglearning/new_project/multi_cluster_informer/resource/config1",
			Resources:[]pkg.ResourceAndNamespace{
				{pkg.Services, "default"},
				{pkg.ConfigMaps, "default"},
				{pkg.Pods, "default"},
			},
		},
		pkg.Cluster{
			ConfigPath:"/Users/zhenyu.jiang/go/src/golanglearning/new_project/multi_cluster_informer/resource/config",
			Resources:[]pkg.ResourceAndNamespace{
				{pkg.Services, "istio-system"},
				{pkg.Pods, "istio-system"},
				{pkg.Pods, "default"},
				{pkg.Deployments, "default"},
			},
		},
	)
	if err != nil {
		panic(err)
	}

	// 执行informer监听
	go r.Run()

	for {
		obj, _ := r.Pop()
		if err := process(obj); err != nil {
			// r.GetByKey()
			// r.ListKeys()
			r.ReQueue(obj) // 重新入列
		} else {
			r.Finish(obj)
		}

	}

	//go r.Stop()
}

// 执行自己的业务逻辑
func process(obj pkg.QueueObject) error {
	// your own logic
	fmt.Println(time.Now(), obj.Event, obj.ResourceType, obj.Key)
	return nil
}

