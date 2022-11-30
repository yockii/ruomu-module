package service

import (
	logger "github.com/sirupsen/logrus"
	"github.com/yockii/ruomu-core/database"
	"github.com/yockii/ruomu-core/util"

	"github.com/yockii/ruomu-module/model"
)

var ModuleService = new(moduleService)

type moduleService struct{}

func (s *moduleService) AddModule(module *model.Module, dependencies []*model.ModuleDependency, injectPoint []*model.ModuleInjectInfo, settings []*model.ModuleSettings) error {
	session := database.DB.NewSession()
	defer session.Close()

	module.Id = util.SnowflakeId()
	if _, err := session.Insert(module); err != nil {
		logger.Errorln(err)
		return err
	}

	for _, dependency := range dependencies {
		dependency.Id = util.SnowflakeId()
		dependency.ModuleCode = module.Code
		if _, err := session.Insert(dependency); err != nil {
			logger.Errorln(err)
			return err
		}
	}

	for _, injectInfo := range injectPoint {
		injectInfo.Id = util.SnowflakeId()
		injectInfo.ModuleId = module.Id
		if _, err := session.Insert(injectInfo); err != nil {
			logger.Errorln(err)
			return err
		}
	}

	for _, setting := range settings {
		setting.Id = util.SnowflakeId()
		setting.ModuleId = module.Id
		if _, err := session.Insert(setting); err != nil {
			logger.Errorln(err)
			return err
		}
	}

	return session.Commit()
}
