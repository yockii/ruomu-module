package domain

import "github.com/yockii/ruomu-module/model"

type Module struct {
	model.Module
	Dependencies []*model.ModuleDependency `json:"dependencies,omitempty"`
	Injects      []*model.ModuleInjectInfo `json:"injects,omitempty"`
	Settings     []*model.ModuleSettings   `json:"settings,omitempty"`
}
