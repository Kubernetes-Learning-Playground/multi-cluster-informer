package queue

import (
	"errors"
	"k8s.io/client-go/util/workqueue"
)

// Queue 接口对象
type Queue interface {
	// Push 将监听到的资源放入queue中
	Push(QueueObject)
	// Pop 拿出队列
	Pop() (QueueObject, error)
	// ReQueue 重新放入队列，次数可配置
	ReQueue(QueueObject) error
	// Finish 完成入列操作
	Finish(QueueObject)
	// Close 关闭所有informer
	Close()
	// SetReMaxReQueueTime 设置最大重新入列次数
	SetReMaxReQueueTime(int)
}

// Wq 使用限速队列实现queue接口
type Wq struct {
	workqueue.RateLimitingInterface
	MaxReQueueTime int
}

var _ Queue = &Wq{}

func NewWorkQueue(maxReQueueTime int) *Wq {
	return &Wq{
		workqueue.NewRateLimitingQueue(workqueue.DefaultItemBasedRateLimiter()),
		maxReQueueTime,
	}
}

// Push 放入队列
func (c *Wq) Push(obj QueueObject) {
	c.AddRateLimited(obj)
}

// Pop 取出队列
func (c *Wq) Pop() (QueueObject, error) {
	obj, quit := c.Get()
	if quit {
		return QueueObject{}, errors.New("Controller has been stoped. ")
	}
	return obj.(QueueObject), nil
}

// Finish 结束要干两件事，忘记+done
func (c *Wq) Finish(obj QueueObject) {
	c.Forget(obj)
	c.Done(obj)
}

// ReQueue 重新放入
func (c *Wq) ReQueue(obj QueueObject) error {
	if c.NumRequeues(obj) < c.MaxReQueueTime {
		// 这里会重新放入对列
		c.AddRateLimited(obj)
		return nil
	}
	// 如果次数大于最大重试次数，直接丢弃
	c.Forget(obj)
	c.Done(obj)
	return errors.New("This object has been requeued for many times, but still fails. ")
}

func (c *Wq) Close() {
	c.ShutDown()
}

func (c *Wq) SetReMaxReQueueTime(maxReQueueTime int) {
	c.MaxReQueueTime = maxReQueueTime
}
