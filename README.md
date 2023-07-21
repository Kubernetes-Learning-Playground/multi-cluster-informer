## kubernetes的多集群多资源informer监听。
### 项目思路与功能
项目背景：一般在kubernetes中的client-go的仅有单集群且单资源的监听demo，并且写的相对简陋。基于这个问题，本项目采用client-go包，实现"**多集群**"且"**多资源**"的
informer机制。

支持功能：
1. 可提供"多集群"informer。
2. 可提供多资源informer，目前只支持pods、services、configmaps、deployments等。
3. 可支持在配置namespace时，使用all字段来监听所有namespace的特定资源。

![](https://github.com/googs1025/multi-cluster-informer/blob/main/image/%E6%B5%81%E7%A8%8B%E5%9B%BE.jpg?raw=true)

### 附注：
1. 目录下创建一个resource文件，把集群的.kube/config文件复制一份放入(记得cluster server需要改成"公网ip")。
2. 本项目使用insecurity模式，所以config文件需要把certificate-authority-data字段删除，否则连接会报错。
3. 可以放置多个.kube/config配置文件，支持多集群list查询。

### 配置文件
```yaml
maxrequeuetime: 5             # 最大重试次数
clusters:                     # 监听集群范围
  - metadata:
      list:                   # 资源对象与namespace
        - rtype: pods
          namespace: default
        - rtype: deployments
          namespace: default
        - rtype: services
          namespace: default
      clustername: cluster1   # 集群名
    # ./kube/config 的目录地址  
    configpath: /Users/zhenyu.jiang/go/src/golanglearning/new_project/multi_cluster_informer/resource/config1
  - metadata:
      list:
        - rtype: pods
          namespace: default
        - rtype: deployments
          namespace: default
        - rtype: services
          namespace: default
      clustername: cluster2
    configpath: /Users/zhenyu.jiang/go/src/golanglearning/new_project/multi_cluster_informer/resource/config
```

### 使用范例
```go
func main() {
	// 1. 项目配置
	sysConfig, err := config.LoadConfig("./pkg/config/config.yaml")
	if err != nil {
		klog.Error("load config error: ", err)
		return
	}

	// 2. 启动多集群informer
	r, err := pkg.NewMultiClusterInformer(
		sysConfig.MaxReQueueTime,
		sysConfig.Clusters,
	)
	if err != nil {
		klog.Fatal("multi cluster informer err: ", err)
	}

	// 3. 执行informer监听
	go r.Run()
	defer r.Stop()

	// 4. 不断从队列取出资源对象
	for {
		obj, _ := r.Pop()
		// 如果自己的业务逻辑发生问题，可以重新放回队列。
		if err = process(obj); err != nil {
			_ = r.ReQueue(obj) // 重新入列
		} else { // 完成就结束
			r.Finish(obj)
		}
	}

}

// process 执行自己的业务逻辑
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
```
