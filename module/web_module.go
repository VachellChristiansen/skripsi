package module

import (
	"os"
	"skripsi/processor"
	"skripsi/utils"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type WebModule interface {
	Init()
	Serve()
}

type WebModuleImpl struct {
	e                *echo.Echo
	WebViewProcessor processor.WebViewProcessor
	WebProcessor     processor.WebProcessor
}

func NewWebModule() WebModule {
	e := echo.New()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     strings.Split(os.Getenv("CORS_ALLOW_ORIGINS"), ","),
		AllowMethods:     strings.Split(os.Getenv("CORS_ALLOW_METHODS"), ","),
		AllowHeaders:     strings.Split(os.Getenv("CORS_ALLOW_HEADERS"), ","),
		AllowCredentials: true,
		ExposeHeaders:    strings.Split(os.Getenv("CORS_EXPOSE_HEADERS"), ","),
		MaxAge:           12 * 60 * 60,
	}))
	e.Renderer = utils.NewTemplate()
	return &WebModuleImpl{
		e: e,
		WebViewProcessor: processor.NewWebViewProcessor(),
		WebProcessor: processor.NewWebProcessor(),
	}
}

func (m *WebModuleImpl) Init() {
	static := m.e.Group("/static")
	static.Static("/", "web_views/static/")

	m.e.GET("/", m.WebViewProcessor.ServeIndexPage)
}

func (m *WebModuleImpl) Serve() {
	m.e.Start(":49991")
}

