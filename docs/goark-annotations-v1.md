# Goark 注解规范 V1

## 状态

本文档定义 Goark 第一版注解系统的落地规范。V1 覆盖 `goark` 核心库的 DI/IoC、Bean 装配、Environment/PropertyResolver、Configuration/ApplicationContext、条件装配、排序和 Scope，并覆盖 `goark/web` 与 `goark/web/mvc` 当前已实现的 Web MVC 注解切片。不包含 `boot` 自动配置、starter、事务、SQL ORM、OpenAPI、安全权限等扩展能力。

Goark 注解是写在 Go 注释中的编译期元数据，由 `goark` CLI 扫描 Go AST 后生成普通 Go 代码。核心 DI 生成代码只依赖 `goark` 核心库；Web MVC 生成代码依赖 `goark/web`、`goark/web/mvc` 和 Arkarta Web 请求上下文。运行时不做 classpath scan、不做反射扫描、不依赖全局 `init()` 自动注册。

## 设计目标

- 提供接近 Spring Framework 的开发体验。
- 保持 Go 代码可读、可编译、可审查。
- 生成代码确定、显式、可提交。
- 支持私有字段注入，生成代码放在同一个 Go package 内。
- 依赖关系默认由生成器基于注入点自动推导，不要求业务代码重复书写初始化依赖。
- 让生成器产出的代码保持显式依赖边界，不依赖 `boot` 或 starter 模块。

## 非目标

- 不支持 Java 注解语法，例如 `@Service`。
- 不支持运行时扫描包、类型或方法。
- 不支持隐藏式全局 Bean 注册。
- V1 不实现 `boot` 自动配置、`ConfigurationProperties`、项目启动约定或 starter 机制。
- V1 不实现事务、SQL、OpenAPI、权限元数据。
- 不执行 Spring SpEL；`value` 使用 Goark 自有的受控 GaEL 表达式。

## 基础语法

Goark 注解统一使用下面两种形式：

```go
//goark:<annotation>
//goark:<annotation>(args...)
```

示例：

```go
//goark:service
//goark:service(name="apiResourceService")
//goark:order(value=100)
```

规范写法不在 `//` 和 `goark:` 之间加空格。CLI 可以兼容 `// goark:service`，但文档、生成器输出和示例必须使用 `//goark:service`。

### 注解名

注解名使用小写 kebab-case：

```go
//goark:service
//goark:profile("dev")
//goark:property-source("classpath:app.properties")
//goark:depends-on("database")
```

不要使用 Java 风格：

```go
//goark:@Service
//goark:RestController
```

### 参数

参数统一放在圆括号 `()` 中：

```go
//goark:service(name="apiResourceService")
//goark:autowired(required=false)
//goark:qualifier("userDao")
```

参数名尽量沿用对应 Spring/JSR 注解的 attribute 名称，例如 `name`、`value`、`required`、`encoding`、`ignoreResourceNotFound`。

V1 参数支持：

- 字符串：`"apiResourceService"`
- 整数：`100`、`-100`
- 布尔值：`true`、`false`

多个参数用英文逗号分隔：

```go
//goark:autowired(qualifier="masterDataSource", required=false)
```

同一个目标允许出现多个同名注解时，必须在具体注解小节中明确说明。未声明可重复的注解默认不可重复。

### value 简写

如果注解存在 `value` 参数，可以省略参数名：

```go
//goark:service("apiResourceService")
```

等价于：

```go
//goark:service(value="apiResourceService")
```

对于下列注解，`value` 与 `name` 等价：

```text
component
service
repository
configuration
bean
qualifier
named
resource
```

对于下列注解，省略参数名时表示 `value` 参数：

```text
value
order
priority
lazy
profile
conditional
depends-on
scope
property-source
property-sources
```

因此：

```go
//goark:bean("database")
```

等价于：

```go
//goark:bean(name="database")
```

`order` 的简写：

```go
//goark:order(100)
```

等价于：

```go
//goark:order(value=100)
```

`named` 的简写：

```go
//goark:named("userDao")
```

等价于：

```go
//goark:named(value="userDao")
```

`resource` 的简写：

```go
//goark:resource("userDao")
```

等价于：

```go
//goark:resource(name="userDao")
```

`profile` 的简写：

```go
//goark:profile("dev | test")
```

等价于：

```go
//goark:profile(value="dev | test")
```

`property-source` 的简写：

```go
//goark:property-source("classpath:app.properties")
```

等价于：

```go
//goark:property-source(value="classpath:app.properties")
```

## 参数选择器

函数或方法参数本身不能直接写 Go 注释。Goark 使用方括号 `[]` 表示“这个注解作用到当前函数或方法的哪个参数上”。

```go
//goark:qualifier[userDao]("userDao")
func (AdminConfiguration) UserService(userDao UserDao) *UserService {
	return &UserService{userDao: userDao}
}
```

语义：

```text
[] = 参数选择器
() = 注解参数
```

`[userDao]` 等价于 `[param="userDao"]`：

```go
//goark:qualifier[param="userDao"]("userDao")
```

规范主推短写：

```go
//goark:qualifier[userDao]("userDao")
```

### 参数选择器限制

V1 中 `[]` 只能用于函数或方法注解，并且只能选择当前函数或方法的参数名。

合法：

```go
//goark:qualifier[userDao]("userDao")
func (AdminConfiguration) UserService(userDao UserDao) *UserService {
	return &UserService{userDao: userDao}
}
```

非法：

```go
//goark:service[userService]
type UserService struct{}
```

非法原因：`service` 是类型注解，不能使用参数选择器。

如果选择器找不到对应参数，生成器必须报错：

```go
//goark:qualifier[userDao]("userDao")
func (AdminConfiguration) UserService(repo UserDao) *UserService {
	return &UserService{userDao: repo}
}
```

错误：

```text
goark: qualifier targets missing parameter "userDao" on AdminConfiguration.UserService
```

## V1 注解清单

| Goark 注解 | 对齐规范 | 作用位置 | 核心语义 |
| --- | --- | --- | --- |
| `component` | Spring `@Component` | 类型 | 注册普通 Bean |
| `service` | Spring `@Service` | 类型 | 注册业务服务 Bean |
| `repository` | Spring `@Repository` | 类型 | 注册数据访问 Bean |
| `configuration` | Spring `@Configuration` | 类型 | 声明配置装配单元 |
| `bean` | Spring `@Bean` | 方法 | 在 Configuration 中注册 Bean |
| `autowired` | Spring `@Autowired` | 字段、方法参数 | 按类型注入，支持可选依赖 |
| `qualifier` | Spring `@Qualifier` | 字段、方法参数 | 指定 Bean 名 |
| `value` | Spring `@Value` | 字段、方法参数 | 注入属性占位符、GaEL 表达式或字面量 |
| `inject` | JSR-330 `@Inject` | 字段、方法参数 | 按类型注入，始终必需 |
| `named` | JSR-330 `@Named` | 字段、方法参数 | 指定 Bean 名 |
| `resource` | JSR-250 `@Resource` | 字段、方法参数 | 按名称优先注入 |
| `order` | Spring `@Order` | 类型、Bean 方法 | 控制排序，数值越小越靠前 |
| `priority` | JSR-250 `@Priority` | 类型、Bean 方法 | 控制候选优先级和排序 |
| `primary` | Spring `@Primary` | 类型、Bean 方法 | 多候选按类型注入时优先 |
| `profile` | Spring `@Profile` | 类型、Bean 方法 | 按 Environment profile 条件注册 |
| `conditional` | Spring `@Conditional` | 类型、Bean 方法 | 按 Condition 条件注册 |
| `lazy` | Spring `@Lazy` | 类型、Bean 方法 | 延迟初始化 Bean |
| `depends-on` | Spring `@DependsOn` | 类型、Bean 方法 | 声明初始化依赖顺序 |
| `scope` | Spring `@Scope` | 类型、Bean 方法 | 声明 Bean 作用域 |
| `property-source` | Spring `@PropertySource` | Configuration 类型 | 加载属性资源 |
| `property-sources` | Spring `@PropertySources` | Configuration 类型 | 加载多个属性资源 |

### component

对齐 Spring `@Component`，注册普通组件 Bean。

位置：`type` 上方。

参数：

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `name` / `value` | string | 类型名 lowerCamel | Bean 名称 |

示例：

```go
//goark:component
type ClientAuthInterceptor struct {
	//goark:autowired
	securityProperties *SecurityProperties
}
```

显式命名：

```go
//goark:component("clientAuthInterceptor")
type ClientAuthInterceptor struct{}
```

### service

对齐 Spring `@Service`，注册业务服务 Bean。

位置：`type` 上方。

参数：

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `name` / `value` | string | 类型名 lowerCamel | Bean 名称 |

示例：

```go
//goark:service("apiResourceService")
type ApiResourceService struct {
	//goark:autowired
	repository *ApiResourceRepository
}
```

### repository

对齐 Spring `@Repository`，注册数据访问 Bean。

位置：`type` 上方。

参数：

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `name` / `value` | string | 类型名 lowerCamel | Bean 名称 |

示例：

```go
//goark:repository
type ApiResourceRepository struct {
	//goark:autowired
	db *sql.DB
}
```

V1 中 `repository` 与 `component` 的注册行为一致，只保留语义差异。`goark` 核心库不实现 SQL 访问、异常转换或 Mapper 生成。

### configuration

对齐 Spring `@Configuration`，声明配置装配单元。

位置：`type` 上方。

参数：

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `name` / `value` | string | 类型名 lowerCamel | Configuration 名称 |

示例：

```go
//goark:configuration("admin")
//goark:order(0)
type AdminConfiguration struct{}
```

生成器需要让该类型满足 `goark.Configuration`：

```go
func (AdminConfiguration) Name() string
func (AdminConfiguration) Order() int
func (AdminConfiguration) Register(ctx context.Context, registry *container.Registry) error
```

如果用户已经定义同名方法且签名不兼容，生成器必须报错。

### bean

对齐 Spring `@Bean`，在 `configuration` 方法上注册 Bean。

位置：带 `//goark:configuration` 类型的方法上方。

参数：

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `name` / `value` | string | 方法名 lowerCamel | Bean 名称 |

示例：

```go
//goark:configuration("admin")
type AdminConfiguration struct{}

//goark:bean("database")
func (AdminConfiguration) Database() (*sql.DB, error) {
	return sql.Open("postgres", "postgres://example")
}
```

带依赖：

```go
//goark:bean("userService")
func (AdminConfiguration) UserService(repository *ApiResourceRepository) *ApiResourceService {
	return &ApiResourceService{repository: repository}
}
```

方法参数默认按类型解析。如果需要指定 Bean 名，使用参数选择器：

```go
//goark:bean("userService")
//goark:qualifier[repository]("apiResourceRepository")
func (AdminConfiguration) UserService(repository *ApiResourceRepository) *ApiResourceService {
	return &ApiResourceService{repository: repository}
}
```

V1 支持的返回签名：

```go
func (...) T
func (...) (T, error)
```

不支持多个业务返回值：

```go
func (...) (A, B, error)
```

### order

对齐 Spring `@Order`，控制 Configuration 或组件排序。

位置：

- `type` 上方。
- 带 `//goark:bean` 的方法上方。

参数：

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `value` | int | `0` | 排序值，越小越靠前 |

示例：

```go
//goark:component
//goark:order(-100)
type ClientAuthInterceptor struct{}
```

Bean 方法示例：

```go
//goark:bean("cacheManager")
//goark:order(10)
func (AppConfiguration) CacheManager() *CacheManager {
	return NewCacheManager()
}
```

等价写法：

```go
//goark:order(value=-100)
```

如果目标类型未实现 `Order() int`，生成器可以生成：

```go
func (*ClientAuthInterceptor) Order() int {
	return -100
}
```

如果目标类型已实现 `Order() int`，以用户代码为准；如果注解值与用户方法同时存在，生成器必须报错，避免两个排序来源冲突。

`order` 用于集合注入、Configuration 注册顺序、生命周期回调顺序等需要稳定排序的场景。数值越小，优先级越高，越先执行。

### priority

对齐 JSR-250 `@Priority`，声明 Bean 的优先级。

位置：

- `type` 上方。
- 带 `//goark:bean` 的方法上方。

参数：

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `value` | int | 无 | 优先级，数值越小优先级越高 |

示例：

```go
//goark:service
//goark:priority(100)
type FastUserDao struct{}
```

`priority` 的规则：

1. 单值依赖按类型存在多个候选 Bean 时，先看 `primary`，再看 `priority`。
2. 只有一个最高优先级 Bean 时选择该 Bean；多个 Bean 拥有相同最高优先级时仍然报多候选错误。
3. 集合注入或 Ordered 列表排序时，如果目标没有 `order`，可以使用 `priority` 作为排序值。
4. 如果同一目标同时存在 `order` 和 `priority`，`order` 只负责排序，`priority` 只负责单值候选优先级。

### primary

对齐 Spring `@Primary`，声明同类型多 Bean 时的首选 Bean。

位置：

- `type` 上方。
- 带 `//goark:bean` 的方法上方。

参数：无。

示例：

```go
//goark:repository
//goark:primary
type PostgresUserDao struct{}
```

Bean 方法示例：

```go
//goark:bean("primaryDataSource")
//goark:primary
func (AppConfiguration) PrimaryDataSource() *sql.DB {
	return OpenPrimaryDataSource()
}
```

候选解析规则：

1. `qualifier`、`named`、`resource(name=...)` 等显式名称优先于 `primary`。
2. 按类型注入存在多个候选时，如果只有一个候选标记 `primary`，选择该 Bean。
3. 同一目标类型存在多个 `primary` 候选时，生成器或容器必须报错。

### profile

对齐 Spring `@Profile`，按 Environment active profiles 条件注册 Bean。

位置：

- `type` 上方。
- 带 `//goark:bean` 的方法上方。

参数：

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `value` | string | 无 | Profile 表达式 |

示例：

```go
//goark:configuration
//goark:profile("dev")
type DevConfiguration struct{}
```

Bean 方法示例：

```go
//goark:bean("dataSource")
//goark:profile("prod")
func (AppConfiguration) ProdDataSource() *sql.DB {
	return OpenProdDataSource()
}
```

Profile 表达式沿用 Spring 语义：

```go
//goark:profile("dev | test")
//goark:profile("!prod")
//goark:profile("prod & mysql")
```

规则：

1. `profile` 不匹配时，目标 BeanDefinition 不注册。
2. `configuration` 上的 `profile` 不匹配时，该配置单元及其 `bean` 方法整体跳过。
3. 多个 `profile` 注解作用到同一目标时按 OR 处理；单个表达式内部按 Spring profile expression 解析。
4. 未指定 active profile 时，Environment 使用 `default` profile。

### conditional

对齐 Spring `@Conditional`，按 Condition 结果决定是否注册 Bean。

位置：

- `type` 上方。
- 带 `//goark:bean` 的方法上方。

参数：

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `value` | string | 无 | Condition Go 类型表达式 |

示例：

```go
//goark:configuration
//goark:conditional("FeatureEnabledCondition")
type FeatureConfiguration struct{}
```

Bean 方法示例：

```go
//goark:bean("auditService")
//goark:conditional("AuditEnabledCondition")
func (AppConfiguration) AuditService() *AuditService {
	return NewAuditService()
}
```

Goark 核心库需要提供等价于 Spring `Condition` 的接口：

```go
type Condition interface {
	Matches(ctx ConfigurationContext, metadata AnnotationMetadata) (bool, error)
}
```

`conditional` 的规则：

1. `value` 指向的 Go 类型必须实现 `goark.Condition`。
2. Condition 类型必须是可零值构造的命名 struct；如果值类型实现接口，生成器使用 `Type{}`；如果指针类型实现接口，生成器使用 `&Type{}`。
3. 多个 `conditional` 注解作用到同一目标时按 AND 处理，全部匹配才注册。
4. `profile` 等价于内置 Condition，和 `conditional` 共同生效。

### lazy

对齐 Spring `@Lazy`，声明 Bean 延迟初始化。

位置：

- `type` 上方。
- 带 `//goark:bean` 的方法上方。

参数：

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `value` | bool | `true` | 是否延迟初始化 |

示例：

```go
//goark:service
//goark:lazy
type ReportService struct{}
```

等价写法：

```go
//goark:lazy(true)
```

关闭延迟初始化：

```go
//goark:lazy(false)
```

V1 中 `lazy` 只支持 BeanDefinition 级别的 lazy-init：容器 refresh 时不主动创建该 Bean，第一次解析依赖或主动获取 Bean 时创建。Spring 的注入点 lazy proxy 依赖运行时代理，Goark 核心库 V1 不做透明代理。

### depends-on

对齐 Spring `@DependsOn`，声明当前 Bean 初始化前必须先初始化的 Bean 名称。

位置：

- `type` 上方。
- 带 `//goark:bean` 的方法上方。

参数：

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `value` | string | 无 | Bean 名称；多个名称可用英文逗号分隔，生成器也必须兼容多个字符串参数 |

示例：

```go
//goark:service
//goark:depends-on("database")
type ApiResourceService struct{}
```

多个依赖：

```go
//goark:depends-on("database,cacheManager")
//goark:depends-on("schemaMigrator", "redisClient")
```

规则：

1. 普通字段注入、方法参数注入、`resource`、`qualifier`、`named` 的依赖关系由生成器自动推导，不需要额外写 `depends-on`。
2. `depends-on` 表示手工初始化和生命周期顺序依赖：当前 Bean 创建前必须先创建 `depends-on` 指定的 Bean。
3. ApplicationContext 关闭时，当前 Bean 必须先于 `depends-on` 指定的依赖 Bean 销毁；自动推导依赖同样参与生命周期顺序。
4. 指定不存在的 Bean 名称必须报错。
5. `depends-on` 形成的循环依赖必须报错，即使启用了循环引用开关也不能放行。

### scope

对齐 Spring `@Scope`，声明 Bean 作用域。

位置：

- `type` 上方。
- 带 `//goark:bean` 的方法上方。

参数：

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `value` | string | `singleton` | Scope 名称 |

示例：

```go
//goark:service
//goark:scope("prototype")
type JobHandler struct{}
```

V1 内置 Scope：

| Scope | 语义 |
| --- | --- |
| `singleton` | 每个 ApplicationContext 一个实例，默认值 |
| `prototype` | 每次请求创建新实例，不进入单例缓存 |

自定义 Scope 属于 `goark` 核心容器扩展点：用户可以向容器注册 Scope 实现。未注册的 Scope 名称必须报错。Web request/session scope 不属于核心库默认 Scope。

### property-source

对齐 Spring `@PropertySource`，为 Configuration 加载属性资源到 Environment。

位置：带 `//goark:configuration` 的类型上方。

参数：

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `value` | string | 无 | Resource 路径 |
| `name` | string | 资源路径 | PropertySource 名称 |
| `encoding` | string | `utf-8` | 文本编码 |
| `ignoreResourceNotFound` | bool | `false` | 资源不存在时是否忽略 |

示例：

```go
//goark:configuration
//goark:property-source("classpath:app.yml")
type AppConfiguration struct{}
```

文件系统资源：

```go
//goark:property-source(value="file:config/admin.properties", name="admin")
```

规则：

1. 资源路径解析使用 `core/resource.ResourceLoader`。
2. 加载结果写入 `core/env.Environment` 的 PropertySource 链。
3. 后加载的 PropertySource 优先级高于先加载的同级资源。
4. `ignoreResourceNotFound=false` 时资源不存在必须报错。
5. 未携带扩展名的资源路径按 `yml`、`properties`、`toml`、`yaml` 顺序查找。

V1 默认支持 YAML、Java `.properties`、TOML 三类配置文件。默认配置基础名称是 `app`，即 `app.yml`、`app.properties`、`app.toml`、`app.yaml`。格式解析由 Koanf 驱动；`.yml` 优先，`.properties` 次之，`.toml` 再次，`.yaml` 作为 YAML 兼容兜底。

### property-sources

对齐 Spring `@PropertySources`，声明多个 PropertySource。

位置：带 `//goark:configuration` 的类型上方。

参数：

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `value` | string | 无 | 多个资源路径，使用英文分号分隔 |
| `encoding` | string | `utf-8` | 文本编码 |
| `ignoreResourceNotFound` | bool | `false` | 资源不存在时是否忽略 |

示例：

```go
//goark:configuration
//goark:property-sources("classpath:app.properties;classpath:admin.properties")
type AppConfiguration struct{}
```

推荐写法仍然是重复使用 `property-source`，更接近 Go 注释的可读性：

```go
//goark:configuration
//goark:property-source("classpath:app.properties")
//goark:property-source("classpath:admin.properties")
type AppConfiguration struct{}
```

重复 `property-source` 与 `property-sources` 的语义一致。

### autowired

对齐 Spring `@Autowired`，声明依赖注入。

位置：

- 结构体字段上方。
- 方法上方带参数选择器，用于方法参数注入。

参数：

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `required` | bool | `true` | 是否必须存在依赖 |
| `qualifier` | string | 空 | 指定 Bean 名 |

示例：

```go
//goark:service
type ApiResourceService struct {
	//goark:autowired
	repository *ApiResourceRepository
}
```

可选依赖：

```go
//goark:service
type ApiResourceService struct {
	//goark:autowired(required=false)
	cache Cache
}
```

`required=false` 只表示依赖不存在时不报错；如果出现多个候选 Bean 或候选 Bean 类型不匹配，仍然必须报错。

指定 Bean 名：

```go
//goark:service
type ApiResourceService struct {
	//goark:autowired(qualifier="apiResourceRepository")
	repository *ApiResourceRepository
}
```

等价拆分：

```go
//goark:service
type ApiResourceService struct {
	//goark:autowired
	//goark:qualifier("apiResourceRepository")
	repository *ApiResourceRepository
}
```

方法参数示例：

```go
//goark:bean("userService")
//goark:autowired[repository]
//goark:qualifier[repository]("apiResourceRepository")
func (AdminConfiguration) UserService(repository *ApiResourceRepository) *ApiResourceService {
	return NewApiResourceService(repository)
}
```

方法参数可选依赖：

```go
//goark:bean("userService")
//goark:autowired[cache](required=false)
func (AdminConfiguration) UserService(repository *ApiResourceRepository, cache Cache) *ApiResourceService {
	return NewApiResourceService(repository, cache)
}
```

方法参数使用 `required=false` 时，参数类型必须是指针、接口、slice、map、chan 或 func。依赖不存在时，生成代码传入 `nil`。非 nil-able 类型不能声明为可选依赖。

### qualifier

对齐 Spring `@Qualifier`，指定依赖使用的 Bean 名。

位置：

- 字段上方，用于字段注入。
- 方法上方带参数选择器，用于方法参数注入。

字段示例：

```go
//goark:autowired
//goark:qualifier("masterDataSource")
db *sql.DB
```

方法参数示例：

```go
//goark:bean("userService")
//goark:qualifier[userDao]("userDao")
func (AdminConfiguration) UserService(userDao UserDao) *UserService {
	return &UserService{userDao: userDao}
}
```

### value

对齐 Spring `@Value`，从 Environment 注入属性占位符、GaEL 表达式或字面量。

位置：

- 结构体字段上方。
- 方法上方带参数选择器，用于方法参数注入。

参数：

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `value` | string | 无 | 属性占位符、GaEL 表达式或字面量 |

字段示例：

```go
//goark:component
type ServerProperties struct {
	//goark:value("${server.port:8080}")
	port int
}
```

方法参数示例：

```go
//goark:bean("httpServer")
//goark:value[port]("${server.port:8080}")
func (ServerConfiguration) HTTPServer(port int) *HTTPServer {
	return NewHTTPServer(port)
}
```

支持的表达形式：

```go
//goark:value("${app.name}")
//goark:value("${server.port:8080}")
//goark:value("#{environment['feature.enabled'] == 'true'}")
//goark:value("literal-value")
```

规则：

1. `${key}` 由 `core/env.PropertyResolver` 解析。
2. `${key:default}` 在属性不存在时使用默认值。
3. 解析后的字符串由 `core/convert.ConversionService` 转成字段或参数的 Go 类型。
4. 字面量直接进入类型转换流程。
5. `#{...}` 表示完整 GaEL 表达式，不允许和表达式外的字面量混写。
6. GaEL 支持字面量、算术、比较、逻辑、括号、变量、索引、属性读取和显式注册函数。
7. GaEL 不允许任意方法调用，也不提供文件、网络、进程或系统命令访问能力。

### inject

对齐 JSR-330 `@Inject`，声明必需依赖注入。

位置：

- 结构体字段上方。
- 方法上方带参数选择器，用于方法参数注入。

参数：无。

`inject` 默认按类型解析，找不到依赖必须报错。它没有 `required` 参数，`inject(required=false)` 是非法写法。如果需要可选依赖，使用 `autowired(required=false)`。

字段示例：

```go
//goark:service
type ApiResourceService struct {
	//goark:inject
	repository *ApiResourceRepository
}
```

方法参数示例：

```go
//goark:bean("userService")
//goark:inject[repository]
func (AdminConfiguration) UserService(repository *ApiResourceRepository) *ApiResourceService {
	return NewApiResourceService(repository)
}
```

如果同类型存在多个 Bean，使用 `named` 或 `qualifier` 指定名称：

```go
//goark:service
type ApiResourceService struct {
	//goark:inject
	//goark:named("apiResourceRepository")
	repository *ApiResourceRepository
}
```

参数场景：

```go
//goark:bean("userService")
//goark:inject[repository]
//goark:named[repository]("apiResourceRepository")
func (AdminConfiguration) UserService(repository *ApiResourceRepository) *ApiResourceService {
	return NewApiResourceService(repository)
}
```

### named

对齐 JSR-330 `@Named`，为 `inject` 指定 Bean 名。

位置：

- 字段上方，用于字段注入。
- 方法上方带参数选择器，用于方法参数注入。

字段示例：

```go
//goark:inject
//goark:named("masterDataSource")
db *sql.DB
```

方法参数示例：

```go
//goark:bean("database")
//goark:inject[dataSource]
//goark:named[dataSource]("masterDataSource")
func (AdminConfiguration) Database(dataSource *sql.DB) *Repository {
	return NewRepository(dataSource)
}
```

V1 中 `named` 与 `qualifier` 的解析行为一致，只保留标准来源差异：

```text
named    -> JSR-330 @Named
qualifier -> Spring @Qualifier
```

`named` 本身不是 Bean 注册注解，也不会单独触发字段注入。注册 Bean 仍然使用 `component`、`service`、`repository`、`configuration` 或 `bean`；字段注入仍然需要 `autowired`、`inject` 或 `resource`。

同一个注入目标上不能同时出现不同值的 `named` 和 `qualifier`。

### resource

对齐 JSR-250 `@Resource`，声明资源注入。

位置：

- 结构体字段上方。
- 方法上方带参数选择器，用于方法参数注入。

`resource` 默认按名称解析，名称缺省时使用字段名或参数名。如果按名称找不到，再退化为按目标类型解析。

字段示例：

```go
//goark:service
type ApiResourceService struct {
	//goark:resource
	repository *ApiResourceRepository
}
```

上面会先按字段名查找 Bean：

```text
repository
```

找不到时，再按 `*ApiResourceRepository` 类型查找。

显式名称：

```go
//goark:resource("apiResourceRepository")
repository *ApiResourceRepository
```

等价于：

```go
//goark:resource(name="apiResourceRepository")
repository *ApiResourceRepository
```

方法参数示例：

```go
//goark:bean("userService")
//goark:resource[repository]("apiResourceRepository")
func (AdminConfiguration) UserService(repository *ApiResourceRepository) *ApiResourceService {
	return NewApiResourceService(repository)
}
```

`resource` 支持的参数：

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `name` / `value` | string | 字段名或参数名 | Bean 名称 |
| `type` | string | 目标 Go 类型 | 限定目标类型 |

匹配规则：

1. 同时指定 `name` 和 `type`：按名称查找，并要求 Bean 类型可赋值给指定类型与目标类型；找不到名称直接报错。
2. 只指定 `name`：按名称查找，并要求 Bean 类型可赋值给目标类型；找不到名称直接报错。
3. 只指定 `type`：按类型查找唯一 Bean，并要求 Bean 类型可赋值给目标类型。
4. 未指定 `name` 和 `type`：先按字段名或参数名查找，找不到再按目标类型查找。

Goark 中目标字段或参数已经有静态类型，因此 `type` 参数通常不需要写。V1 可以解析 `type` 但不建议常规业务代码使用。

`type` 参数使用 Go 类型表达式字符串，例如：

```go
//goark:resource(type="*PostgresUserDao")
dao UserDao
```

生成器必须基于 Go 类型检查结果解析该类型。无法解析类型、类型不唯一或不可赋值给目标字段/参数类型时，必须报错。

## 默认命名规则

默认 Bean 名使用类型名或方法名的 lowerCamel 形式：

```text
ApiResourceService -> apiResourceService
Database           -> database
```

显式名称优先：

```go
//goark:service("apiResourceService")
type ApiResourceServiceImpl struct{}
```

## 注入解析规则

V1 将注入注解分成两类：

- Bean 依赖注入注解：`autowired`、`inject`、`resource`。
- 属性值注入注解：`value`。
- 限定名注解：`qualifier`、`named`。

同一个字段或方法参数最多只能有一个注入来源：`autowired`、`inject`、`resource`、`value` 四者互斥。`qualifier` 和 `named` 可以限定 `autowired` 或 `inject` 的候选 Bean；`resource` 自带名称和类型语义，不能再叠加 `qualifier` 或 `named`；`value` 注入属性值，不能叠加任何 Bean 限定名注解。

字段注入：

```go
//goark:autowired
repository *ApiResourceRepository
```

`autowired` 默认按字段类型解析。若容器中存在多个同类型 Bean，必须使用 `qualifier` 或 `named` 指定名称。

```go
//goark:inject
//goark:named("apiResourceRepository")
repository *ApiResourceRepository
```

`inject` 也按字段类型解析，但始终是必需依赖，不支持 `required=false`。

```go
//goark:resource
repository *ApiResourceRepository
```

`resource` 先按字段名 `repository` 查找 Bean；找不到时按字段类型查找。

```go
//goark:value("${server.port:8080}")
port int
```

`value` 通过 Environment 解析属性，再通过 ConversionService 转换成目标类型。

方法参数注入：

```go
//goark:bean("userService")
func (AdminConfiguration) UserService(repository *ApiResourceRepository) *ApiResourceService
```

Bean 方法参数默认按参数类型解析，并且默认是必需依赖。若存在多个同类型 Bean，必须使用参数选择器：

```go
//goark:qualifier[repository]("apiResourceRepository")
```

也可以使用 JSR-330 风格：

```go
//goark:named[repository]("apiResourceRepository")
```

或者使用 JSR-250 风格：

```go
//goark:resource[repository]("apiResourceRepository")
```

属性值参数使用 `value`：

```go
//goark:value[port]("${server.port:8080}")
```

`autowired(required=false)` 对字段和方法参数都合法。字段依赖不存在时保留字段零值；方法参数依赖不存在时传入 `nil`，因此方法参数类型必须是 nil-able 类型。`required=false` 不屏蔽多候选 Bean、类型不匹配、限定名冲突等错误。

## 依赖图推导与循环依赖

生成器必须用 Go 类型检查结果构建完整依赖图，不能依赖运行时反射扫描。依赖边来源分两类：

- 自动推导依赖：字段注入、setter 注入、Bean 方法参数、`autowired`、`inject`、`resource`、`qualifier`、`named`。
- 手工依赖：用户显式声明的 `goark:depends-on`。

依赖边语义分三类：

- `factory`：Bean 方法参数、构造函数参数或 provider 内必须先解析的依赖。该类依赖不能通过 early singleton 解决循环。
- `injection`：字段或 setter 注入依赖。单例 Bean 在启用循环引用开关时可以通过 early singleton 解决循环。
- `depends-on`：手工初始化顺序依赖。该类依赖控制创建和生命周期顺序，不执行注入，循环必须报错。

生成器解析依赖候选时必须保持 Spring 顺序：

1. `qualifier`、`named`、`resource(name=...)` 等显式名称优先。
2. 按类型存在多个候选时，先选唯一 `primary`。
3. 没有 `primary` 时按最高 `priority` 选择唯一候选。
4. 仍不唯一时报多候选错误。
5. `required=false` 且依赖不存在时不生成硬依赖边；依赖存在时仍参与依赖图校验。

循环依赖规则：

1. 默认不允许循环依赖。
2. `goark.WithAllowCircularReferences(true)` 或 boot 配置 `goark.main.allow-circular-references=true` 打开后，只允许单例 Bean 的字段或 setter 注入循环。
3. Bean 方法参数、构造参数、provider 内即时解析依赖、prototype Bean、`depends-on` 初始化顺序依赖形成的循环必须报错。
4. 允许的单例字段注入循环由容器按 Spring 风格分两阶段处理：先实例化并暴露 early singleton，再执行生成器写入的依赖注入函数。
5. early singleton 只允许同一解析链内的循环注入使用；普通并发 `Get` 必须等待 Bean 注入完成后返回。

## 生成规则

CLI 扫描源码后生成普通 Go 文件，文件名建议使用：

```text
zz_goark_<package>_gen.go
```

每个 package 内生成本 package 的 provider，因此可以为私有字段赋值。

生成内容包括：

- Component、Service、Repository 的 provider。
- Configuration 的 `Name`、`Order`、`Register` 方法。
- Bean 方法对应的 provider。
- BeanDefinition 元数据：`primary`、`lazy`、`depends-on`、`scope`、`order`、`priority`、依赖描述符。
- Environment 装配代码：`property-source`、`property-sources`。
- 条件装配判断代码：`profile`、`conditional`。
- 字段和方法参数的 Bean 依赖注入代码：`autowired`、`inject`、`resource`、`qualifier`、`named`。
- 字段和方法参数的属性值注入代码：`value`。
- 应用级 `RegisterConfigurations(app *goark.ApplicationContext) error`。

生成代码必须：

- 使用确定性排序。
- 用图算法合并自动推导依赖和手工 `depends-on` 依赖，并去重。
- 所有依赖描述都必须参与单例启动和生命周期关闭顺序计算。
- 字段或 setter 注入依赖生成 `WithInjectionDependencies` 和 `WithDependencyInjector`；Bean 方法参数或构造参数依赖生成 `WithFactoryDependencies`；手工 `depends-on` 生成 `WithDependsOn`。
- 不依赖运行时扫描。
- 不修改用户源码。
- 可被 `go test ./...` 直接编译。

## 命令模型

V1 注解的扫描和生成流程由 `goark` CLI 负责；本仓库只规定生成代码对 `goark` 核心库的调用契约，不在核心库中实现命令行入口。

```bash
goark generate
goark run ./cmd/admin
goark build ./cmd/admin
goark test ./...
```

`goark run`、`goark build`、`goark test` 必须先执行扫描和生成，再调用 Go 原生命令。

原生命令仍然可用，但只编译已有生成文件：

```bash
go run ./cmd/admin
go build ./cmd/admin
go test ./...
```

如果生成文件不存在或过期，原生命令不会自动重新生成。

## 完整示例

```go
package admin

import "database/sql"

//goark:repository
//goark:primary
type ApiResourceRepository struct {
	//goark:autowired
	db *sql.DB
}

//goark:service("apiResourceService")
type ApiResourceService struct {
	//goark:autowired
	//goark:qualifier("apiResourceRepository")
	repository *ApiResourceRepository

	//goark:value("${admin.api-resource.cache-enabled:false}")
	cacheEnabled bool
}

func NewApiResourceService(repository *ApiResourceRepository, cacheEnabled bool) *ApiResourceService {
	return &ApiResourceService{repository: repository, cacheEnabled: cacheEnabled}
}
```

```go
package config

import (
	"database/sql"

	"example.com/goark-example/internal/admin"
	_ "github.com/lib/pq"
)

//goark:configuration("admin")
//goark:order(0)
//goark:profile("dev | prod")
//goark:property-source("classpath:app.properties")
type AdminConfiguration struct{}

//goark:bean("database")
//goark:scope("singleton")
func (AdminConfiguration) Database() (*sql.DB, error) {
	return sql.Open("postgres", "postgres://example")
}

//goark:bean("apiResourceService")
//goark:qualifier[repository]("apiResourceRepository")
//goark:value[cacheEnabled]("${admin.api-resource.cache-enabled:false}")
func (AdminConfiguration) ApiResourceService(repository *admin.ApiResourceRepository, cacheEnabled bool) *admin.ApiResourceService {
	return admin.NewApiResourceService(repository, cacheEnabled)
}
```

## 错误规范

生成器必须在编译前报出明确错误。

重复 Bean 名：

```text
goark: duplicate bean name "apiResourceService"
```

参数选择器不存在：

```text
goark: qualifier targets missing parameter "repository" on AdminConfiguration.ApiResourceService
```

注解位置错误：

```text
goark: annotation "service" can only be used on type declarations
```

`[]` 用在非函数参数场景：

```text
goark: selector is only allowed for function or method parameters
```

Bean 方法返回值不支持：

```text
goark: bean method AdminConfiguration.Database must return T or (T, error)
```

排序来源冲突：

```text
goark: type ClientAuthInterceptor has both goark:order and user-defined Order method
```

`inject` 不允许 `required` 参数：

```text
goark: annotation "inject" does not support parameter "required"
```

方法参数使用 `required=false` 但类型不可传 `nil`：

```text
goark: autowired(required=false) parameter "retryCount" must be a nil-able type
```

同一注入目标存在多个主注入注解：

```text
goark: injection target "repository" has multiple injection annotations: autowired, resource
```

`named` 与 `qualifier` 值冲突：

```text
goark: injection target "repository" has conflicting named and qualifier values
```

`resource` 不能叠加 `qualifier` 或 `named`：

```text
goark: resource injection target "repository" cannot use qualifier or named
```

`resource` 类型无法解析：

```text
goark: resource type "*PostgresUserDao" cannot be resolved on AdminConfiguration.UserService parameter "dao"
```

`value` 占位符无法解析且没有默认值：

```text
goark: property "server.port" required by value injection is not defined
```

`value` 类型转换失败：

```text
goark: cannot convert property "server.port" value "abc" to int
```

多个 `primary` 候选冲突：

```text
goark: multiple primary beans found for type UserDao: postgresUserDao, mysqlUserDao
```

Profile 表达式非法：

```text
goark: invalid profile expression "dev | | test"
```

Condition 类型不满足接口：

```text
goark: conditional type "FeatureEnabledCondition" must implement goark.Condition
```

`depends-on` 指向不存在的 Bean：

```text
goark: bean "apiResourceService" depends on missing bean "database"
```

Scope 未注册：

```text
goark: scope "request" is not registered
```

PropertySource 资源不存在：

```text
goark: property source "classpath:missing.properties" not found
```

## Web MVC 注解

Web MVC 注解属于 `goark/web` 与 `goark/web/mvc` 的框架能力，对标 Spring Framework 的 `spring-web` 与 `spring-webmvc`，不属于 `boot` 或 starter。生成代码通过 `goweb.Configurer` 把 MVC 控制器贡献给 Web 注册表，再由 `gbc-web` 这类 starter 负责自动装配和启动容器。

控制器注解：

| Goark 注解 | Java/Spring 对照 | 目标 | V1 状态 |
| --- | --- | --- | --- |
| `//goark:controller` | `@Controller` | struct type | 实现 |
| `//goark:rest-controller` | `@RestController` | struct type | 实现 |
| `//goark:mvc-controller` | MVC 控制器构造型注解 | struct type | 实现 |
| `//goark:controller-advice` | `@ControllerAdvice` | struct type | 实现 |
| `//goark:rest-controller-advice` | `@RestControllerAdvice` | struct type | 实现 |
| `//goark:request-mapping` | `@RequestMapping` | struct type / method | 实现 |

HTTP 方法映射：

| Goark 注解 | Java/Spring 对照 | 默认状态码 |
| --- | --- | --- |
| `//goark:get` | `@GetMapping` | `200` |
| `//goark:head` | `@RequestMapping(method = HEAD)` | `200` |
| `//goark:post` | `@PostMapping` | `201` |
| `//goark:put` | `@PutMapping` | `200` |
| `//goark:patch` | `@PatchMapping` | `200` |
| `//goark:delete` | `@DeleteMapping` | `200` |
| `//goark:options` | `@RequestMapping(method = OPTIONS)` | `200` |

`request-mapping` 在方法上必须显式指定 `method`，支持 `GET`、`HEAD`、`POST`、`PUT`、`PATCH`、`DELETE`、`OPTIONS`：

```go
//goark:request-mapping("/healthz", method="OPTIONS")
func (c *SystemController) OptionsHealth() {}
```

参数绑定注解：

| Goark 注解 | Java/Spring 对照 | 说明 |
| --- | --- | --- |
| `//goark:request-body[input]` | `@RequestBody` | JSON 请求体绑定并校验 |
| `//goark:model-attribute[criteria]` | `@ModelAttribute` | query/form 聚合绑定并校验 |
| `//goark:path-variable[id]` | `@PathVariable` | 路径变量 |
| `//goark:request-param[query]` | `@RequestParam` | query 与 urlencoded form 参数 |
| `//goark:request-header[requestID]` | `@RequestHeader` | 请求头 |
| `//goark:cookie-value[theme]` | `@CookieValue` | Cookie 值 |

响应状态注解：

| Goark 注解 | Java/Spring 对照 | 说明 |
| --- | --- | --- |
| `//goark:response-status(204)` | `@ResponseStatus` | 覆盖普通返回值写出时使用的 HTTP 状态码 |

`response-status` 只用于 MVC 路由方法，并且不能和映射注解上的显式 `status` / `statusCode` 同时声明。默认状态码仍由 HTTP 方法决定：`POST` 默认为 `201`，其他方法默认为 `200`。

异常映射注解：

| Goark 注解 | Java/Spring 对照 | 说明 |
| --- | --- | --- |
| `//goark:exception-handler` | `@ExceptionHandler` | 标记 `controller-advice` 上的方法，使用 `errors.As` 按错误类型匹配 |

示例：

```go
//goark:rest-controller("adminController")
//goark:request-mapping("/admin")
type AdminController struct{}

//goark:get("/users/{id}")
//goark:path-variable[id]("id")
//goark:request-param[verbose](defaultValue="false")
func (c *AdminController) Detail(ctx *arkweb.Context, id int64, verbose bool) (User, error) {
	return c.service.Detail(ctx.Context(), id, verbose)
}
```

控制器方法支持的返回签名：

| 签名 | 生成行为 |
| --- | --- |
| `void` | 执行业务方法后返回 `204 No Content` |
| `error` | 无错误时返回 `204 No Content`，有错误时进入错误映射链 |
| `T` | 使用映射注解状态码写 JSON |
| `(T, error)` | 无错误时使用映射注解状态码写 JSON，有错误时进入错误映射链 |
| `arkweb.Result` | 直接写 Arkarta Web Result |
| `(arkweb.Result, error)` | 无错误时直接写 Arkarta Web Result，有错误时进入错误映射链 |
| `goweb.ResponseEntity[T]` | 直接写 Goark 响应实体，状态码和响应头由实体决定 |
| `(goweb.ResponseEntity[T], error)` | 无错误时直接写 Goark 响应实体，有错误时进入错误映射链 |

普通 `T` / `(T, error)` 返回值使用映射注解默认状态码，或者使用 `status` / `statusCode` / `response-status` 指定的状态码：

```go
//goark:post("/jobs")
//goark:response-status(200)
func (c *JobController) Search(input JobSearchRequest) (JobSearchResponse, error) {
	return c.service.Search(input)
}
```

`goweb.ResponseEntity[T]` 对标 Spring `ResponseEntity<T>` 的 Go 化能力。实体响应默认使用 Arkarta JSON Codec 写出响应体，并按 `Accept: application/json` 做内容协商；如果返回 `goweb.NoBody(status)`，只写状态码和响应头。方法级 `status` / `response-status` 只作用于普通 `T` / `(T, error)` 返回值，`ResponseEntity` 的状态码以实体自身为准。

```go
//goark:get("/users/{id}")
//goark:path-variable[id]("id")
func (c *AdminController) Detail(ctx *arkweb.Context, id int64) (goweb.ResponseEntity[UserDetailResponse], error) {
	user, err := c.service.Detail(ctx.Context(), id)
	if err != nil {
		return goweb.ResponseEntity[UserDetailResponse]{}, err
	}
	return goweb.OK(UserDetailResponse{User: user}).
		WithHeader("X-Admin-User-ID", strconv.FormatInt(user.ID, 10)), nil
}
```

异常处理示例：

```go
//goark:controller-advice("adminAdvice")
type AdminAdvice struct{}

//goark:exception-handler
func (a *AdminAdvice) NotFound(ctx *arkweb.Context, err *UserNotFoundError) arkweb.Result {
	return arkweb.JSON(404, map[string]any{
		"error": map[string]any{
			"code": "USER_NOT_FOUND",
			"id":   err.ID,
		},
	})
}
```

异常处理方法必须返回 `arkarta/web.Result`，参数必须包含一个错误类型参数，可选包含一个 `*arkarta/web.Context` 参数。生成器会生成 `mvc.ExceptionHandlerAs[T]`，并通过 `goweb.Configurer` 把异常处理器贡献给 `goark/web.Registry` 的错误映射链。错误映射器按注册顺序匹配；没有匹配时回退到 Arkarta Web 默认错误响应。

生成器会生成 `GoarkWebMVCConfiguration`，并注册 `goweb.Configurer` Bean。业务启动层仍需要显式把生成的配置单元交给 `goark.ApplicationContext` 或 `boot.Run`。

## Java/Spring 注解对照

| Goark 注解 | Java/Spring 对照 | 来源 | V1 状态 |
| --- | --- | --- | --- |
| `//goark:component` | `@Component` | Spring Framework | 实现 |
| `//goark:service` | `@Service` | Spring Framework | 实现 |
| `//goark:repository` | `@Repository` | Spring Framework | 实现 |
| `//goark:configuration` | `@Configuration` | Spring Framework | 实现 |
| `//goark:bean` | `@Bean` | Spring Framework | 实现 |
| `//goark:order` | `@Order` | Spring Framework | 实现 |
| `//goark:priority` | `@Priority` | JSR-250 / Jakarta Annotations | 实现 |
| `//goark:primary` | `@Primary` | Spring Framework | 实现 |
| `//goark:profile` | `@Profile` | Spring Framework | 实现 |
| `//goark:conditional` | `@Conditional` | Spring Framework | 实现 |
| `//goark:lazy` | `@Lazy` | Spring Framework | 实现 |
| `//goark:depends-on` | `@DependsOn` | Spring Framework | 实现 |
| `//goark:scope` | `@Scope` | Spring Framework | 实现 |
| `//goark:property-source` | `@PropertySource` | Spring Framework | 实现 |
| `//goark:property-sources` | `@PropertySources` | Spring Framework | 实现 |
| `//goark:autowired` | `@Autowired` | Spring Framework | 实现 |
| `//goark:qualifier` | `@Qualifier` | Spring Framework | 实现 |
| `//goark:value` | `@Value` | Spring Framework | 实现 |
| `//goark:inject` | `@Inject` | JSR-330 / Jakarta Inject | 实现 |
| `//goark:named` | `@Named` | JSR-330 / Jakarta Inject | 实现 |
| `//goark:resource` | `@Resource` | JSR-250 / Jakarta Annotations | 实现 |
| `//goark:controller` | `@Controller` | Spring Web MVC | 实现 |
| `//goark:rest-controller` | `@RestController` | Spring Web MVC | 实现 |
| `//goark:controller-advice` | `@ControllerAdvice` | Spring Web MVC | 实现 |
| `//goark:rest-controller-advice` | `@RestControllerAdvice` | Spring Web MVC | 实现 |
| `//goark:exception-handler` | `@ExceptionHandler` | Spring Web MVC | 实现 |
| `//goark:request-mapping` | `@RequestMapping` | Spring Web MVC | 实现 |
| `//goark:get` | `@GetMapping` | Spring Web MVC | 实现 |
| `//goark:head` | `@RequestMapping(method = HEAD)` | Spring Web MVC | 实现 |
| `//goark:post` | `@PostMapping` | Spring Web MVC | 实现 |
| `//goark:put` | `@PutMapping` | Spring Web MVC | 实现 |
| `//goark:patch` | `@PatchMapping` | Spring Web MVC | 实现 |
| `//goark:delete` | `@DeleteMapping` | Spring Web MVC | 实现 |
| `//goark:options` | `@RequestMapping(method = OPTIONS)` | Spring Web MVC | 实现 |
| `//goark:request-body` | `@RequestBody` | Spring Web MVC | 实现 |
| `//goark:model-attribute` | `@ModelAttribute` | Spring Web MVC | 实现 |
| `//goark:path-variable` | `@PathVariable` | Spring Web MVC | 实现 |
| `//goark:request-param` | `@RequestParam` | Spring Web MVC | 实现 |
| `//goark:request-header` | `@RequestHeader` | Spring Web MVC | 实现 |
| `//goark:cookie-value` | `@CookieValue` | Spring Web MVC | 实现 |
| `//goark:response-status` | `@ResponseStatus` | Spring Web MVC | 实现 |

## 核心库边界

本规范只要求 `goark`、`goark/web` 与 `goark/web/mvc` 提供这些注解对应的运行时契约、容器元数据和生成代码依赖面。`boot` 可以基于这些能力提供配置文件约定、自动注册和 starter 体验，但不能把新的框架语义反向塞进 `boot`。
