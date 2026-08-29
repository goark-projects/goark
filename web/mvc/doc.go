// Package mvc 提供 Goark Web MVC 的显式路由装配、请求参数绑定、聚合表单绑定和异常映射能力。
//
// 第一版采用描述符和生成代码友好的注册模型，不做 Java 风格运行期扫描。
// RequestMapping 提供对齐 Spring MVC 的无 method 限定路由展开能力。
// Controller 级 method 和条件用于表达类型级 RequestMapping 的条件继承语义。
package mvc
