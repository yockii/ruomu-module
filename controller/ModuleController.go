package controller

import (
	"github.com/gofiber/fiber/v2"
	logger "github.com/sirupsen/logrus"
	"github.com/yockii/ruomu-core/config"
	"github.com/yockii/ruomu-core/server"

	"github.com/yockii/ruomu-module/manager"
	"github.com/yockii/ruomu-module/model"
	"github.com/yockii/ruomu-module/service"
)

var ModuleController = new(moduleController)

type moduleController struct{}

func (c *moduleController) AddModule(ctx *fiber.Ctx) error {
	type moduleReq struct {
		model.Module
		Dependencies        []*model.ModuleDependency `json:"dependencies,omitempty"`
		Injects             []*model.ModuleInjectInfo `json:"injects,omitempty"`
		NeedDb              bool                      `json:"needDb,omitempty"`
		NeedUserTokenExpire bool                      `json:"needUserTokenExpire,omitempty"`
	}
	req := new(moduleReq)
	if err := ctx.BodyParser(req); err != nil {
		logger.Errorln(err)
		return ctx.JSON(&server.CommonResponse{
			Code: server.ResponseCodeParamParseError,
			Msg:  server.ResponseMsgParamParseError,
		})
	}
	var settings []*model.ModuleSettings
	if req.NeedDb {
		for k, v := range config.GetStringMapString("database") {
			settings = append(settings, &model.ModuleSettings{
				Code:  "database." + k,
				Value: v,
			})
		}
	}
	if req.NeedUserTokenExpire {
		settings = append(settings, &model.ModuleSettings{
			Code:  "userTokenExpire",
			Value: config.GetString("userTokenExpire"),
		})
	}
	module := &req.Module
	err := service.ModuleService.AddModule(module, req.Dependencies, req.Injects, settings)
	if err != nil {
		return ctx.JSON(&server.CommonResponse{
			Code: server.ResponseCodeUnknownError,
			Msg:  err.Error(),
		})
	}

	if req.Status == 1 {
		manager.RegisterModule(module)
		go func() {
			server.Shutdown()
		}()
	}

	return ctx.JSON(&server.CommonResponse{Data: true})
}
