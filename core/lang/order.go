package lang

const (
	// HighestPrecedence 表示最高优先级。
	HighestPrecedence = -1 << 30
	// LowestPrecedence 表示最低优先级。
	LowestPrecedence = 1<<30 - 1
)

// Ordered 允许对象声明稳定排序权重，数值越小优先级越高。
type Ordered interface {
	Order() int
}

// PriorityOrdered 表示需要先于普通 Ordered 处理的高优先级对象。
type PriorityOrdered interface {
	Ordered
	PriorityOrdered()
}
