package helper

import "sync"

type CacheHelper interface {
	Get(key string) (value interface{})
	Set(key string, value interface{})
}

type CacheHelperImpl struct {
	logger LoggerHelper
	data   map[string]interface{}
	mu     sync.Mutex
}

func NewCacheHelper(l LoggerHelper) CacheHelper {
	return &CacheHelperImpl{
		logger: l,
		data:   make(map[string]interface{}),
	}
}

func (h *CacheHelperImpl) Get(key string) (value interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()

	value, found := h.data[key]
	if !found {
		h.logger.LogAndContinue("Key not found in cache, returning nil")
		return nil
	}
	return value
}

func (h *CacheHelperImpl) Set(key string, value interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.data[key] = value
}
