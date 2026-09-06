package env

import (
	"context"
	"strings"

	"github.com/knadh/koanf/maps"
	koanftoml "github.com/knadh/koanf/parsers/toml"
	koanfyaml "github.com/knadh/koanf/parsers/yaml"
	koanf "github.com/knadh/koanf/v2"
	"goark.dev/goark/core/resource"
	arkerrors "goark.dev/goark/errors"
)

func preparePropertySourceLoad(
	ctx context.Context,
	loader resource.Loader,
	location string,
	options []PropertySourceLoadOption,
) (string, propertySourceLoadOptions, error) {
	if ctx == nil {
		return "", propertySourceLoadOptions{}, arkerrors.New(
			arkerrors.CodeInvalidArgument,
			"context is nil",
		)
	}
	if loader == nil {
		return "", propertySourceLoadOptions{}, arkerrors.New(
			arkerrors.CodeInvalidArgument,
			"resource loader is nil",
		)
	}
	location = strings.TrimSpace(location)
	if location == "" {
		return "", propertySourceLoadOptions{}, arkerrors.New(
			arkerrors.CodeInvalidArgument,
			"property source location is empty",
		)
	}
	loadOptions := newPropertySourceLoadOptions(options)
	if err := validatePropertySourceEncoding(loadOptions.encoding); err != nil {
		return "", propertySourceLoadOptions{}, err
	}
	if loadOptions.nameSet && loadOptions.name == "" {
		return "", propertySourceLoadOptions{}, arkerrors.New(
			arkerrors.CodeInvalidArgument,
			"property source name is empty",
		)
	}
	return location, loadOptions, nil
}

func loadConfigCandidate(
	ctx context.Context,
	loader resource.Loader,
	location string,
	format ConfigFormat,
	options propertySourceLoadOptions,
) (*ConfigPropertySource, error) {
	res, err := loader.Load(location)
	if err != nil {
		return nil, err
	}
	exists, err := res.Exists(ctx)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, arkerrors.Newf(arkerrors.CodeNotFound, "property source %q not found", location)
	}
	data, err := res.ReadAll(ctx)
	if err != nil {
		return nil, err
	}
	values, err := ParseConfig(format, data)
	if err != nil {
		return nil, err
	}
	name := location
	if options.nameSet && options.name != "" {
		name = options.name
	}
	return NewConfigPropertySource(name, values)
}

// LoadPropertiesPropertySource 从资源位置加载 .properties 配置源。
func LoadPropertiesPropertySource(
	ctx context.Context,
	loader resource.Loader,
	location string,
	options ...PropertiesLoadOption,
) (*PropertiesPropertySource, error) {
	location, loadOptions, err := preparePropertySourceLoad(ctx, loader, location, options)
	if err != nil {
		return nil, err
	}
	source, err := loadConfigCandidate(ctx, loader, location, ConfigFormatProperties, loadOptions)
	if err != nil {
		if loadOptions.ignoreResourceNotFound && arkerrors.Is(err, arkerrors.CodeNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return source, nil
}

func koanfParserForFormat(format ConfigFormat) (koanf.Parser, error) {
	switch format {
	case ConfigFormatYAML:
		return koanfyaml.Parser(), nil
	case ConfigFormatProperties:
		return propertiesKoanfParser{}, nil
	case ConfigFormatTOML:
		return koanftoml.Parser(), nil
	default:
		return nil, arkerrors.Newf(
			arkerrors.CodeInvalidArgument,
			"config format %q is not supported",
			format,
		)
	}
}

const (
	// DefaultConfigBaseName 是默认配置文件基础名称。
	DefaultConfigBaseName = "app"

	// ConfigFormatYAML 表示 YAML 配置，支持 .yml 与 .yaml 扩展名。
	ConfigFormatYAML ConfigFormat = "yml"
	// ConfigFormatProperties 表示 Java .properties 风格配置。
	ConfigFormatProperties ConfigFormat = "properties"
	// ConfigFormatTOML 表示 TOML 配置。
	ConfigFormatTOML ConfigFormat = "toml"
)

func newPropertySourceLoadOptions(options []PropertySourceLoadOption) propertySourceLoadOptions {
	out := propertySourceLoadOptions{encoding: "utf-8"}
	for _, option := range options {
		if option != nil {
			option(&out)
		}
	}
	return out
}

// LoadDefaultConfigPropertySource 按默认名称加载 app.yml、app.properties 或 app.toml 配置源。
func LoadDefaultConfigPropertySource(
	ctx context.Context,
	loader resource.Loader,
	options ...PropertySourceLoadOption,
) (*ConfigPropertySource, error) {
	return LoadConfigPropertySource(ctx, loader, DefaultConfigBaseName, options...)
}

func (propertiesKoanfParser) Unmarshal(data []byte) (map[string]any, error) {
	values, err := ParseProperties(data)
	if err != nil {
		return nil, err
	}
	return maps.Unflatten(values, "."), nil
}

type propertySourceLoadOptions struct {
	name                   string
	nameSet                bool
	encoding               string
	ignoreResourceNotFound bool
}

// WithIgnoreResourceNotFound 设置资源不存在时是否忽略。
func WithIgnoreResourceNotFound(ignore bool) PropertySourceLoadOption {
	return func(options *propertySourceLoadOptions) {
		options.ignoreResourceNotFound = ignore
	}
}

// ParseYAML 解析 YAML 配置文本，并将对象层级展开为点号风格配置键。
func ParseYAML(data []byte) (map[string]any, error) {
	return parseKoanfConfig(ConfigFormatYAML, data)
}

func (propertiesKoanfParser) Marshal(map[string]any) ([]byte, error) {
	return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "properties marshal is not supported")
}

// PropertySourceLoadOption 调整配置源加载行为。
type PropertySourceLoadOption func(*propertySourceLoadOptions)

type propertiesKoanfParser struct{}
