package context

// RefreshedEvent 表示应用上下文已完成 Bean 初始化。
type RefreshedEvent struct {
	Source *ApplicationContext
}

// StartedEvent 表示应用上下文生命周期已启动。
type StartedEvent struct {
	Source *ApplicationContext
}

// StoppedEvent 表示应用上下文生命周期已停止。
type StoppedEvent struct {
	Source *ApplicationContext
}

// ClosedEvent 表示应用上下文已关闭。
type ClosedEvent struct {
	Source *ApplicationContext
}
