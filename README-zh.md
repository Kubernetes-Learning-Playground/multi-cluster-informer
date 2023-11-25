## kubernetes的多集群多资源informer监听。
### 项目思路与功能
项目背景：一般在kubernetes中的client-go的仅有单集群且单资源的监听demo，并且写的相对简陋。基于这个问题，本项目采用client-go包进行扩展封装，实现"**多集群**"且"**多资源**"的
informer机制。调用方仅需要维护config.yaml配置文件与handlerFunc即可。

支持功能：
1. 可提供"多集群"informer配置。
2. 可提供多资源informer，目前只支持pods、services、configmaps、deployments、events、secrets、statefulsets、daemonsets等。
3. 可支持在配置namespace时，使用all字段来监听所有namespace的特定资源。
4. 可支持跳过tls认证过程直接调用informer
5. 可支持回传监听到资源对象的runtime.Object实例

![](https://github.com/Kubernetes-Learning-Playground/multi-cluster-informer/blob/main/image/%E6%97%A0%E6%A0%87%E9%A2%98-2023-08-10-2343.png?raw=true)

### 附注：
1. 目录下创建一个resource文件，把集群的.kube/config文件复制一份放入(记得cluster server需要改成"公网ip")。
2. 本项目支持insecurity模式，所以config文件需要把certificate-authority-data字段删除，否则连接会报错(本身支持tls证书也可以不删除)。
3. 可配置多个.kube/config配置文件，支持多集群list查询。

### 配置文件
- **重要** 配置文件可参考config.yaml中配置，调用方只需要关注配置文件中的内容即可。
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

### 使用范例
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
```
