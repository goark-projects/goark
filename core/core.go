package core

// FrameworkName 是 Goark 核心框架名称。
const FrameworkName = "goark"

// Version 表示当前核心库版本。正式发布前保持 dev。
const Version = "dev"

// Info 描述 Goark 核心库的静态信息。
type Info struct {
	Name    string
	Version string
}

// BuildInfo 返回核心库静态信息。
func BuildInfo() Info {
	return Info{
		Name:    FrameworkName,
		Version: Version,
	}
}
