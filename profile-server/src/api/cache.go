package api

import (
	"github.com/go-redis/redis/v9"
)

func (h *Handler) cachePut(key string, value string) error {
	err := h.Rdb.Set(h.Ctx, key, value, TTL).Err()

	if err != nil {
		h.Logger.Error("[%s] Failed to cache item: %v", key, err)
	}

	return err
}

func (h *Handler) cacheGet(key string) (bool, string, error) {
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
