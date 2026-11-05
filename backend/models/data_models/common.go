package data_models

import (
	"time"

	"gorm.io/gorm"
)

// 描述ORM模型使用的数据结构
type OrmModel struct {
	// 唯一标识
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`
	// 创建时间
	CreatedAt time.Time `json:"created_at"`
	// 更新时间
	UpdatedAt time.Time `json:"updated_at"`
	// 软删除时间
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}
