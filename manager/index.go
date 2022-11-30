package manager

import (
	"encoding/json"
	"os"
	"os/exec"

	"github.com/gofiber/fiber/v2"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/sirupsen/logrus"
	"github.com/yockii/ruomu-core/database"
	"github.com/yockii/ruomu-core/server"
	"github.com/yockii/ruomu-core/shared"

	"github.com/yockii/ruomu-module/model"
)

var defaultManager = &Manager{
	modules:           make(map[string]*model.Module),
	pluginMap:         make(map[string]plugin.Plugin),
	pluginClient:      make(map[string]*plugin.Client),
	moduleExecMap:     make(map[string]shared.Communicate),
	moduleInjectCodes: make(map[string][]string),
}

type Manager struct {
	modules           map[string]*model.Module
	pluginMap         map[string]plugin.Plugin
	pluginClient      map[string]*plugin.Client
	moduleExecMap     map[string]shared.Communicate
	moduleInjectCodes map[string][]string
}

// RegisterModule 注入模块
func (m *Manager) RegisterModule(module *model.Module) {
	m.pluginMap[module.Name] = &shared.CommunicatePlugin{}
	moduleName := module.Name
	if _, has := m.modules[moduleName]; has {
		logrus.Warnln("模块: ", moduleName, "已存在, 忽略该模块")
		return
	}
	logrus.Infoln("开始加载模块: ", moduleName)

	var err error
	// 加载模块
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  shared.Handshake,
		Plugins:          m.pluginMap,
		Cmd:              exec.Command(module.Cmd),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Logger: hclog.New(&hclog.LoggerOptions{
			Name:   moduleName,
			Output: os.Stdout,
			Level:  hclog.Debug,
		}),
	})
	defer func() {
		if err != nil {
			client.Kill()
		}
	}()

	var cp plugin.ClientProtocol
	cp, err = client.Client()
	if err != nil {
		logrus.Errorln("模块", moduleName, "加载失败", err)
		return
	}
	defer func() {
		if err != nil {
			cp.Close()
		}
	}()
	var raw interface{}
	raw, err = cp.Dispense(moduleName)
	if err != nil {
		logrus.Errorln("模块", moduleName, "加载失败", err)
		return
	}

	instance := raw.(shared.Communicate)

	logrus.Infoln("模块【", moduleName, "】加载完成，进行初始化...")

	// 查询模块参数
	var settings []*model.ModuleSettings
	if err = database.DB.Find(&settings, &model.ModuleSettings{ModuleId: module.Id}); err != nil {
		logrus.Errorln(err)
		return
	}
	var params = make(map[string]string)
	for _, setting := range settings {
		params[setting.Code] = setting.Value
	}
	err = instance.Initial(params)
	if err != nil {
		logrus.Errorln(err)
		logrus.Warnln("模块【", moduleName, "】初始化失败")
		return
	}

	logrus.Infoln("开始注入模块【", moduleName, "】HTTP请求接口")
	// 注入http请求
	var injects []*model.ModuleInjectInfo
	if err = database.DB.Find(&injects, &model.ModuleInjectInfo{
		ModuleId: module.Id,
	}); err != nil {
		logrus.Errorln(err)
		return
	}
	var injectCodes []string
	for _, inject := range injects {
		switch inject.Type {
		case 1:
			server.Get(inject.InjectCode, m.checkAuthorization(inject), m.handleGet(moduleName, inject.InjectCode))
		case 2:
			server.Post(inject.InjectCode, m.checkAuthorization(inject), m.handlePost(moduleName, inject.InjectCode))
		case 3:
			server.Put(inject.InjectCode, m.checkAuthorization(inject), m.handlePost(moduleName, inject.InjectCode))
		case 4:
			server.Delete(inject.InjectCode, m.checkAuthorization(inject), m.handleGet(moduleName, inject.InjectCode))
		}
		logrus.Infoln("模块【"+moduleName+"】成功注入HTTP请求:", inject.InjectCode)
		injectCodes = append(injectCodes, inject.InjectCode)
	}

	m.modules[moduleName] = module
	m.pluginClient[moduleName] = client
	m.moduleExecMap[moduleName] = instance
	m.moduleInjectCodes[moduleName] = injectCodes
	logrus.Info("模块", moduleName, "初始化完毕")
}

func (m *Manager) handleGet(moduleName string, code string) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		moduleExec, has := m.moduleExecMap[moduleName]
		if has {
			ps := ctx.AllParams()
			ctx.Context().QueryArgs().VisitAll(func(key, value []byte) {
				ps[string(key)] = string(value)
			})
			v, _ := json.Marshal(ps)

			result, err := moduleExec.InjectCall(code, ctx.GetReqHeaders(), v)
			if err != nil {
				logrus.Errorln(err)
				return ctx.JSON(&server.CommonResponse{
					Code: server.ResponseCodeUnknownError,
					Msg:  err.Error(),
				})
			}
			ctx.Response().Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
			return ctx.Send(result)
		}
		return ctx.JSON(&server.CommonResponse{
			Code: server.ResponseCodeModuleNotExists,
			Msg:  server.ResponseMsgModuleNotExists,
		})
	}
}

func (m *Manager) handlePost(moduleName string, code string) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		moduleExec, has := m.moduleExecMap[moduleName]
		if has {
			v := ctx.Body()
			result, err := moduleExec.InjectCall(code, ctx.GetReqHeaders(), v)
			if err != nil {
				logrus.Errorln(err)
				return ctx.JSON(&server.CommonResponse{
					Code: server.ResponseCodeUnknownError,
					Msg:  err.Error(),
				})
			}
			ctx.Response().Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
			return ctx.Send(result)
		}
		return ctx.JSON(&server.CommonResponse{
			Code: server.ResponseCodeModuleNotExists,
			Msg:  server.ResponseMsgModuleNotExists,
		})
	}
}

func (m *Manager) Destroy() {
	for name, client := range m.pluginClient {
		client.Kill()
		delete(m.moduleInjectCodes, name)
		delete(m.moduleExecMap, name)
		delete(m.pluginClient, name)
		delete(m.modules, name)
	}
}

func (m *Manager) UnregisterModule(name string) {
	pc, has := m.pluginClient[name]
	if has {
		pc.Kill()
	}
	delete(m.moduleInjectCodes, name)
	delete(m.moduleExecMap, name)
	delete(m.pluginClient, name)
	delete(m.modules, name)
}

// RegisterModule 注入模块
func RegisterModule(module *model.Module) {
	defaultManager.RegisterModule(module)
}

func Destroy() {
	defaultManager.Destroy()
}
