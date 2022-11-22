package ruomu_module

import (
	"os"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	logger "github.com/sirupsen/logrus"
	"github.com/yockii/ruomu-core/database"
	"github.com/yockii/ruomu-core/shared"

	"github.com/yockii/ruomu-module/model"
)

var defaultManager *manager

type manager struct {
	modules       map[string]*model.Module
	pluginMap     map[string]plugin.Plugin
	pluginClient  map[string]*plugin.Client
	moduleExecMap map[string]shared.Communicate
}

func Initial() (err error) {
	defaultManager = &manager{
		modules:      make(map[string]*model.Module),
		pluginMap:    make(map[string]plugin.Plugin),
		pluginClient: make(map[string]*plugin.Client),
	}

	var modules []*model.Module
	err = database.DB.Find(&modules, &model.Module{Status: 1})
	if err != nil {
		logger.Errorln(err)
		return
	}

	// 已注册模块进行加载
	for _, module := range modules {
		RegisterModule(module)
	}
	return
}

var handshake = plugin.HandshakeConfig{
	ProtocolVersion: 1,
}

// registerModule 注入模块
func (m *manager) registerModule(module *model.Module) {
	m.pluginMap[module.Name] = &shared.CommunicatePlugin{}
	moduleName := module.Name
	if _, has := m.modules[moduleName]; has {
		logger.Warnln("模块: ", moduleName, "已存在, 忽略该模块")
		return
	}
	logger.Infoln("加载模块: ", moduleName)
	// 加载模块
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  handshake,
		Plugins:          m.pluginMap,
		Cmd:              exec.Command("sh", "-c", module.Cmd),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Logger: hclog.New(&hclog.LoggerOptions{
			Name:   moduleName,
			Output: os.Stdout,
			Level:  hclog.Debug,
		}),
	})
	cp, err := client.Client()
	if err != nil {
		logger.Errorln("模块", moduleName, "加载失败", err)
		client.Kill()
		return
	}
	raw, err := cp.Dispense(moduleName)
	if err != nil {
		logger.Errorln("模块", moduleName, "加载失败", err)
		cp.Close()
		client.Kill()
		return
	}

	instance := raw.(shared.Communicate)
	// TODO 对应的参数信息
	err = instance.Initial(map[string]string{})
	if err != nil {
		logger.Errorln(err)
		logger.Warnln("模块", moduleName, "加载失败")
		return
	}

	m.modules[moduleName] = module
	m.pluginClient[moduleName] = client
	m.moduleExecMap[moduleName] = instance
	logger.Info("模块", moduleName, "加载完毕")
}

func (m *manager) destroy() {
	for _, client := range m.pluginClient {
		client.Kill()
	}
}

// RegisterModule 注入模块
func RegisterModule(module *model.Module) {
	defaultManager.registerModule(module)
}

func Destroy() {
	defaultManager.destroy()
}
