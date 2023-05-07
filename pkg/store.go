package pkg

import (
	"k8s.io/client-go/tools/cache"
)

// 本地缓存 接口
type store interface {
	List(string) []interface{}
	ListKeys(string) []string
	GetByKey(r string, key string) (items []interface{}, exists bool)
}

var _ store = mapIndexers{}

type mapIndexers map[string][]cache.Indexer

func (mapIndexer mapIndexers) List(r string) (l []interface{}) {
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

func (mapIndexer mapIndexers) ListKeys(r string) (keys []string) {
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
		for _, indexer := range mapIndexer[ConfigMaps] {
			keys = append(keys, indexer.ListKeys()...)
		}
	}
	return
}

func (mapIndexer mapIndexers) GetByKey(r string, key string) ([]interface{}, bool) {
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
	}

	return items, ok
}
