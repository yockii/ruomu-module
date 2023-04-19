package service

import (
	"errors"
	logger "github.com/sirupsen/logrus"
	"github.com/yockii/ruomu-core/database"
	"github.com/yockii/ruomu-core/util"
	"github.com/yockii/ruomu-module/domain"
	"gorm.io/gorm"

	"github.com/yockii/ruomu-module/model"
)

var ModuleService = new(moduleService)

type moduleService struct{}

func (s *moduleService) AddModule(module *model.Module, dependencies []*model.ModuleDependency, injectPoint []*model.ModuleInjectInfo, settings []*model.ModuleSettings) error {
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		module.ID = util.SnowflakeId()

		if err := tx.Create(module).Error; err != nil {
			logger.Errorln(err)
			return err
		}

		for _, dependency := range dependencies {
			dependency.ID = util.SnowflakeId()
			dependency.ModuleCode = module.Code
			if err := tx.Create(dependency).Error; err != nil {
				logger.Errorln(err)
				return err
			}
		}
		for _, injectInfo := range injectPoint {
			injectInfo.ID = util.SnowflakeId()
			injectInfo.ModuleID = module.ID
			if err := tx.Create(injectInfo).Error; err != nil {
				logger.Errorln(err)
				return err
			}
		}

		for _, setting := range settings {
			setting.ID = util.SnowflakeId()
			setting.ModuleID = module.ID
			if err := tx.Create(setting).Error; err != nil {
				logger.Errorln(err)
				return err
			}
		}

		return nil
	})
	return err
}

func (s *moduleService) List(condition *model.Module, limit, offset int) (list []*model.Module, total int64, err error) {
	db := database.DB.Model(&model.Module{})
	if condition.Code != "" {
		db = db.Where("code like ?", "%"+condition.Code+"%")
	}
	if condition.Name != "" {
		db = db.Where("name like ?", "%"+condition.Name+"%")
	}
	if condition.Status != 0 {
		db = db.Where("status = ?", condition.Status)
	}
	if err = db.Limit(limit).Offset(offset).Find(&list).Offset(-1).Count(&total).Error; err != nil {
		logger.Errorln(err)
		return
	}
	return
}

func (s *moduleService) Detail(id uint64) (*domain.Module, error) {
	result := new(domain.Module)
	module := new(model.Module)
	if err := database.DB.Where("id = ?", id).First(module).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.Errorln(err)
		return result, err
	}
	result.Module = *module
	if err := database.DB.Where("module_code = ?", result.Code).Find(&result.Dependencies).Error; err != nil {
		logger.Errorln(err)
	}
	if err := database.DB.Where("module_id = ?", result.ID).Find(&result.Injects).Error; err != nil {
		logger.Errorln(err)
	}
	if err := database.DB.Where("module_id = ?", result.ID).Find(&result.Settings).Error; err != nil {
		logger.Errorln(err)
	}
	return result, nil
}

// Instance 仅获取mudule本身
func (s *moduleService) Instance(id uint64) (*model.Module, error) {
	module := new(model.Module)
	if err := database.DB.Where("id = ?", id).First(module).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.Errorln(err)
		return nil, err
	}
	return module, nil
}

func (s *moduleService) UpdateStatus(id uint64, status int) error {
	if err := database.DB.Model(&model.Module{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		logger.Errorln(err)
		return err
	}
	return nil
}
