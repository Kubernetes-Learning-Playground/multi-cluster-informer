package pkg

import (
	"errors"
	"k8s.io/client-go/util/workqueue"
)

// 接口对象
type queue interface {
	// 将监听到的资源放入queue中
	push(QueueObject)

	// 拿出队列
	Pop() (QueueObject, error)

	// 重新放入队列，次数可配置
	ReQueue(QueueObject) error

	// 完成入列操作
	Finish(QueueObject)

	// 关闭所有informer
	close()

	// 设置最大重新入列次数
	SetReMaxReQueueTime(int)
}

// 使用限速队列实现queue接口
type wq struct {
	workqueue.RateLimitingInterface
	MaxReQueueTime int
}

var _ queue = &wq{}

func newWorkQueue(maxReQueueTime int) *wq {
	return &wq{
		workqueue.NewRateLimitingQueue(workqueue.DefaultItemBasedRateLimiter()),
		maxReQueueTime,
	}
}

// 放入队列
func (c *wq) push(obj QueueObject) {
	c.AddRateLimited(obj)
}

// 取出队列
func (c *wq) Pop() (QueueObject, error) {
	obj, quit := c.Get()
	if quit {
		return QueueObject{}, errors.New("Controller has been stoped. ")
	}

	return obj.(QueueObject), nil
}

// 结束要干两件事，忘记+done
func (c *wq) Finish(obj QueueObject) {
	c.Forget(obj)
	c.Done(obj)
}

// 重新放入
func (c *wq) ReQueue(obj QueueObject) error {
	if c.NumRequeues(obj) < c.MaxReQueueTime {
		// 这里会重新放入对列
		c.AddRateLimited(obj)
		return nil
	}

	c.Forget(obj)

	c.Done(obj)

	return errors.New("This object has been requeued for many times, but still fails. ")
}

func (c *wq) close() {
	c.ShutDown()
}

func (c *wq) SetReMaxReQueueTime(maxReQueueTime int) {
	c.MaxReQueueTime = maxReQueueTime
}
