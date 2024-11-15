package module

import (
	"fmt"
	"os"
	"skripsi/helper"
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
	e         *echo.Echo
	logger    helper.LoggerHelper
	Processor processor.Processor
}

func NewWebModule(l helper.LoggerHelper) WebModule {
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
		e:         e,
		logger:    l,
		Processor: processor.NewProcessor(l),
	}
}

func (m *WebModuleImpl) Init() {
	static := m.e.Group("/static")
	static.Static("/", "web_views/static/")

	m.e.GET("/", m.Processor.WebViewProcessor.ServeIndexPage)
	m.e.POST("/flood", m.Processor.WebProcessor.HandleFloodPredictionRequestV2)
}

func (m *WebModuleImpl) Serve() {
	m.e.Start(fmt.Sprintf(":%s", os.Getenv("WEB_PORT")))
}
