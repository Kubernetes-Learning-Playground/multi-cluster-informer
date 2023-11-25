package queue

import (
	"k8s.io/client-go/tools/cache"
)

// Store 本地缓存接口
type Store interface {
	// List 列出所有资源对象
	List(string) []interface{}
	// ListKeys 列出所有资源对象的key
	ListKeys(string) []string
	// GetByKey 输入特定key，返回资源对象
	GetByKey(r string, key string) (items []interface{}, exists bool)
}

var _ Store = MapIndexers{}

type MapIndexers map[string][]cache.Indexer

func (mapIndexer MapIndexers) List(r string) (l []interface{}) {
	switch r {
	case All:
		for _, set := range mapIndexer {
			for _, indexer := range set {
				l = append(l, indexer.List()...)
			}
		}
	case Services:
		for _, indexer := range mapIndexer[Services] {
			l = append(l, indexer.List()...)
		}
	case Pods:
		for _, indexer := range mapIndexer[Pods] {
			l = append(l, indexer.List()...)
		}
	case ConfigMaps:
		for _, indexer := range mapIndexer[ConfigMaps] {
			l = append(l, indexer.List()...)
		}
	case Deployments:
		for _, indexer := range mapIndexer[Deployments] {
			l = append(l, indexer.List()...)
		}
	}
	return
}

func (mapIndexer MapIndexers) ListKeys(r string) (keys []string) {
	switch r {
	case All:
		for _, set := range mapIndexer {
			for _, indexer := range set {
				keys = append(keys, indexer.ListKeys()...)
			}
		}
	case Services:
		for _, indexer := range mapIndexer[Services] {
			keys = append(keys, indexer.ListKeys()...)

		}
	case Pods:
		for _, indexer := range mapIndexer[Pods] {
			keys = append(keys, indexer.ListKeys()...)
		}
	case ConfigMaps:
		for _, indexer := range mapIndexer[ConfigMaps] {
			keys = append(keys, indexer.ListKeys()...)
		}
	case Deployments:
		for _, indexer := range mapIndexer[Deployments] {
			keys = append(keys, indexer.ListKeys()...)
		}
	case Statefulsets:
		for _, indexer := range mapIndexer[Statefulsets] {
			keys = append(keys, indexer.ListKeys()...)

		}
	case Daemonsets:
		for _, indexer := range mapIndexer[Daemonsets] {
			keys = append(keys, indexer.ListKeys()...)

		}
	case Secrets:
		for _, indexer := range mapIndexer[Secrets] {
			keys = append(keys, indexer.ListKeys()...)

		}
	}
	return
}

func (mapIndexer MapIndexers) GetByKey(r string, key string) ([]interface{}, bool) {
	var items []interface{}
	ok := false

	switch r {
	case All:
		for _, set := range mapIndexer {
			for _, indexer := range set {
				item, exists, err := indexer.GetByKey(key)
				if err != nil {
					continue
				}
				if exists {
					ok = true
					items = append(items, item)
				}
			}
		}
	case Services:
		for _, indexer := range mapIndexer[Services] {
			item, exists, err := indexer.GetByKey(key)
			if err != nil {
				continue
			}
			if exists {
				ok = true
				items = append(items, item)
			}
		}
	case Pods:
		for _, indexer := range mapIndexer[Pods] {
			item, exists, err := indexer.GetByKey(key)
			if err != nil {
				continue
			}
			if exists {
				ok = true
				items = append(items, item)
			}
		}
	case ConfigMaps:
		for _, indexer := range mapIndexer[ConfigMaps] {
			item, exists, err := indexer.GetByKey(key)
			if err != nil {
				continue
			}
			if exists {
				ok = true
				items = append(items, item)
			}
		}
	case Deployments:
		for _, indexer := range mapIndexer[Deployments] {
			item, exists, err := indexer.GetByKey(key)
			if err != nil {
				continue
			}
			if exists {
				ok = true
				items = append(items, item)
			}
		}
	case Statefulsets:
		for _, indexer := range mapIndexer[Statefulsets] {
			item, exists, err := indexer.GetByKey(key)
			if err != nil {
				continue
			}
			if exists {
				ok = true
				items = append(items, item)
			}
		}
	case Daemonsets:
		for _, indexer := range mapIndexer[Daemonsets] {
			item, exists, err := indexer.GetByKey(key)
			if err != nil {
				continue
			}
			if exists {
				ok = true
				items = append(items, item)
			}
		}
	case Secrets:
		for _, indexer := range mapIndexer[Secrets] {
			item, exists, err := indexer.GetByKey(key)
			if err != nil {
				continue
			}
			if exists {
				ok = true
				items = append(items, item)
			}
		}
	}

	return items, ok
}
