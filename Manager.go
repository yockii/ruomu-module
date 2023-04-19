package ruomu_module

import (
	logger "github.com/sirupsen/logrus"
	"github.com/yockii/ruomu-core/database"

	"github.com/yockii/ruomu-module/controller"
	"github.com/yockii/ruomu-module/manager"
	"github.com/yockii/ruomu-module/model"
)

func Initial() (err error) {
	syncModels()

	var modules []*model.Module
	err = database.DB.Find(&modules, &model.Module{Status: 1}).Error
	if err != nil {
		logger.Errorln(err)
		return
	}

	// 已注册模块进行加载
	for _, module := range modules {
		manager.RegisterModule(module)
	}

	// 所有模块加载完毕
	logger.Infoln("所有已注册模块加载完毕")
	logger.Infoln("注入模块管理接口")
	controller.InitRouter()
	logger.Infoln("模块管理接口注入完成")

	return
}

// Destroy 销毁模块管理
func Destroy() {
	manager.Destroy()
}

func syncModels() {
	_ = database.AutoMigrate(
		model.Module{},
		model.ModuleDependency{},
		model.ModuleInjectInfo{},
		model.ModuleSettings{},
	)
}
