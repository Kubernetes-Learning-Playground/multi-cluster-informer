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
	// clients 多客户端 list
	clients []*kubernetes.Clientset
	// Informers 多个informer list
	Informers InformerList
	// HandleFunc 统一的handle方法
	HandleFunc HandleFunc
	// Queue 一个工作队列: 多集群的所有资源都会放入此队列
	queue.Queue
	// Store 一个本地缓存：多集群的所有资源都会放入此缓存
	queue.Store
	// stop chan
	StopC chan struct{}
}

// 是否实现MultiClusterInformer接口
var _ MultiClusterInformer = &Controller{}

// initHandle 处理informer逻辑
// 执行的逻辑：当监听到新增、修改、删除事件时，放入工作队列中
func initHandle(resource string, worker queue.Queue, clusterName string, isObjSave bool) cache.ResourceEventHandlerFuncs {
	handler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				qo := queue.QueueObject{ClusterName: clusterName, Event: queue.EventAdd, ResourceType: resource, Key: key, CreateAt: time.Now()}
				if isObjSave {
					qo.Obj = obj
				}
				worker.Push(qo)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				qo := queue.QueueObject{ClusterName: clusterName, Event: queue.EventUpdate, ResourceType: resource, Key: key, CreateAt: time.Now()}
				if isObjSave {
					qo.Obj = new
				}
				// 放入
				worker.Push(qo)

			}
		},
		DeleteFunc: func(obj interface{}) {

			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				qo := queue.QueueObject{ClusterName: clusterName, Event: queue.EventDelete, ResourceType: resource, Key: key, CreateAt: time.Now()}
				if isObjSave {
					qo.Obj = obj
				}
				worker.Push(qo)
			}
		},
	}
	return handler
}

// Run 执行informer
func (c *Controller) Run() {
	klog.Info("run controller...")
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
	ObjSave   bool   `json:"objSave" yaml:"objSave"`
}

// MetaData 集群对象所需的信息
type MetaData struct {
	List        []ResourceAndNamespace `json:"list" yaml:"list"`
	ConfigPath  string                 `json:"configPath" yaml:"configPath"` // kube config文件
	Insecure    bool                   `json:"insecure" yaml:"insecure"`     // 是否跳过证书认证
	ClusterName string                 `json:"clusterName" yaml:"clusterName"`
}

// CreateCoreV1IndexInformer 构造informer需要的资源
func (r *ResourceAndNamespace) CreateCoreV1IndexInformer(client *kubernetes.Clientset, worker queue.Queue, clusterName string) (indexer cache.Indexer, informer cache.Controller) {
	lw := cache.NewListWatchFromClient(client.CoreV1().RESTClient(), r.RType, r.Namespace, fields.Everything())
	switch r.RType {
	case queue.Services:
		indexer, informer = cache.NewIndexerInformer(lw, &v1.Service{}, 0, initHandle(queue.Services, worker, clusterName, r.ObjSave), cache.Indexers{})
	case queue.Pods:
		indexer, informer = cache.NewIndexerInformer(lw, &v1.Pod{}, 0, initHandle(queue.Pods, worker, clusterName, r.ObjSave), cache.Indexers{})
	case queue.ConfigMaps:
		indexer, informer = cache.NewIndexerInformer(lw, &v1.ConfigMap{}, 0, initHandle(queue.ConfigMaps, worker, clusterName, r.ObjSave), cache.Indexers{})
	}
	return
}

// CreateAppsV1IndexInformer 构造informer需要的资源
func (r *ResourceAndNamespace) CreateAppsV1IndexInformer(client *kubernetes.Clientset, worker queue.Queue, clusterName string) (indexer cache.Indexer, informer cache.Controller) {
	lw := cache.NewListWatchFromClient(client.AppsV1().RESTClient(), r.RType, r.Namespace, fields.Everything())
	switch r.RType {
	case queue.Deployments:
		indexer, informer = cache.NewIndexerInformer(lw, &appsv1.Deployment{}, 0, initHandle(queue.Deployments, worker, clusterName, r.ObjSave), cache.Indexers{})
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
		klog.Infof("informer all [%v] namespace, namespace: [%v]", r.RType, v.Name)
		lw := cache.NewListWatchFromClient(client.CoreV1().RESTClient(), r.RType, v.Namespace, fields.Everything())
		switch r.RType {
		case queue.Services:
			indexer, informer := cache.NewIndexerInformer(lw, &v1.Service{}, 0, initHandle(queue.Services, worker, clusterName, r.ObjSave), cache.Indexers{})
			informerListRes = append(informerListRes, informer)
			indexerListRes = append(indexerListRes, indexer)
		case queue.Pods:
			indexer, informer := cache.NewIndexerInformer(lw, &v1.Pod{}, 0, initHandle(queue.Pods, worker, clusterName, r.ObjSave), cache.Indexers{})
			informerListRes = append(informerListRes, informer)
			indexerListRes = append(indexerListRes, indexer)
		case queue.ConfigMaps:
			indexer, informer := cache.NewIndexerInformer(lw, &v1.ConfigMap{}, 0, initHandle(queue.ConfigMaps, worker, clusterName, r.ObjSave), cache.Indexers{})
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
		klog.Infof("informer all [%v] namespace, namespace: [%v]", r.RType, v.Name)
		lw := cache.NewListWatchFromClient(client.AppsV1().RESTClient(), r.RType, v.Namespace, fields.Everything())
		switch r.RType {
		case queue.Deployments:
			indexer, informer := cache.NewIndexerInformer(lw, &appsv1.Deployment{}, 0, initHandle(queue.Deployments, worker, clusterName, r.ObjSave), cache.Indexers{})
			informerListRes = append(informerListRes, informer)
			indexerListRes = append(indexerListRes, indexer)
		}
	}

	isAll = true

	return indexerListRes, informerListRes, isAll

}

// Cluster 集群对象
type Cluster struct {
	MetaData MetaData `json:"metadata" yaml:"metadata"`
}

// NewClient 初始化client
func (c *Cluster) NewClient() (*kubernetes.Clientset, error) {

	if c.MetaData.ConfigPath != "" {
		config, err := clientcmd.BuildConfigFromFlags("", c.MetaData.ConfigPath)
		if err != nil {
			return nil, err
		}
		config.Insecure = c.MetaData.Insecure
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			return nil, err
		}
		return clientset, nil
	}
	return nil, errors.New("无法找到集群client端")
}