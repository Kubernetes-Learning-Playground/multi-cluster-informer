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
	// EventAdd is sent when an object is added
	EventAdd = "add"

	// EventUpdate is sent when an object is modified
	// Captures the modified object
	EventUpdate = "update"

	// EventDelete is sent when an object is deleted
	// Captures the object at the last known state
	EventDelete = "delete"
)

// 放入队列 的数据
type QueueObject struct {
	Event    		string 		// 事件对象
	ResourceType 	string 	    // 资源对象
	Key      		string  	// <namespace>/<name>
	CreateAt    	time.Time   // 创建时间，也可以记录更新次数 与 更新时间
}
