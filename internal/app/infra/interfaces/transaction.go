package interfaces

import "context"

// TransactionManager 事务管理接口，遵循架构指导文档
type TransactionManager interface {
	// Execute 将一个函数包裹在事务中执行。
	// 如果函数返回错误，事务将回滚；否则将提交。
	Execute(ctx context.Context, fn func(ctx context.Context) error) error
}

// Locker 分布式锁接口
type Locker interface {
	// TryLock 尝试获取锁
	TryLock(ctx context.Context) (bool, error)

	// Lock 获取锁，会阻塞直到获取成功
	Lock(ctx context.Context) error

	// Unlock 释放锁
	Unlock(ctx context.Context) error

	// Renew 续约锁
	Renew(ctx context.Context) error

	// IsLocked 检查是否持有锁
	IsLocked() bool
}

// MessageQueue 消息队列接口
type MessageQueue interface {
	// Publish 发布消息
	Publish(ctx context.Context, topic string, message []byte) error

	// Subscribe 订阅消息
	Subscribe(ctx context.Context, topic string, handler MessageHandler) error

	// Close 关闭连接
	Close() error
}

// MessageHandler 消息处理器
type MessageHandler func(ctx context.Context, message []byte) error

// Cache 缓存接口
type Cache interface {
	// Get 获取缓存值
	Get(ctx context.Context, key string) ([]byte, error)

	// Set 设置缓存值
	Set(ctx context.Context, key string, value []byte, expiration int64) error

	// Delete 删除缓存值
	Delete(ctx context.Context, key string) error

	// Exists 检查键是否存在
	Exists(ctx context.Context, key string) (bool, error)
}

// HTTPClient HTTP客户端接口
type HTTPClient interface {
	// Get 发送GET请求
	Get(ctx context.Context, url string, headers map[string]string) (*HTTPResponse, error)

	// Post 发送POST请求
	Post(ctx context.Context, url string, body []byte, headers map[string]string) (*HTTPResponse, error)

	// Put 发送PUT请求
	Put(ctx context.Context, url string, body []byte, headers map[string]string) (*HTTPResponse, error)

	// Delete 发送DELETE请求
	Delete(ctx context.Context, url string, headers map[string]string) (*HTTPResponse, error)
}

// HTTPResponse HTTP响应
type HTTPResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
}
