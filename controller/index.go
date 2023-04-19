package controller

import "github.com/yockii/ruomu-core/server"

func InitRouter() {
	module := server.Group("/module")
	module.Post("/add", ModuleController.AddModule)
	module.Get("/list", ModuleController.List)

	module.Get("/detail/:id", ModuleController.Detail)

	module.Post("/updateStatus", ModuleController.UpdateStatus)
}
