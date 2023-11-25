## Multi-cluster informer for kubernetes
<a href="./README.md">English</a> | <a href="./README-zh.md">简体中文</a>
### Introduction
Project background: Generally, client-go in kubernetes only has a single-cluster and single-resource monitoring demo, and the writing is relatively simple. Based on this problem, this project uses the client-go package for extended encapsulation to achieve "**multiple clusters**" and "**multiple resources**"
informer mechanism. 

The caller only needs to maintain the config.yaml configuration file and handlerFunc.

Supported:
1. "Multi-cluster" informer configuration can be provided.
2. Can provide multi-resource informer, currently only supports pods,services,configmaps,deployments,events,secrets,statefulsets,daemonsets, etc.
3. Supports using the all field to monitor specific resources of all namespaces when configuring a namespace.
4. Can support skipping the TLS authentication process and calling informer directly.
5. Supports callback to listen to the runtime.Object instance of the resource object.

![](https://github.com/Kubernetes-Learning-Playground/multi-cluster-informer/blob/main/image/%E6%97%A0%E6%A0%87%E9%A2%98-2023-08-10-2343.png?raw=true)

### P.S.:
1. Create a resource file in the directory, copy the cluster's .kube/config file and put it in the project root directory (remember that the cluster server needs to be changed to "public network ip").
2. This project supports insecurity mode, so the certificate-authority-data field needs to be deleted in the config file, otherwise the connection will report an error (it does not need to delete it if it supports TLS certificate).
3. Multiple .kube/config configuration files can be configured to support multi-cluster list query.

### Configuration file
- **Important** The configuration file can refer to the configuration in config.yaml. The caller only needs to pay attention to the content in the configuration file.
```yaml
maxrequeuetime: 5             # 最大重入队列次数
clusters:                     # 集群列表
  - metadata:
      clusterName: cluster1   # 自定义集群名
      insecure: true          # 是否开启跳过tls证书认证
      configPath: /Users/zhenyu.jiang/go/src/golanglearning/new_project/multi_cluster_informer/resource/config2 # kube config配置文件地址
      list:                   # 列表：目前支持：pods deployments services configmaps events等资源对象的监听
        - rType: pods         # 资源对象
          namespace: all      # namespace：可支持特定namespace或all
          objSave: false      # 支持informer实例返回runtime.Object对象，在多集群监听时，需要考虑内存问题，
          # 如果没有特殊要求，可以设置为objSave
        - rType: deployments
          namespace: default
          objSave: true
        - rType: services
          namespace: default
          objSave: true
```

### Usage examples
```go
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
    
    
    // The following method is recommended. The caller only needs to care about the settings in the configuration file.
    // 1. Get controller object
    r, err := multi_informer.NewMultiClusterInformerFromConfig("./config.yaml")
    if err != nil {
        klog.Fatal("multi cluster informer err: ", err)
    }
    // 2. add handler
    r.AddEventHandler(func(object queue.QueueObject) error {
        // only the add event
        if object.Event == queue.EventAdd {
            fmt.Println("目前监听到事件为add的资源对象", object.ResourceType)
        }
        //fmt.Println("目前监听到的资源对象", obj.ResourceType, obj.Event)
       
        if object.ClusterName == "cluster1" {
            fmt.Println("目前监听到集群为cluster1的资源对象", object.ResourceType)
        }
    
        if object.ClusterName == "cluster2" {
            fmt.Println("目前监听到集群为cluster2的资源对象", object.ResourceType)
        }
        
        fmt.Println(time.Now(), object.Event, object.ResourceType, object.Key, object.ClusterName)
        return nil
    })
    
    // 3. run informer
    go r.Run()
    defer r.Stop()

    // 4. Continuously remove resource objects from the queue
    for {
        obj, _ := r.Pop()
        // method one：use handler
        // If there is a problem with your own logic, 
		// you can put it back in the queue.
        if err = r.HandleObject(obj); err != nil {
            _ = r.ReQueue(obj)  // reQueue
        } else { 
			// t's done
            r.Finish(obj)
    }

    // method two：Handle it yourself
    //if err = process(obj); err != nil {
        // _ = r.ReQueue(obj) // 重新入列
    //} else { // 完成就结束
        // r.Finish(obj)
    //}
}

// process execute your own logic
func process(obj queue.QueueObject) error {
	
    if obj.Event == queue.EventAdd {
        fmt.Println("目前监听到事件为add的资源对象", obj.ResourceType)
    }
    
    if obj.ClusterName == "cluster2" {
        fmt.Println("目前监听到集群为cluster2的资源对象", obj.ResourceType)
    }
    
    fmt.Println(time.Now(), obj.Event, obj.ResourceType, obj.Key, obj.ClusterName)
    return nil
}
```
