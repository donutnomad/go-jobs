package commonrepo

import "time"

type Mode struct {
	DefaultID uint64    `gorm:"primarykey"`
	ID        uint64    `gorm:"uniqueIndex"`
	CreatedAt time.Time `gorm:"index;autoCreateTime"`
	UpdatedAt time.Time `gorm:"index;autoUpdateTime"`
}
