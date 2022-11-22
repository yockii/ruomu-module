package model

import "github.com/yockii/ruomu-core/database"

type Module struct {
	Id         int64             `json:"id,omitempty" xorm:"pk"`
	Name       string            `json:"name,omitempty"`
	Code       string            `json:"code,omitempty"`
	Cmd        string            `json:"cmd,omitempty"`
	Mode       int               `json:"mode,omitempty"`   // 模块模式，1-请求转发
	Status     int               `json:"status,omitempty"` // 状态 1-启用 -1-禁用
	CreateTime database.DateTime `json:"createTime" xorm:"created"`
}

func (_ Module) TableComment() string {
	return "模块表"
}

type ModuleDependence struct {
	Id             int64  `json:"id,omitempty" xorm:"pk"`
	ModuleCode     string `json:"moduleCode,omitempty"`
	DependenceCode string `json:"dependenceCode,omitempty"`
}

func (_ ModuleDependence) TableComment() string {
	return "模块依赖"
}

type ModuleHttpRequest struct {
	Id         int64  `json:"id,omitempty" xorm:"pk"`
	ModuleId   int64  `json:"moduleId,omitempty"`
	Code       string `json:"code,omitempty"`       // 代码
	Type       int    `json:"type,omitempty"`       // 类型  1-http_get, 2-http_post, 3-http_put, 4-http_delete, 51-hook
	InjectCode string `json:"injectCode,omitempty"` // 注入点（http请求路径或注入点代码）
}

func (_ ModuleHttpRequest) TableComment() string {
	return "模块功能注入,系统将根据注入信息调用模块方法"
}
