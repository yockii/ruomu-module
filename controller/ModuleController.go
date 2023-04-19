package controller

import (
	"github.com/gofiber/fiber/v2"
	logger "github.com/sirupsen/logrus"
	"github.com/yockii/ruomu-core/config"
	"github.com/yockii/ruomu-core/server"
	"strconv"

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
			_ = server.Shutdown() // 重启server
		}()
	}

	return ctx.JSON(&server.CommonResponse{Data: true})
}

// List 获取Module列表
func (c *moduleController) List(ctx *fiber.Ctx) error {
	// 获取筛选条件和分页信息
	condition := new(model.Module)
	if err := ctx.QueryParser(condition); err != nil {
		logger.Errorln(err)
		return ctx.JSON(&server.CommonResponse{
			Code: server.ResponseCodeParamParseError,
			Msg:  server.ResponseMsgParamParseError,
		})
	}
	paginate := new(server.Paginate)
	if err := ctx.QueryParser(paginate); err != nil {
		logger.Errorln(err)
		return ctx.JSON(&server.CommonResponse{
			Code: server.ResponseCodeParamParseError,
			Msg:  server.ResponseMsgParamParseError,
		})
	}

	// 获取列表
	modules, total, err := service.ModuleService.List(condition, paginate.Limit, paginate.Offset)
	if err != nil {
		logger.Errorln(err)
		return ctx.JSON(&server.CommonResponse{
			Code: server.ResponseCodeDatabase,
			Msg:  server.ResponseMsgDatabase + err.Error(),
		})
	}

	return ctx.JSON(&server.CommonResponse{
		Data: &server.Paginate{
			Total:  total,
			Offset: paginate.Offset,
			Limit:  paginate.Limit,
			Items:  modules,
		},
	})
}

// Detail 根据ID获取Module详情
func (c *moduleController) Detail(ctx *fiber.Ctx) error {
	id, _ := strconv.ParseUint(ctx.Params("id"), 10, 64)
	module, err := service.ModuleService.Detail(id)
	if err != nil {
		logger.Errorln(err)
		return ctx.JSON(&server.CommonResponse{
			Code: server.ResponseCodeDatabase,
			Msg:  server.ResponseMsgDatabase + err.Error(),
		})
	}

	return ctx.JSON(&server.CommonResponse{Data: module})
}

// UpdateStatus 更新Module状态, 若原状态与目标状态不一致, 则根据情况处理server和路由
func (c *moduleController) UpdateStatus(ctx *fiber.Ctx) error {
	instance := new(model.Module)
	if err := ctx.BodyParser(instance); err != nil {
		logger.Errorln(err)
		return ctx.JSON(&server.CommonResponse{
			Code: server.ResponseCodeParamParseError,
			Msg:  server.ResponseMsgParamParseError,
		})
	}
	// 检查ID和status是否传递
	if instance.ID == 0 || instance.Status == 0 {
		return ctx.JSON(&server.CommonResponse{
			Code: server.ResponseCodeParamParseError,
			Msg:  server.ResponseMsgParamParseError,
		})
	}
	// 获取数据库中的module
	module, err := service.ModuleService.Instance(instance.ID)
	if err != nil {
		return ctx.JSON(&server.CommonResponse{
			Code: server.ResponseCodeDatabase,
			Msg:  server.ResponseMsgDatabase + err.Error(),
		})
	}
	// 若原状态与目标状态不一致, 则根据情况处理server和路由
	if module.Status != instance.Status {
		// 若目标状态为启用, 则注册module
		if instance.Status == 1 {
			manager.RegisterModule(module)
			go func() {
				_ = server.Shutdown() // 重启server
			}()
		}
		// 若目标状态为禁用, 则注销module
		if instance.Status == -1 {
			manager.UnregisterModule(module.Name)
		}

		// 更新数据库状态
		err = service.ModuleService.UpdateStatus(instance.ID, instance.Status)
		if err != nil {
			return ctx.JSON(&server.CommonResponse{
				Code: server.ResponseCodeDatabase,
				Msg:  server.ResponseMsgDatabase + err.Error(),
			})
		}
	}
	// 直接返回成功
	return ctx.JSON(&server.CommonResponse{Data: true})
}
