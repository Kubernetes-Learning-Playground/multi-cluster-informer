package pkg

import (
	"context"
	"errors"
	appsv1 "k8s.io/api/apps/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"time"

	v1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

// MultiClusterInformer 多集群informer的接口对象
type MultiClusterInformer interface {

	// 执行多集群的informer的方法
	Run()

	// 停止informer
	Stop()

	queue

	store
}

// 主要的控制器
type controller struct {
	// 多客户端 list
	clients   []*kubernetes.Clientset
	// 多个informer list
	informers informerList

	stop chan struct{}

	// 一个工作队列: 多集群的所有资源都会放入此队列
	queue
	// 一个本地缓存：多集群的所有资源都会放入此缓存
	store
}

// 是否实现MultiClusterInformer接口
var _ MultiClusterInformer = &controller{}

// NewMultiClusterInformer 入参：最大重回对列次数、集群对象列表
func NewMultiClusterInformer(maxReQueueTime int, clusters ...Cluster) (MultiClusterInformer, error) {
	core := &controller{
		queue:   newWorkQueue(maxReQueueTime),
		stop:    make(chan struct{}, 1),
	}

	store := make(mapIndexers)
	informers := make(informerList, 0)

	// 遍历所有集群，并初始化
	for _, c := range clusters {
		client, err := c.newClient()
		if err != nil {
			return nil, err
		}
		// 遍历所有资源，建立indexer
		for _, r := range c.MetaData.List {

			// 当namespace 为all时 单独处理
			if r.Namespace == "all" {
				if r.RType == Deployments {

					indexerListRes, informerListRes, isAll := r.createAllAppsV1IndexInformer(client, core.queue, c.MetaData.ClusterName, false)

					// 全部放入
					if isAll == true {
						for k, v := range indexerListRes {
							store[r.RType] = append(store[r.RType], v)
							informers = append(informers, informerListRes[k])
						}
					}

				} else {

					indexerListRes, informerListRes, isAll := r.createAllCoreV1IndexInformer(client, core.queue, c.MetaData.ClusterName, false)

					if isAll == true {
						for k, v := range indexerListRes {
							store[r.RType] = append(store[r.RType], v)
							informers = append(informers, informerListRes[k])
						}
					}

				}


			} else {

				var indexer cache.Indexer
				var informer cache.Controller
				if r.RType == Deployments {
					indexer, informer = r.createAppsV1IndexInformer(client, core.queue, c.MetaData.ClusterName)
				} else {
					indexer, informer = r.createCoreV1IndexInformer(client, core.queue, c.MetaData.ClusterName)
				}

				// 放入 list中
				store[r.RType] = append(store[r.RType], indexer)
				informers = append(informers, informer)

			}



		}
	}

	core.informers = informers
	core.store = store

	return core, nil
}

// 处理informer逻辑
func initHandle(resource string, worker queue, clusterName string) cache.ResourceEventHandlerFuncs {
	handler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				worker.push(QueueObject{clusterName, EventAdd, resource, key, time.Now()})
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				// 放入
				worker.push(QueueObject{clusterName,EventUpdate, resource, key, time.Now()})

			}
		},
		DeleteFunc: func(obj interface{}) {

			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				worker.push(QueueObject{clusterName,EventDelete, resource, key, time.Now()})
			}
		},
	}
	return handler
}


func (c *controller) Run() {
	defer c.queue.close()

	c.informers.run(c.stop)

	<-c.stop
}

// 停止
func (c *controller) Stop() {
	c.stop <- struct{}{}
}

type informerList []cache.Controller

func (s informerList) run(done chan struct{}) {
	// 执行所有list 中的informer
	for _, one := range s {
		// 使用不同goroutine执行 informer
		go one.Run(done)

		if !cache.WaitForCacheSync(done, one.HasSynced) {
			panic("等待内部缓存超时")
		}
	}
}

// 资源与namespace
type ResourceAndNamespace struct {
	RType          string
	Namespace      string
}

// MetaData 集群对象所需的信息
type MetaData struct {
	List []ResourceAndNamespace
	ClusterName string
}

// 构造informer需要的资源
func (r *ResourceAndNamespace) createCoreV1IndexInformer(client *kubernetes.Clientset, worker queue, clusterName string) (indexer cache.Indexer, informer cache.Controller)  {
	lw := cache.NewListWatchFromClient(client.CoreV1().RESTClient(), r.RType, r.Namespace, fields.Everything())
	switch r.RType {
	case Services:
		indexer, informer = cache.NewIndexerInformer(lw, &v1.Service{}, 0, initHandle(Services, worker, clusterName), cache.Indexers{})
	case Pods:
		indexer, informer = cache.NewIndexerInformer(lw, &v1.Pod{}, 0, initHandle(Pods, worker, clusterName), cache.Indexers{})
	case ConfigMaps:
		indexer, informer = cache.NewIndexerInformer(lw, &v1.ConfigMap{}, 0, initHandle(ConfigMaps, worker, clusterName), cache.Indexers{})
	}
	return
}

// 构造informer需要的资源
func (r *ResourceAndNamespace) createAppsV1IndexInformer(client *kubernetes.Clientset, worker queue, clusterName string) (indexer cache.Indexer, informer cache.Controller)  {
	lw := cache.NewListWatchFromClient(client.AppsV1().RESTClient(), r.RType, r.Namespace, fields.Everything())
	switch r.RType {
	case Deployments:
		indexer, informer = cache.NewIndexerInformer(lw, &appsv1.Deployment{}, 0, initHandle(Deployments, worker, clusterName), cache.Indexers{})
	}
	return
}

// 创建选项是 all namespace时的解决方法
func (r *ResourceAndNamespace) createAllCoreV1IndexInformer(client *kubernetes.Clientset, worker queue, clusterName string, isAll bool) ([]cache.Indexer , []cache.Controller, bool){
	// 1. 先查一下所有ns
	nsList, err := client.CoreV1().Namespaces().List(context.Background(), v12.ListOptions{})
	if err != nil {
		return nil, nil, false
	}
	// 2. 遍历所有ns，并创建informer，存入list中，返回
	var indexerListRes = make([]cache.Indexer, 0)
	var informerListRes = make([]cache.Controller, 0)
	// 让所有ns都初始化indexers informer
	for _, v := range nsList.Items {
		klog.Info("namespace: ", v.Name, v.Namespace)
		lw := cache.NewListWatchFromClient(client.CoreV1().RESTClient(), r.RType, v.Namespace, fields.Everything())
		switch r.RType {
		case Services:
			indexer, informer := cache.NewIndexerInformer(lw, &v1.Service{}, 0, initHandle(Services, worker, clusterName), cache.Indexers{})
			informerListRes = append(informerListRes, informer)
			indexerListRes = append(indexerListRes, indexer)
		case Pods:
			indexer, informer := cache.NewIndexerInformer(lw, &v1.Pod{}, 0, initHandle(Pods, worker, clusterName), cache.Indexers{})
			informerListRes = append(informerListRes, informer)
			indexerListRes = append(indexerListRes, indexer)
		case ConfigMaps:
			indexer, informer := cache.NewIndexerInformer(lw, &v1.ConfigMap{}, 0, initHandle(ConfigMaps, worker, clusterName), cache.Indexers{})
			informerListRes = append(informerListRes, informer)
			indexerListRes = append(indexerListRes, indexer)
		}


	}

	isAll = true

	return indexerListRes, informerListRes, isAll


}

// 创建选项是 all namespace时的解决方法
func (r *ResourceAndNamespace) createAllAppsV1IndexInformer(client *kubernetes.Clientset, worker queue, clusterName string, isAll bool) ([]cache.Indexer , []cache.Controller, bool){
	// 1. 先查一下所有ns
	nsList, err := client.CoreV1().Namespaces().List(context.Background(), v12.ListOptions{})
	if err != nil {
		return nil, nil, false
	}
	// 2. 遍历所有ns，并创建informer，存入list中，返回
	var indexerListRes = make([]cache.Indexer, 0)
	var informerListRes = make([]cache.Controller, 0)
	// 让所有ns都初始化indexers informer
	for _, v := range nsList.Items {
		klog.Info("namespace: ", v.Name, v.Namespace)
		lw := cache.NewListWatchFromClient(client.AppsV1().RESTClient(), r.RType, v.Namespace, fields.Everything())
		switch r.RType {
		case Deployments:
			indexer, informer := cache.NewIndexerInformer(lw, &appsv1.Deployment{}, 0, initHandle(Deployments, worker, clusterName), cache.Indexers{})
			informerListRes = append(informerListRes, informer)
			indexerListRes = append(indexerListRes, indexer)
		}
	}

	isAll = true

	return indexerListRes, informerListRes, isAll


}

// Cluster 集群对象
type Cluster struct {
	ConfigPath      string // kube config文件
	MetaData 		MetaData
}

// 初始化client
func (c *Cluster) newClient() (*kubernetes.Clientset, error) {

	if c.ConfigPath != "" {
		config, err := clientcmd.BuildConfigFromFlags("", c.ConfigPath)
		config.Insecure = true
		if err != nil {
			return nil, err
		}
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			return nil, err
		}
		return clientset, nil
	}
	return nil, errors.New("无法找到集群client端")
}

