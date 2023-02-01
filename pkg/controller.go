package pkg

import (
	"errors"
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
		for _, r := range c.Resources {
			var indexer cache.Indexer
			var informer cache.Controller
			if r.RType == Deployments {
				indexer, informer = r.createAppsV1IndexInformer(client, core.queue)
			} else {
				indexer, informer = r.createCoreV1IndexInformer(client, core.queue)
			}

			// 放入 list中
			store[r.RType] = append(store[r.RType], indexer)
			informers = append(informers, informer)
		}
	}

	core.informers = informers
	core.store = store

	return core, nil
}

// 处理informer逻辑
func initHandle(resource string, worker queue) cache.ResourceEventHandlerFuncs {
	handler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				worker.push(QueueObject{EventAdd, resource, key, time.Now()})
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			//bAdd := true
			if err == nil {
				// 放入
				worker.push(QueueObject{EventUpdate, resource, key, time.Now()})

			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				worker.push(QueueObject{EventDelete, resource, key, time.Now()})
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
			panic("Timed out waiting for caches to sync")
		}
	}
}

// 资源与namespace
type ResourceAndNamespace struct {
	RType          string
	Namespace      string
}

// 构造informer需要的资源
func (r *ResourceAndNamespace) createCoreV1IndexInformer(client *kubernetes.Clientset, worker queue) (indexer cache.Indexer, informer cache.Controller)  {
	lw := cache.NewListWatchFromClient(client.CoreV1().RESTClient(), r.RType, r.Namespace, fields.Everything())
	switch r.RType {
	case Services:
		indexer, informer = cache.NewIndexerInformer(lw, &v1.Service{}, 0, initHandle(Services, worker), cache.Indexers{})
	case Pods:
		indexer, informer = cache.NewIndexerInformer(lw, &v1.Pod{}, 0, initHandle(Pods, worker), cache.Indexers{})
	case ConfigMaps:
		indexer, informer = cache.NewIndexerInformer(lw, &v1.ConfigMap{}, 0, initHandle(ConfigMaps, worker), cache.Indexers{})
	}
	return
}

// 构造informer需要的资源
func (r *ResourceAndNamespace) createAppsV1IndexInformer(client *kubernetes.Clientset, worker queue) (indexer cache.Indexer, informer cache.Controller)  {
	lw := cache.NewListWatchFromClient(client.AppsV1().RESTClient(), r.RType, r.Namespace, fields.Everything())
	switch r.RType {
	case Deployments:
		indexer, informer = cache.NewIndexerInformer(lw, &v1.Service{}, 0, initHandle(Deployments, worker), cache.Indexers{})
	}
	return
}

// 集群对象
type Cluster struct {
	ConfigPath      string // kube config文件
	Resources       []ResourceAndNamespace
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
	return nil, errors.New("Can`t find a way to access to k8s api. Please make sure ConfigPath or MasterUrl in cluster ")
}

