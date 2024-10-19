package processor

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type WebViewProcessor interface {
	ServeIndexPage(c echo.Context) error
}

type WebViewProcessorImpl struct {
}

func NewWebViewProcessor() WebViewProcessor {
	return &WebViewProcessorImpl{}
}

func (p *WebViewProcessorImpl) ServeIndexPage(c echo.Context) error {
	return c.Render(http.StatusOK, "index", nil)
}
