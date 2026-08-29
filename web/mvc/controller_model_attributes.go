package mvc

// WithModelAttributes 设置控制器级模型初始化器，对齐 Spring 方法级 @ModelAttribute。
func (c Controller) WithModelAttributes(initializers ...ModelAttributeInitializer) Controller {
	c.modelAttrs = append([]ModelAttributeInitializer(nil), initializers...)
	return c
}
