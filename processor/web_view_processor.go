package processor

import (
	"net/http"
	"skripsi/helper"
	"time"

	"github.com/labstack/echo/v4"
)

type WebViewProcessor interface {
	ServeIndexPage(c echo.Context) error
}

type WebViewProcessorImpl struct {
	logger helper.LoggerHelper
}

func NewWebViewProcessor(l helper.LoggerHelper) WebViewProcessor {
	return &WebViewProcessorImpl{
		logger: l,
	}
}

func (p *WebViewProcessorImpl) ServeIndexPage(c echo.Context) error {
	return c.Render(http.StatusOK, "index", map[string]interface{}{
		"Timestamp": time.Now().Unix(),
	})
}
