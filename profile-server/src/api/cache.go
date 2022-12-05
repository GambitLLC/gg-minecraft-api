package api

import (
	"github.com/go-redis/redis/v9"
	"time"
)

func (h *Handler) CachePut(key string, value string, ttl time.Duration) error {
	err := h.Rdb.Set(h.Ctx, key, value, ttl).Err()

	if err != nil {
		h.Logger.Error("[%s] Failed to cache item: %v", key, err)
	}

	return err
}

func (h *Handler) CacheGet(key string) (bool, string, error) {
	item, err := h.Rdb.Get(h.Ctx, key).Result()

	switch err {
	case redis.Nil:
		//key does not exist in cache
		return false, "", nil
	case nil:
		//key does exist in cache
		return true, item, nil
	default:
		//some redis error occurred during cache lookup
		h.Logger.Error("%v", err)
		return false, "", err
	}
}
