package module

import (
	"skripsi/database"
	"skripsi/helper"
)

type CoreModule struct {
	Database  database.Database
	Helper    helper.Helper
	WebModule WebModule
}

func NewCoreModule() CoreModule {
	l := helper.NewLoggerHelper()
	return CoreModule{
		Database:  database.NewDatabase(),
		Helper:    helper.NewHelper(),
		WebModule: NewWebModule(l),
	}
}
