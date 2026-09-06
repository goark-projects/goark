package env

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/providers/rawbytes"
	koanf "github.com/knadh/koanf/v2"
	"goark.dev/goark/core/resource"
	arkerrors "goark.dev/goark/errors"
)

// LoadConfigPropertySource 从资源位置加载 yml/properties/toml 配置源。
func LoadConfigPropertySource(
	ctx context.Context,
	loader resource.Loader,
	location string,
	options ...PropertySourceLoadOption,
) (*ConfigPropertySource, error) {
	location, loadOptions, err := preparePropertySourceLoad(ctx, loader, location, options)
	if err != nil {
		return nil, err
	}

	extension := configLocationExtension(location)
	if extension != "" {
		format, ok := configFormatByExtension(extension)
		if !ok {
			return nil, arkerrors.Newf(
				arkerrors.CodeInvalidArgument,
				"config format %q is not supported",
				extension,
			)
		}
		source, err := loadConfigCandidate(ctx, loader, location, format, loadOptions)
		if err != nil {
			if loadOptions.ignoreResourceNotFound && arkerrors.Is(err, arkerrors.CodeNotFound) {
				return nil, nil
			}
			return nil, err
		}
		return source, nil
	}

	tried := make([]string, 0, len(configSearchOrder))
	for _, candidate := range configSearchOrder {
		candidateLocation := appendConfigExtension(location, candidate.extension)
		tried = append(tried, candidateLocation)
		source, err := loadConfigCandidate(
			ctx,
			loader,
			candidateLocation,
			candidate.format,
			loadOptions,
		)
		if err == nil {
			return source, nil
		}
		if !arkerrors.Is(err, arkerrors.CodeNotFound) {
			return nil, err
		}
	}
	if loadOptions.ignoreResourceNotFound {
		return nil, nil
	}
	return nil, arkerrors.Newf(
		arkerrors.CodeNotFound,
		"config property source %q not found; tried: %s",
		location,
		strings.Join(tried, ", "),
	)
}

func parseKoanfConfig(format ConfigFormat, data []byte) (map[string]any, error) {
	if strings.TrimSpace(string(data)) == "" {
		return map[string]any{}, nil
	}
	parser, err := koanfParserForFormat(format)
	if err != nil {
		return nil, err
	}
	config := koanf.New(".")
	if err := config.Load(rawbytes.Provider(data), parser); err != nil {
		return nil, arkerrors.Wrapf(
			arkerrors.CodeInvalidArgument,
			err,
			"%s config is invalid",
			format,
		)
	}
	values := config.Raw()
	if values == nil {
		return map[string]any{}, nil
	}
	return flattenConfigMap(values)
}

// ParseConfig 按指定格式解析配置文本，并输出点号风格配置键。
func ParseConfig(format ConfigFormat, data []byte) (map[string]any, error) {
	switch format {
	case ConfigFormatYAML:
		return ParseYAML(data)
	case ConfigFormatProperties:
		return parseKoanfConfig(ConfigFormatProperties, data)
	case ConfigFormatTOML:
		return ParseTOML(data)
	default:
		return nil, arkerrors.Newf(
			arkerrors.CodeInvalidArgument,
			"config format %q is not supported",
			format,
		)
	}
}

func configFormatByExtension(extension string) (ConfigFormat, bool) {
	switch strings.ToLower(extension) {
	case ".yml", ".yaml":
		return ConfigFormatYAML, true
	case ".properties":
		return ConfigFormatProperties, true
	case ".toml":
		return ConfigFormatTOML, true
	default:
		return "", false
	}
}

func validatePropertySourceEncoding(encoding string) error {
	if encoding == "" || encoding == "utf-8" || encoding == "utf8" {
		return nil
	}
	return arkerrors.Newf(
		arkerrors.CodeInvalidArgument,
		"property source encoding %q is not supported",
		encoding,
	)
}

var configSearchOrder = []struct {
	extension string
	format    ConfigFormat
}{
	{extension: ".yml", format: ConfigFormatYAML},
	{extension: ".properties", format: ConfigFormatProperties},
	{extension: ".toml", format: ConfigFormatTOML},
}

// WithPropertySourceName 指定加载后的 PropertySource 名称。
func WithPropertySourceName(name string) PropertySourceLoadOption {
	return func(options *propertySourceLoadOptions) {
		options.name = strings.TrimSpace(name)
		options.nameSet = true
	}
}

func configLocationExtension(location string) string {
	location = strings.TrimSpace(location)
	if index := strings.IndexAny(location, "?#"); index >= 0 {
		location = location[:index]
	}
	return strings.ToLower(filepath.Ext(location))
}

// WithPropertySourceEncoding 指定配置文本编码；V1 只支持 UTF-8。
func WithPropertySourceEncoding(encoding string) PropertySourceLoadOption {
	return func(options *propertySourceLoadOptions) {
		options.encoding = strings.TrimSpace(strings.ToLower(encoding))
	}
}

func appendConfigExtension(location string, extension string) string {
	if index := strings.IndexAny(location, "?#"); index >= 0 {
		return location[:index] + extension + location[index:]
	}
	return location + extension
}

// ParseTOML 解析 TOML 配置文本，并将表层级展开为点号风格配置键。
func ParseTOML(data []byte) (map[string]any, error) {
	return parseKoanfConfig(ConfigFormatTOML, data)
}

// ConfigFormat 表示配置文件格式。
type ConfigFormat string

// PropertiesLoadOption 保留旧版 properties 加载 Option 名称。
type PropertiesLoadOption = PropertySourceLoadOption
