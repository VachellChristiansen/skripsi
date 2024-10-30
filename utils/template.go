package utils

import (
	"html/template"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

// Template Renderer
type Template struct {
	templ *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templ.ExecuteTemplate(w, name, data)
}

func Repeat(text string, times int) string {
	return strings.Repeat(text, times)
}

func ReplaceAll(text, old, new string) string {
	return strings.ReplaceAll(text, old, new)
}

func TruncateTime(datetime string) string {
	parts := strings.Split(datetime, "T")
	return parts[0]
}

func SafeJS(s string) template.JS {
	return template.JS(s)
}

func NewTemplate() *Template {
	funcMap := template.FuncMap{
		"Repeat":       Repeat,
		"ReplaceAll":   ReplaceAll,
		"TruncateTime": TruncateTime,
		"SafeJS":       SafeJS,
	}
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get Work Directory, err: %v", err)
	}

	webViewsPath := filepath.Join(wd, "web_views")

	t := template.Must(template.New("").Funcs(funcMap).ParseGlob(webViewsPath + "/*.html"))
	t = template.Must(t.ParseGlob(webViewsPath + "/components/*.html"))

	return &Template{
		templ: t,
	}
}
