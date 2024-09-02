package Session

type MemoryPosition int

const (
	// Memo 内存实现
	Memo = iota
	// Io 磁盘实现
	Io
	// Redis redis实现
	Redis
	// Other 自定义实现
	Other
)
