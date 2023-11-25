package multi_informer

import (
	"github.com/practice/multi_cluster_informer/pkg/config"
	"github.com/practice/multi_cluster_informer/pkg/controller"
	"github.com/practice/multi_cluster_informer/pkg/queue"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

// NewMultiClusterInformerFromConfig 输入配置文件目录，返回 MultiClusterInformer 对象
// 推荐调用者直接使用此方法初始化对象
func NewMultiClusterInformerFromConfig(path string) (controller.MultiClusterInformer, error) {

	sysConfig, err := config.LoadConfig(path)
	if err != nil {
		klog.Error("load config error: ", err)
		return nil, err
	}

	return NewMultiClusterInformer(sysConfig.MaxReQueueTime, sysConfig.Clusters)
}

// NewMultiClusterInformer 入参：最大重回对列次数、集群对象列表
func NewMultiClusterInformer(maxReQueueTime int, clusters []controller.Cluster) (controller.MultiClusterInformer, error) {
	core := &controller.Controller{
		Queue: queue.NewWorkQueue(maxReQueueTime),
		StopC: make(chan struct{}, 1),
	}

	store := make(queue.MapIndexers)
	informers := make(controller.InformerList, 0)

	// 遍历所有集群，并初始化
	for _, c := range clusters {
		client, err := c.NewClient()
		if err != nil {
			return nil, err
		}
		// 遍历所有资源，建立 indexer
		for _, r := range c.MetaData.List {

			// 当 namespace 为all时 单独处理
			if r.Namespace == queue.All {
				var indexerListRes []cache.Indexer
				var informerListRes []cache.Controller
				switch r.RType {
				case queue.Deployments:
					indexerListRes, informerListRes = r.CreateAllAppsV1IndexInformer(client, core.Queue, c.MetaData.ClusterName)
				case queue.Statefulsets:
					indexerListRes, informerListRes = r.CreateAllAppsV1IndexInformer(client, core.Queue, c.MetaData.ClusterName)
				case queue.Daemonsets:
					indexerListRes, informerListRes = r.CreateAllAppsV1IndexInformer(client, core.Queue, c.MetaData.ClusterName)
				case queue.Pods:
					indexerListRes, informerListRes = r.CreateAllCoreV1IndexInformer(client, core.Queue, c.MetaData.ClusterName)
				case queue.ConfigMaps:
					indexerListRes, informerListRes = r.CreateAllCoreV1IndexInformer(client, core.Queue, c.MetaData.ClusterName)
				case queue.Secrets:
					indexerListRes, informerListRes = r.CreateAllCoreV1IndexInformer(client, core.Queue, c.MetaData.ClusterName)
				case queue.Services:
					indexerListRes, informerListRes = r.CreateAllCoreV1IndexInformer(client, core.Queue, c.MetaData.ClusterName)
				case queue.Events:
					indexerListRes, informerListRes = r.CreateAllCoreV1IndexInformer(client, core.Queue, c.MetaData.ClusterName)
				}

				for k, v := range indexerListRes {
					if v != nil || informerListRes[k] != nil {
						store[r.RType] = append(store[r.RType], v)
						informers = append(informers, informerListRes[k])
					}

				}

			} else {
				var indexer cache.Indexer
				var informer cache.Controller

				switch r.RType {
				case queue.Deployments:
					indexer, informer = r.CreateAppsV1IndexInformer(client, core.Queue, c.MetaData.ClusterName)
				case queue.Statefulsets:
					indexer, informer = r.CreateAppsV1IndexInformer(client, core.Queue, c.MetaData.ClusterName)
				case queue.Daemonsets:
					indexer, informer = r.CreateAppsV1IndexInformer(client, core.Queue, c.MetaData.ClusterName)
				case queue.Pods:
					indexer, informer = r.CreateCoreV1IndexInformer(client, core.Queue, c.MetaData.ClusterName)
				case queue.ConfigMaps:
					indexer, informer = r.CreateCoreV1IndexInformer(client, core.Queue, c.MetaData.ClusterName)
				case queue.Secrets:
					indexer, informer = r.CreateCoreV1IndexInformer(client, core.Queue, c.MetaData.ClusterName)
				case queue.Services:
					indexer, informer = r.CreateCoreV1IndexInformer(client, core.Queue, c.MetaData.ClusterName)
				case queue.Events:
					indexer, informer = r.CreateCoreV1IndexInformer(client, core.Queue, c.MetaData.ClusterName)
				}

				// 放入 list中
				store[r.RType] = append(store[r.RType], indexer)
				informers = append(informers, informer)

			}

		}
	}

	core.Informers = informers
	core.Store = store

	return core, nil
}
