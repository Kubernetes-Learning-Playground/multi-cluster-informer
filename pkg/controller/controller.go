package controller

import (
	"context"
	"errors"
	"github.com/practice/multi_cluster_informer/pkg/queue"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"time"
)

// MultiClusterInformer 多集群informer的接口对象
type MultiClusterInformer interface {
	// Run 执行多集群的informer的方法
	Run()
	// Stop 停止informer
	Stop()
	// AddEventHandler 加入回调handler
	AddEventHandler(handler HandleFunc)
	// HandleObject 调用handler处理资源对象
	HandleObject(object queue.QueueObject) error
	// Queue 队列接口对象
	queue.Queue
	// Store 本地缓存接口对象
	queue.Store
}

// Controller 控制器
// 主要保存所有集群的客户端实例，并保存
type Controller struct {
	// 多客户端 list
	clients []*kubernetes.Clientset
	// 多个informer list
	Informers InformerList

	StopC chan struct{}

	HandleFunc HandleFunc
	// 一个工作队列: 多集群的所有资源都会放入此队列
	queue.Queue
	// 一个本地缓存：多集群的所有资源都会放入此缓存
	queue.Store
}

// 是否实现MultiClusterInformer接口
var _ MultiClusterInformer = &Controller{}

// initHandle 处理informer逻辑
func initHandle(resource string, worker queue.Queue, clusterName string) cache.ResourceEventHandlerFuncs {
	handler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				worker.Push(queue.QueueObject{ClusterName: clusterName, Event: queue.EventAdd, ResourceType: resource, Key: key, CreateAt: time.Now()})
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				// 放入
				worker.Push(queue.QueueObject{ClusterName: clusterName, Event: queue.EventUpdate, ResourceType: resource, Key: key, CreateAt: time.Now()})

			}
		},
		DeleteFunc: func(obj interface{}) {

			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				worker.Push(queue.QueueObject{ClusterName: clusterName, Event: queue.EventDelete, ResourceType: resource, Key: key, CreateAt: time.Now()})
			}
		},
	}
	return handler
}

// Run 执行informer
func (c *Controller) Run() {
	defer c.Queue.Close()
	c.Informers.run(c.StopC)
	<-c.StopC
}

// Stop 停止
func (c *Controller) Stop() {
	c.StopC <- struct{}{}
}

type HandleFunc func(object queue.QueueObject) error

func (c *Controller) AddEventHandler(handler HandleFunc) {
	c.HandleFunc = handler
}

func (c *Controller) HandleObject(obj queue.QueueObject) error {
	if c.HandleFunc != nil {
		err := c.HandleFunc(obj)
		return err
	}
	return nil
}

type InformerList []cache.Controller

func (s InformerList) run(done chan struct{}) {
	// 执行所有list 中的informer
	for _, one := range s {
		// 使用不同goroutine执行 informer
		go one.Run(done)

		if !cache.WaitForCacheSync(done, one.HasSynced) {
			panic("等待内部缓存超时")
		}
	}
}

// ResourceAndNamespace 资源与namespace
type ResourceAndNamespace struct {
	RType     string `json:"rType" yaml:"rType"`
	Namespace string `json:"namespace" yaml:"namespace"`
}

// MetaData 集群对象所需的信息
type MetaData struct {
	List        []ResourceAndNamespace `json:"list"cyaml:"list"`
	ClusterName string                 `json:"clusterName" yaml:"clusterName"`
}

// CreateCoreV1IndexInformer 构造informer需要的资源
func (r *ResourceAndNamespace) CreateCoreV1IndexInformer(client *kubernetes.Clientset, worker queue.Queue, clusterName string) (indexer cache.Indexer, informer cache.Controller) {
	lw := cache.NewListWatchFromClient(client.CoreV1().RESTClient(), r.RType, r.Namespace, fields.Everything())
	switch r.RType {
	case queue.Services:
		indexer, informer = cache.NewIndexerInformer(lw, &v1.Service{}, 0, initHandle(queue.Services, worker, clusterName), cache.Indexers{})
	case queue.Pods:
		indexer, informer = cache.NewIndexerInformer(lw, &v1.Pod{}, 0, initHandle(queue.Pods, worker, clusterName), cache.Indexers{})
	case queue.ConfigMaps:
		indexer, informer = cache.NewIndexerInformer(lw, &v1.ConfigMap{}, 0, initHandle(queue.ConfigMaps, worker, clusterName), cache.Indexers{})
	}
	return
}

// CreateAppsV1IndexInformer 构造informer需要的资源
func (r *ResourceAndNamespace) CreateAppsV1IndexInformer(client *kubernetes.Clientset, worker queue.Queue, clusterName string) (indexer cache.Indexer, informer cache.Controller) {
	lw := cache.NewListWatchFromClient(client.AppsV1().RESTClient(), r.RType, r.Namespace, fields.Everything())
	switch r.RType {
	case queue.Deployments:
		indexer, informer = cache.NewIndexerInformer(lw, &appsv1.Deployment{}, 0, initHandle(queue.Deployments, worker, clusterName), cache.Indexers{})
	}
	return
}

// CreateAllCoreV1IndexInformer 创建选项是 all namespace时的解决方法
func (r *ResourceAndNamespace) CreateAllCoreV1IndexInformer(client *kubernetes.Clientset, worker queue.Queue, clusterName string, isAll bool) ([]cache.Indexer, []cache.Controller, bool) {
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
		case queue.Services:
			indexer, informer := cache.NewIndexerInformer(lw, &v1.Service{}, 0, initHandle(queue.Services, worker, clusterName), cache.Indexers{})
			informerListRes = append(informerListRes, informer)
			indexerListRes = append(indexerListRes, indexer)
		case queue.Pods:
			indexer, informer := cache.NewIndexerInformer(lw, &v1.Pod{}, 0, initHandle(queue.Pods, worker, clusterName), cache.Indexers{})
			informerListRes = append(informerListRes, informer)
			indexerListRes = append(indexerListRes, indexer)
		case queue.ConfigMaps:
			indexer, informer := cache.NewIndexerInformer(lw, &v1.ConfigMap{}, 0, initHandle(queue.ConfigMaps, worker, clusterName), cache.Indexers{})
			informerListRes = append(informerListRes, informer)
			indexerListRes = append(indexerListRes, indexer)
		}

	}

	isAll = true

	return indexerListRes, informerListRes, isAll

}

// CreateAllAppsV1IndexInformer 创建选项是 all namespace时的解决方法
func (r *ResourceAndNamespace) CreateAllAppsV1IndexInformer(client *kubernetes.Clientset, worker queue.Queue, clusterName string, isAll bool) ([]cache.Indexer, []cache.Controller, bool) {
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
		case queue.Deployments:
			indexer, informer := cache.NewIndexerInformer(lw, &appsv1.Deployment{}, 0, initHandle(queue.Deployments, worker, clusterName), cache.Indexers{})
			informerListRes = append(informerListRes, informer)
			indexerListRes = append(indexerListRes, indexer)
		}
	}

	isAll = true

	return indexerListRes, informerListRes, isAll

}

// Cluster 集群对象
type Cluster struct {
	ConfigPath string   `json:"configPath" yaml:"configPath"` // kube config文件
	MetaData   MetaData `json:"metadata" yaml:"metadata"`
}

// NewClient 初始化client
func (c *Cluster) NewClient() (*kubernetes.Clientset, error) {

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