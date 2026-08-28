package env

import (
	"context"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/knadh/koanf/maps"
	koanftoml "github.com/knadh/koanf/parsers/toml"
	koanfyaml "github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"
	koanf "github.com/knadh/koanf/v2"
	"goark.dev/goark/core/resource"
	arkerrors "goark.dev/goark/errors"
)

// ConfigFormat 表示配置文件格式。
type ConfigFormat string

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

// PropertySourceLoadOption 调整配置源加载行为。
type PropertySourceLoadOption func(*propertySourceLoadOptions)

// PropertiesLoadOption 保留旧版 properties 加载 Option 名称。
type PropertiesLoadOption = PropertySourceLoadOption

type propertySourceLoadOptions struct {
	name                   string
	nameSet                bool
	encoding               string
	ignoreResourceNotFound bool
}

var configSearchOrder = []struct {
	extension string
	format    ConfigFormat
}{
	{extension: ".yml", format: ConfigFormatYAML},
	{extension: ".properties", format: ConfigFormatProperties},
	{extension: ".toml", format: ConfigFormatTOML},
	{extension: ".yaml", format: ConfigFormatYAML},
}

// WithPropertySourceName 指定加载后的 PropertySource 名称。
func WithPropertySourceName(name string) PropertySourceLoadOption {
	return func(options *propertySourceLoadOptions) {
		options.name = strings.TrimSpace(name)
		options.nameSet = true
	}
}

// WithPropertySourceEncoding 指定配置文本编码；V1 只支持 UTF-8。
func WithPropertySourceEncoding(encoding string) PropertySourceLoadOption {
	return func(options *propertySourceLoadOptions) {
		options.encoding = strings.TrimSpace(strings.ToLower(encoding))
	}
}

// WithIgnoreResourceNotFound 设置资源不存在时是否忽略。
func WithIgnoreResourceNotFound(ignore bool) PropertySourceLoadOption {
	return func(options *propertySourceLoadOptions) {
		options.ignoreResourceNotFound = ignore
	}
}

// LoadConfigPropertySource 从资源位置加载 yml/properties/toml 配置源。
func LoadConfigPropertySource(ctx context.Context, loader resource.Loader, location string, options ...PropertySourceLoadOption) (*ConfigPropertySource, error) {
	location, loadOptions, err := preparePropertySourceLoad(ctx, loader, location, options)
	if err != nil {
		return nil, err
	}

	extension := configLocationExtension(location)
	if extension != "" {
		format, ok := configFormatByExtension(extension)
		if !ok {
			return nil, arkerrors.Newf(arkerrors.CodeInvalidArgument, "config format %q is not supported", extension)
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
		source, err := loadConfigCandidate(ctx, loader, candidateLocation, candidate.format, loadOptions)
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
	return nil, arkerrors.Newf(arkerrors.CodeNotFound, "config property source %q not found; tried: %s", location, strings.Join(tried, ", "))
}

// LoadDefaultConfigPropertySource 按默认名称加载 app.yml/app.properties/app.toml/app.yaml 配置源。
func LoadDefaultConfigPropertySource(ctx context.Context, loader resource.Loader, options ...PropertySourceLoadOption) (*ConfigPropertySource, error) {
	return LoadConfigPropertySource(ctx, loader, DefaultConfigBaseName, options...)
}

// LoadPropertiesPropertySource 从资源位置加载 .properties 配置源。
func LoadPropertiesPropertySource(ctx context.Context, loader resource.Loader, location string, options ...PropertiesLoadOption) (*PropertiesPropertySource, error) {
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
		return nil, arkerrors.Newf(arkerrors.CodeInvalidArgument, "config format %q is not supported", format)
	}
}

// ParseYAML 解析 YAML 配置文本，并将对象层级展开为点号风格配置键。
func ParseYAML(data []byte) (map[string]any, error) {
	return parseKoanfConfig(ConfigFormatYAML, data)
}

// ParseTOML 解析 TOML 配置文本，并将表层级展开为点号风格配置键。
func ParseTOML(data []byte) (map[string]any, error) {
	return parseKoanfConfig(ConfigFormatTOML, data)
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
		return nil, arkerrors.Wrapf(arkerrors.CodeInvalidArgument, err, "%s config is invalid", format)
	}
	values := config.Raw()
	if values == nil {
		return map[string]any{}, nil
	}
	return flattenConfigMap(values)
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
		return nil, arkerrors.Newf(arkerrors.CodeInvalidArgument, "config format %q is not supported", format)
	}
}

type propertiesKoanfParser struct{}

func (propertiesKoanfParser) Unmarshal(data []byte) (map[string]any, error) {
	values, err := ParseProperties(data)
	if err != nil {
		return nil, err
	}
	return maps.Unflatten(values, "."), nil
}

func (propertiesKoanfParser) Marshal(map[string]any) ([]byte, error) {
	return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "properties marshal is not supported")
}

func preparePropertySourceLoad(ctx context.Context, loader resource.Loader, location string, options []PropertySourceLoadOption) (string, propertySourceLoadOptions, error) {
	if ctx == nil {
		return "", propertySourceLoadOptions{}, arkerrors.New(arkerrors.CodeInvalidArgument, "context is nil")
	}
	if loader == nil {
		return "", propertySourceLoadOptions{}, arkerrors.New(arkerrors.CodeInvalidArgument, "resource loader is nil")
	}
	location = strings.TrimSpace(location)
	if location == "" {
		return "", propertySourceLoadOptions{}, arkerrors.New(arkerrors.CodeInvalidArgument, "property source location is empty")
	}
	loadOptions := newPropertySourceLoadOptions(options)
	if err := validatePropertySourceEncoding(loadOptions.encoding); err != nil {
		return "", propertySourceLoadOptions{}, err
	}
	if loadOptions.nameSet && loadOptions.name == "" {
		return "", propertySourceLoadOptions{}, arkerrors.New(arkerrors.CodeInvalidArgument, "property source name is empty")
	}
	return location, loadOptions, nil
}

func newPropertySourceLoadOptions(options []PropertySourceLoadOption) propertySourceLoadOptions {
	out := propertySourceLoadOptions{encoding: "utf-8"}
	for _, option := range options {
		if option != nil {
			option(&out)
		}
	}
	return out
}

func validatePropertySourceEncoding(encoding string) error {
	if encoding == "" || encoding == "utf-8" || encoding == "utf8" {
		return nil
	}
	return arkerrors.Newf(arkerrors.CodeInvalidArgument, "property source encoding %q is not supported", encoding)
}

func loadConfigCandidate(ctx context.Context, loader resource.Loader, location string, format ConfigFormat, options propertySourceLoadOptions) (*ConfigPropertySource, error) {
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

func configLocationExtension(location string) string {
	location = strings.TrimSpace(location)
	if index := strings.IndexAny(location, "?#"); index >= 0 {
		location = location[:index]
	}
	return strings.ToLower(filepath.Ext(location))
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

func appendConfigExtension(location string, extension string) string {
	if index := strings.IndexAny(location, "?#"); index >= 0 {
		return location[:index] + extension + location[index:]
	}
	return location + extension
}

func flattenConfigMap(values map[string]any) (map[string]any, error) {
	out := make(map[string]any)
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "config key is empty")
		}
		if err := flattenConfigValue(key, value, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func flattenConfigValue(prefix string, value any, out map[string]any) error {
	switch typed := value.(type) {
	case map[string]any:
		return flattenStringMap(prefix, typed, out)
	case map[any]any:
		return flattenAnyMap(prefix, typed, out)
	case []any:
		return flattenSlice(prefix, typed, out)
	default:
		return flattenReflectValue(prefix, value, out)
	}
}

func flattenStringMap(prefix string, values map[string]any, out map[string]any) error {
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			return arkerrors.New(arkerrors.CodeInvalidArgument, "config key is empty")
		}
		if err := flattenConfigValue(prefix+"."+key, value, out); err != nil {
			return err
		}
	}
	return nil
}

func flattenAnyMap(prefix string, values map[any]any, out map[string]any) error {
	for key, value := range values {
		keyText, ok := key.(string)
		if !ok {
			return arkerrors.Newf(arkerrors.CodeInvalidArgument, "config key %v is not a string", key)
		}
		keyText = strings.TrimSpace(keyText)
		if keyText == "" {
			return arkerrors.New(arkerrors.CodeInvalidArgument, "config key is empty")
		}
		if err := flattenConfigValue(prefix+"."+keyText, value, out); err != nil {
			return err
		}
	}
	return nil
}

func flattenSlice(prefix string, values []any, out map[string]any) error {
	out[prefix] = values
	for index, value := range values {
		key := prefix + "[" + strconv.Itoa(index) + "]"
		if err := flattenConfigValue(key, value, out); err != nil {
			return err
		}
	}
	return nil
}

func flattenReflectValue(prefix string, value any, out map[string]any) error {
	if value == nil {
		out[prefix] = nil
		return nil
	}
	reflectValue := reflect.ValueOf(value)
	switch reflectValue.Kind() {
	case reflect.Map:
		if reflectValue.Type().Key().Kind() != reflect.String {
			return arkerrors.Newf(arkerrors.CodeInvalidArgument, "config key under %q is not string", prefix)
		}
		for _, key := range reflectValue.MapKeys() {
			keyText := strings.TrimSpace(key.String())
			if keyText == "" {
				return arkerrors.New(arkerrors.CodeInvalidArgument, "config key is empty")
			}
			if err := flattenConfigValue(prefix+"."+keyText, reflectValue.MapIndex(key).Interface(), out); err != nil {
				return err
			}
		}
		return nil
	case reflect.Slice, reflect.Array:
		values := make([]any, 0, reflectValue.Len())
		for index := 0; index < reflectValue.Len(); index++ {
			values = append(values, reflectValue.Index(index).Interface())
		}
		return flattenSlice(prefix, values, out)
	default:
		out[prefix] = normalizeConfigScalar(value)
		return nil
	}
}

func normalizeConfigScalar(value any) any {
	return value
}
