package commonrepo

import "time"

type Mode struct {
	ID        uint64    `gorm:"primarykey"`
	CreatedAt time.Time `gorm:"index;autoCreateTime"`
	UpdatedAt time.Time `gorm:"index;autoUpdateTime"`
}
