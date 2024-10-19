package helper

type Helper struct {
	LoggerHelper     LoggerHelper
	CacheHelper      CacheHelper
	TypeAssertHelper TypeAssertHelper
}

func NewHelper() Helper {
	logger := NewLoggerHelper()
	return Helper{
		LoggerHelper:     logger,
		CacheHelper:      NewCacheHelper(logger),
		TypeAssertHelper: NewTypeAssertHelper(logger),
	}
}
