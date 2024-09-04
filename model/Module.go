package model

type Module struct {
	ID         uint64 `json:"id,omitempty,string" gorm:"primaryKey"`
	Name       string `json:"name,omitempty" gorm:"comment:模块名称"`
	Code       string `json:"code,omitempty" gorm:"size:50;index;comment:模块代码"`
	Cmd        string `json:"cmd,omitempty" gorm:"size:500;comment:模块执行命令"`
	Status     int    `json:"status,omitempty" gorm:"comment:模块状态 1-启用 -1-禁用"` // 状态 1-启用 -1-禁用
	CreateTime int64  `json:"createTime" gorm:"autoCreateTime"`
}

func (_ Module) TableComment() string {
	return "模块表"
}

type ModuleDependency struct {
	ID             uint64 `json:"id,omitempty,string" gorm:"primaryKey"`
	ModuleCode     string `json:"moduleCode,omitempty" gorm:"size:50;index;comment:模块代码"`
	DependenceCode string `json:"dependenceCode,omitempty" gorm:"comment:依赖的模块代码"`
}

func (_ ModuleDependency) TableComment() string {
	return "模块依赖"
}

type ModuleInjectInfo struct {
	ID                uint64 `json:"id,omitempty,string" gorm:"primaryKey"`
	ModuleID          uint64 `json:"moduleId,omitempty,string" gorm:"comment:模块ID"`
	Name              string `json:"name,omitempty" gorm:"comment:注入的名称"`                                                                                                                   // 名称
	Type              int    `json:"type,omitempty" gorm:"comment:类型  1-json_get, 2-json_post, 3-json_put, 4-json_delete, 11-html_get, 12-html_post, 13-html_put, 14-html_delete, 51-hook"` // 类型  1-http_get, 2-http_post, 3-http_put, 4-http_delete, 51-hook
	InjectCode        string `json:"injectCode,omitempty" gorm:"comment:注入点代码，http请求路径或定义的注入点"`                                                                                             // 注入点（http请求路径或注入点代码）
	AuthorizationCode string `json:"authorizationCode,omitempty" gorm:"comment:授权代码 anon或空表示不需要权限 user-需要登录 其他-需要具体对应的资源权限"`                                                                // 权限代码 特殊用例：anno或空-不需要权限  user-需要登录 其他-需要具体对应的资源权限
}

func (_ ModuleInjectInfo) TableComment() string {
	return "模块功能注入,系统将根据注入信息调用模块方法"
}

type ModuleSettings struct {
	ID       uint64 `json:"id,omitempty,string" gorm:"primaryKey"`
	ModuleID uint64 `json:"moduleId,omitempty,string" gorm:"comment:模块ID"`
	Code     string `json:"code,omitempty" gorm:"comment:配置键"`
	Value    string `json:"value,omitempty" gorm:"comment:配置值"`
}

func (_ ModuleSettings) TableComment() string {
	return "模块参数配置，设置传递给模块的配置信息"
}
