package helper

type TypeAssertHelper interface {
	String(interface{}) string
	StringSlice(interface{}) []string
	StringSlice2D(interface{}) [][]string
	MapStringStringSlice(interface{}) map[string][]string
}

type TypeAssertHelperImpl struct {
	logger LoggerHelper
}

func NewTypeAssertHelper(l LoggerHelper) TypeAssertHelper {
	return &TypeAssertHelperImpl{
		logger: NewLoggerHelper(),
	}
}

func (h *TypeAssertHelperImpl) String(base interface{}) (result string) {
	result, ok := base.(string)
	if !ok {
		h.logger.SetWarningPrefix()
		h.logger.OpenOutputFile()
		h.logger.LogAndContinue("Type assertion to string fails, returning empty string")
		h.logger.CloseOutputFile()
		return ""
	}
	return
}

func (h *TypeAssertHelperImpl) StringSlice(base interface{}) (result []string) {
	result, ok := base.([]string)
	if !ok {
		h.logger.SetWarningPrefix()
		h.logger.OpenOutputFile()
		h.logger.LogAndContinue("Type assertion to []string fails, returning nil")
		h.logger.CloseOutputFile()
		return nil
	}
	return
}

func (h *TypeAssertHelperImpl) StringSlice2D(base interface{}) (result [][]string) {
	result, ok := base.([][]string)
	if !ok {
		h.logger.SetWarningPrefix()
		h.logger.OpenOutputFile()
		h.logger.LogAndContinue("Type assertion to [][]string fails, returning nil")
		h.logger.CloseOutputFile()
		return nil
	}
	return
}

func (h *TypeAssertHelperImpl) MapStringStringSlice(base interface{}) (result map[string][]string) {
	result, ok := base.(map[string][]string)
	if !ok {
		h.logger.SetWarningPrefix()
		h.logger.OpenOutputFile()
		h.logger.LogAndContinue("Type assertion to map[string][]string fails, returning nil")
		h.logger.CloseOutputFile()
		return nil
	}
	return
}
