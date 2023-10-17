package selector

import (
	"chinchilla/transfer/service"
	"github.com/meow-pad/persian/frame/pservice/cache"
)

func NewCacheSelector(cache *cache.Cache) *CacheSelector {
	return &CacheSelector{cache: cache}
}

func buildRouterKey(routerId string) string {
	return "r:" + routerId
}

type CacheSelector struct {
	cache *cache.Cache
}

func (selector *CacheSelector) Select(routerId string) (string, error) {
	// 先查是否在当前缓存中
	value, ok, err := selector.cache.Get(routerId)
	if err == nil {
		if ok {
			return value, nil
		} else {
			return "", nil
		}
	} else {
		return "", err
	}
}

func (selector *CacheSelector) Update(_ []service.Info) {
}
