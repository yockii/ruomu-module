package controller

import "github.com/yockii/ruomu-core/server"

func InitRouter() {
	module := server.Group("/module")
	module.Post("/add", ModuleController.AddModule)
}
