package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// ID 通用ID类型
type ID string

// NewID 创建新的ID
func NewID(s string) (ID, error) {
	if s == "" {
		return "", fmt.Errorf("ID cannot be empty")
	}
	return ID(s), nil
}

// String 返回字符串表示
func (id ID) String() string {
	return string(id)
}

// IsZero 判断是否为零值
func (id ID) IsZero() bool {
	return id == ""
}

// Status 通用状态枚举
type Status int

const (
	StatusActive Status = iota + 1
	StatusInactive
	StatusDeleted
)

func (s Status) String() string {
	switch s {
	case StatusActive:
		return "active"
	case StatusInactive:
		return "inactive"
	case StatusDeleted:
		return "deleted"
	default:
		return "unknown"
	}
}

// TimeRange 时间范围
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// IsValid 验证时间范围是否有效
func (tr TimeRange) IsValid() bool {
	return !tr.Start.IsZero() && !tr.End.IsZero() && tr.Start.Before(tr.End)
}

// Contains 检查给定时间是否在范围内
func (tr TimeRange) Contains(t time.Time) bool {
	return !t.Before(tr.Start) && t.Before(tr.End)
}

// Pagination 分页参数
type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// NewPagination 创建分页参数
func NewPagination(page, pageSize int) Pagination {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	return Pagination{
		Page:     page,
		PageSize: pageSize,
	}
}

// Offset 计算偏移量
func (p Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Limit 获取限制数量
func (p Pagination) Limit() int {
	return p.PageSize
}

// Money 金额类型
type Money struct {
	Amount   decimal.Decimal `json:"amount"`
	Currency string          `json:"currency"`
}

// NewMoney 创建金额
func NewMoney(amount string, currency string) (Money, error) {
	d, err := decimal.NewFromString(amount)
	if err != nil {
		return Money{}, fmt.Errorf("invalid amount: %w", err)
	}
	if currency == "" {
		currency = "CNY"
	}
	return Money{Amount: d, Currency: currency}, nil
}

// IsZero 判断是否为零
func (m Money) IsZero() bool {
	return m.Amount.IsZero()
}

// String 返回字符串表示
func (m Money) String() string {
	return fmt.Sprintf("%s %s", m.Amount.String(), m.Currency)
}

// JSONMap JSON映射类型，用于动态参数
type JSONMap map[string]interface{}

// Value 实现driver.Valuer接口
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan 实现sql.Scanner接口
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into JSONMap", value)
	}
	return json.Unmarshal(bytes, j)
}

// Get 获取值
func (j JSONMap) Get(key string) interface{} {
	if j == nil {
		return nil
	}
	return j[key]
}

// Set 设置值
func (j JSONMap) Set(key string, value interface{}) {
	if j == nil {
		j = make(map[string]interface{})
	}
	j[key] = value
}

// Has 检查是否存在键
func (j JSONMap) Has(key string) bool {
	if j == nil {
		return false
	}
	_, exists := j[key]
	return exists
}

// Filter 过滤器基础结构
type Filter struct {
	Pagination Pagination             `json:"pagination"`
	Conditions map[string]interface{} `json:"conditions"`
	OrderBy    string                 `json:"order_by"`
	OrderDir   string                 `json:"order_dir"`
}

// NewFilter 创建过滤器
func NewFilter() Filter {
	return Filter{
		Pagination: NewPagination(1, 20),
		Conditions: make(map[string]interface{}),
		OrderBy:    "created_at",
		OrderDir:   "DESC",
	}
}

// AddCondition 添加条件
func (f *Filter) AddCondition(key string, value interface{}) {
	if f.Conditions == nil {
		f.Conditions = make(map[string]interface{})
	}
	f.Conditions[key] = value
}

// GetCondition 获取条件
func (f Filter) GetCondition(key string) interface{} {
	if f.Conditions == nil {
		return nil
	}
	return f.Conditions[key]
}

// HasCondition 检查是否有条件
func (f Filter) HasCondition(key string) bool {
	if f.Conditions == nil {
		return false
	}
	_, exists := f.Conditions[key]
	return exists
}
