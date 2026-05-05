package data_models

type Chat struct {
	OrmModel
	Uuid         string `gorm:"unique;index" json:"uuid"`
	Title        string `gorm:"type:varchar(255)" json:"title"`
	Prompt       string `gorm:"type:text" json:"prompt"`
	IsCollection bool   `gorm:"index;default:false" json:"is_collection"`
	ChatType     string `gorm:"type:varchar(32);index;default:''" json:"chat_type"`
}

const (
	ChatTypeNormal    = ""           // 普通聊天
	ChatTypeOPCPerson = "opc_person" // OPC 人员私聊
	ChatTypeOPCGroup  = "opc_group"  // OPC 群聊
)
