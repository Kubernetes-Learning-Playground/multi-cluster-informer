package queue

import (
	"time"
)

// 支持资源对象类型
const (
	All         = "all"
	Services    = "services"
	Deployments = "deployments"
	Pods        = "pods"
	ConfigMaps  = "configmaps"
	Events      = "events"
)

// 事件类型
const (
	EventAdd    = "add"
	EventUpdate = "update"
	EventDelete = "delete"
)

// QueueObject 入队对象
// 用来包装经由informer收到的资源对象
type QueueObject struct {
	ClusterName  string      // 集群名称
	Event        string      // 事件对象
	ResourceType string      // 资源类型
	Key          string      // <namespace>/<name>
	Obj          interface{} // runtime.Object
	CreateAt     time.Time   // 创建时间，也可以记录更新次数 与 更新时间
}
