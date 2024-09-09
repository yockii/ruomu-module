package controller

import (
	"github.com/yockii/ruomu-core/server"
	"github.com/yockii/ruomu-module/manager"
)

func InitRouter() {
	module := server.Group("/module")
	module.Post("/add", manager.CheckAuthorizationMiddleware("module:add"), ModuleController.AddModule)
	module.Get("/list", manager.CheckAuthorizationMiddleware("module:list"), ModuleController.List)
	module.Get("/detail/:id", manager.CheckAuthorizationMiddleware("module:detail"), ModuleController.Detail)
	module.Post("/updateStatus", manager.CheckAuthorizationMiddleware("module:updateStatus"), ModuleController.UpdateStatus)
}
