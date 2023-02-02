package pkg

import "time"

const (
	All = "all"

	Services = "services"

	Deployments = "deployments"

	Pods = "pods"

	ConfigMaps = "configmaps"
)



const (
	EventAdd = "add"
	EventUpdate = "update"
	EventDelete = "delete"
)

// 放入队列 的数据
type QueueObject struct {
	ClusterName 	string      // 集群名称
	Event    		string 		// 事件对象
	ResourceType 	string 	    // 资源对象
	Key      		string  	// <namespace>/<name>
	CreateAt    	time.Time   // 创建时间，也可以记录更新次数 与 更新时间
}
