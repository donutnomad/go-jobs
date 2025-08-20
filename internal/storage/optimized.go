package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jobs/scheduler/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Cache 简单的内存缓存
type Cache struct {
	mu      sync.RWMutex
	items   map[string]CacheItem
	ttl     time.Duration
	maxSize int
}

type CacheItem struct {
	Value      interface{}
	Expiration time.Time
}

func NewCache(ttl time.Duration, maxSize int) *Cache {
	cache := &Cache{
		items:   make(map[string]CacheItem),
		ttl:     ttl,
		maxSize: maxSize,
	}

	// 启动清理goroutine
	go cache.cleanup()

	return cache
}

func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 如果缓存满了，删除最旧的项
	if len(c.items) >= c.maxSize {
		var oldestKey string
		oldestTime := time.Now()
		for k, v := range c.items {
			if v.Expiration.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.Expiration
			}
		}
		delete(c.items, oldestKey)
	}

	c.items[key] = CacheItem{
		Value:      value,
		Expiration: time.Now().Add(c.ttl),
	}
}

func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(item.Expiration) {
		return nil, false
	}

	return item.Value, true
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

func (c *Cache) cleanup() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.Expiration) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// OptimizedStorage 优化后的存储层
type OptimizedStorage struct {
	*Storage
	cache         *Cache
	logger        *zap.Logger
	queryBatchers map[string]*QueryBatcher
}

// QueryBatcher 批量查询处理器
type QueryBatcher struct {
	mu         sync.Mutex
	queries    []func()
	interval   time.Duration
	maxBatch   int
	flushTimer *time.Timer
}

func NewQueryBatcher(interval time.Duration, maxBatch int) *QueryBatcher {
	return &QueryBatcher{
		queries:  make([]func(), 0, maxBatch),
		interval: interval,
		maxBatch: maxBatch,
	}
}

func (qb *QueryBatcher) Add(query func()) {
	qb.mu.Lock()
	defer qb.mu.Unlock()

	qb.queries = append(qb.queries, query)

	if len(qb.queries) >= qb.maxBatch {
		qb.flush()
		return
	}

	if qb.flushTimer == nil {
		qb.flushTimer = time.AfterFunc(qb.interval, func() {
			qb.mu.Lock()
			defer qb.mu.Unlock()
			qb.flush()
		})
	}
}

func (qb *QueryBatcher) flush() {
	if len(qb.queries) == 0 {
		return
	}

	// 执行批量查询
	queries := qb.queries
	qb.queries = make([]func(), 0, qb.maxBatch)

	if qb.flushTimer != nil {
		qb.flushTimer.Stop()
		qb.flushTimer = nil
	}

	// 并发执行查询
	var wg sync.WaitGroup
	for _, query := range queries {
		wg.Add(1)
		go func(q func()) {
			defer wg.Done()
			q()
		}(query)
	}
	wg.Wait()
}

// NewOptimizedStorage 创建优化的存储层
func NewOptimizedStorage(storage *Storage, logger *zap.Logger) *OptimizedStorage {
	return &OptimizedStorage{
		Storage:       storage,
		cache:         NewCache(5*time.Minute, 1000),
		logger:        logger,
		queryBatchers: make(map[string]*QueryBatcher),
	}
}

// GetTaskWithCache 带缓存的任务查询
func (os *OptimizedStorage) GetTaskWithCache(ctx context.Context, taskID string) (*models.Task, error) {
	cacheKey := fmt.Sprintf("task:%s", taskID)

	// 尝试从缓存获取
	if cached, ok := os.cache.Get(cacheKey); ok {
		if task, ok := cached.(*models.Task); ok {
			return task, nil
		}
	}

	// 从数据库查询
	var task models.Task
	if err := os.DB().WithContext(ctx).Where("id = ?", taskID).First(&task).Error; err != nil {
		return nil, err
	}

	// 存入缓存
	os.cache.Set(cacheKey, &task)

	return &task, nil
}

// GetExecutorsWithCache 带缓存的执行器查询
func (os *OptimizedStorage) GetExecutorsWithCache(ctx context.Context) ([]*models.Executor, error) {
	cacheKey := "executors:all"

	// 尝试从缓存获取
	if cached, ok := os.cache.Get(cacheKey); ok {
		if executors, ok := cached.([]*models.Executor); ok {
			return executors, nil
		}
	}

	// 从数据库查询
	var executors []*models.Executor
	if err := os.DB().WithContext(ctx).
		Where("status != ?", models.ExecutorStatusOffline).
		Find(&executors).Error; err != nil {
		return nil, err
	}

	// 存入缓存
	os.cache.Set(cacheKey, executors)

	return executors, nil
}

// InvalidateTaskCache 使任务缓存失效
func (os *OptimizedStorage) InvalidateTaskCache(taskID string) {
	os.cache.Delete(fmt.Sprintf("task:%s", taskID))
}

// InvalidateExecutorCache 使执行器缓存失效
func (os *OptimizedStorage) InvalidateExecutorCache() {
	os.cache.Delete("executors:all")
}

// GetRunningExecutionsCount 优化的运行中执行数量查询
func (os *OptimizedStorage) GetRunningExecutionsCount(ctx context.Context, executorID string) (int64, error) {
	cacheKey := fmt.Sprintf("running_count:%s", executorID)

	// 尝试从缓存获取
	if cached, ok := os.cache.Get(cacheKey); ok {
		if count, ok := cached.(int64); ok {
			return count, nil
		}
	}

	var count int64
	if err := os.DB().WithContext(ctx).
		Model(&models.TaskExecution{}).
		Where("executor_id = ? AND status = ?", executorID, models.ExecutionStatusRunning).
		Count(&count).Error; err != nil {
		return 0, err
	}

	// 存入缓存（较短的TTL）
	os.cache.Set(cacheKey, count)

	return count, nil
}

// BatchCreateExecutions 批量创建执行记录
func (os *OptimizedStorage) BatchCreateExecutions(ctx context.Context, executions []*models.TaskExecution) error {
	if len(executions) == 0 {
		return nil
	}

	// 使用批量插入优化
	return os.DB().WithContext(ctx).CreateInBatches(executions, 100).Error
}

// OptimizedQuery 优化的查询构建器
func (os *OptimizedStorage) OptimizedQuery(model interface{}) *gorm.DB {
	return os.DB().
		Model(model).
		Session(&gorm.Session{PrepareStmt: true}) // 使用预编译语句
}

// AddIndex 添加数据库索引（应在迁移中执行）
func (os *OptimizedStorage) AddIndexes() error {
	// 为常用查询添加索引
	indexes := []struct {
		Table   string
		Name    string
		Columns []string
	}{
		{"task_executions", "idx_task_executions_status", []string{"status"}},
		{"task_executions", "idx_task_executions_task_id", []string{"task_id"}},
		{"task_executions", "idx_task_executions_executor_id", []string{"executor_id"}},
		{"task_executions", "idx_task_executions_scheduled_time", []string{"scheduled_time"}},
		{"tasks", "idx_tasks_status", []string{"status"}},
		{"tasks", "idx_tasks_name", []string{"name"}},
		{"executors", "idx_executors_status", []string{"status"}},
		{"executors", "idx_executors_instance_id", []string{"instance_id"}},
	}

	for _, idx := range indexes {
		if err := os.DB().Table(idx.Table).
			Raw(fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)",
				idx.Name, idx.Table, idx.Columns[0])).Error; err != nil {
			os.logger.Warn("failed to create index",
				zap.String("index", idx.Name),
				zap.Error(err))
		}
	}

	return nil
}
