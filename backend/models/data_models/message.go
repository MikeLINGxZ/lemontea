package data_models

import (
	"encoding/json"
	"time"

	"github.com/cloudwego/eino/schema"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/utils"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/utils/ierror"
	"gorm.io/gorm"
)

type Message struct {
	OrmModel
	ChatUuid                     string                 `gorm:"index" json:"chat_uuid"`
	MessageUuid                  string                 `gorm:"unique;index" json:"message_uuid"`
	Role                         schema.RoleType        `gorm:"index" json:"role"`
	Content                      string                 `gorm:"type:text" json:"content"`
	ReasoningContent             string                 `gorm:"type:text" json:"reasoning_content"`
	UserMessageExtraContent      string                 `gorm:"type:text" json:"user_message_extra_content"`
	AssistantMessageExtraContent string                 `gorm:"type:text" json:"assistant_message_extra_content"`
	UserMessageExtra             *UserMessageExtra      `gorm:"-" json:"user_message_extra"`
	AssistantMessageExtra        *AssistantMessageExtra `gorm:"-" json:"assistant_message_extra"`
	SenderPersonUuid             string                 `gorm:"type:varchar(255);default:''" json:"sender_person_uuid"`
}

func (m *Message) ToSchemaMessage() (*schema.Message, error) {
	schemaMessage := &schema.Message{
		Role:             m.Role,
		Content:          m.Content,
		ReasoningContent: m.ReasoningContent,
	}
	// MultiContent
	if m.Role == schema.User && m.UserMessageExtra != nil && len(m.UserMessageExtra.Files) > 0 {
		schemaMessage.Content = ""
		schemaMessage.ReasoningContent = ""
		var userInputMultiContent []schema.MessageInputPart
		if m.Content != "" {
			userInputMultiContent = append(userInputMultiContent, schema.MessageInputPart{
				Type: schema.ChatMessagePartTypeText,
				Text: m.Content,
			})
		}

		for _, item := range m.UserMessageExtra.Files {
			// 通过mineType获取消息类型
			chatMessagePartType, err := utils.MimeType2ChatMessagePartType(item.MineType)
			if err != nil {
				return nil, ierror.NewError(err)
			}

			var text string
			var img *schema.MessageInputImage
			var audio *schema.MessageInputAudio
			var video *schema.MessageInputVideo
			var file *schema.MessageInputFile
			switch chatMessagePartType {
			case schema.ChatMessagePartTypeText:
				text, err = utils.ReadTextFileForChat(item.Path, item.Name, item.MineType)
				if err != nil {
					return nil, err
				}
			default:
				base64Data, err := utils.ReadFile2Base64Data(item.Path)
				if err != nil {
					return nil, err
				}
				messagePartCommon := schema.MessagePartCommon{
					Base64Data: &base64Data,
					MIMEType:   item.MineType,
					Extra: map[string]interface{}{
						"name":                   item.Name,
						"path":                   item.Path,
						"mime_type":              item.MineType,
						"chat_message_part_type": chatMessagePartType,
						"size":                   item.Size,
					},
				}
				switch chatMessagePartType {
				case schema.ChatMessagePartTypeFileURL:
					file = &schema.MessageInputFile{
						MessagePartCommon: messagePartCommon,
						Name:              item.Name,
					}
				case schema.ChatMessagePartTypeImageURL:
					img = &schema.MessageInputImage{
						MessagePartCommon: messagePartCommon,
						Detail:            schema.ImageURLDetailHigh,
					}
				case schema.ChatMessagePartTypeAudioURL:
					audio = &schema.MessageInputAudio{
						MessagePartCommon: messagePartCommon,
					}
				case schema.ChatMessagePartTypeVideoURL:
					video = &schema.MessageInputVideo{
						MessagePartCommon: messagePartCommon,
					}
				}
			}
			if img == nil && audio == nil && video == nil && file == nil {
				if text == "" {
					continue
				}
			}
			userInputMultiContent = append(userInputMultiContent, schema.MessageInputPart{
				Type:  chatMessagePartType,
				Text:  text,
				Image: img,
				Audio: audio,
				Video: video,
				File:  file,
			})
		}
		schemaMessage.UserInputMultiContent = userInputMultiContent
	}

	return schemaMessage, nil
}

type UserMessageExtra struct {
	ModelId   uint     `json:"model_id"`   // 模型id
	ModelName string   `json:"model_name"` // 模型名称
	Files     []File   `json:"files"`      // 文件路径
	Tools     []string `json:"tools"`      // 工具id
	Agents    []string `json:"agents"`     // agent id
}

type AssistantMessageExtra struct {
	ToolUses                []ToolUse             `json:"tool_uses"`
	ExecutionTrace          ExecutionTrace        `json:"execution_trace"`
	PendingApprovals        []ToolApprovalSummary `json:"pending_approvals"`
	SubAgentTasks           []SubAgentTask        `json:"sub_agent_tasks"`
	RouteType               RouteType             `json:"route_type"`
	RetryCount              int                   `json:"retry_count"`
	CurrentStage            string                `json:"current_stage"`
	CurrentAgent            string                `json:"current_agent"`
	PrefaceContent          string                `json:"preface_content"`
	PrefaceReasoningContent string                `json:"preface_reasoning_content"`
	FinishReason            string                `json:"finish_reason"`
	FinishError             string                `json:"finish_error"`
}

type ToolUseStatus string

const (
	ToolUseStatusPending          ToolUseStatus = "pending"
	ToolUseStatusRunning          ToolUseStatus = "running"
	ToolUseStatusAwaitingApproval ToolUseStatus = "awaiting_approval"
	ToolUseStatusDone             ToolUseStatus = "done"
	ToolUseStatusRejected         ToolUseStatus = "rejected"
	ToolUseStatusError            ToolUseStatus = "error"
)

type ToolUse struct {
	Index           int           `json:"index"`
	CallID          string        `json:"call_id"`
	ContentPos      int           `json:"content_pos"`
	ToolID          string        `json:"tool_id"`
	ToolName        string        `json:"tool_name"`
	ToolDescription string        `json:"tool_description"`
	ToolResult      string        `json:"tool_result"`
	Status          ToolUseStatus `json:"status"`
	StartedAt       *time.Time    `json:"started_at"`
	FinishedAt      *time.Time    `json:"finished_at"`
	ElapsedMs       int64         `json:"elapsed_ms"`
}

type SubAgentTask struct {
	TaskID       string        `json:"task_id"`
	AgentID      string        `json:"agent_id"`
	AgentName    string        `json:"agent_name"`
	Status       ToolUseStatus `json:"status"`
	Input        string        `json:"input"`
	Output       string        `json:"output"`
	ToolCalls    []ToolUse     `json:"tool_calls"`
	CreatorAgent string        `json:"creator_agent"`
	StartedAt    *time.Time    `json:"started_at"`
	FinishedAt   *time.Time    `json:"finished_at"`
	ElapsedMs    int64         `json:"elapsed_ms"`
}

type File struct {
	Name     string  `json:"name"`
	Path     string  `json:"path"`
	Preview  *string `json:"preview"` // 如果是图像的话，生成60x60的预览图
	MineType string  `json:"mine_type"`
	Size     int64   `json:"size"`
}

func (m *Message) BeforeCreate(tx *gorm.DB) (err error) {
	return m.before(tx)
}

func (m *Message) BeforeUpdate(tx *gorm.DB) (err error) {
	return m.before(tx)
}

func (m *Message) BeforeSave(tx *gorm.DB) (err error) {
	return m.before(tx)
}

func (m *Message) AfterFind(tx *gorm.DB) (err error) {

	if m.UserMessageExtraContent != "" {
		var userMessageExtra UserMessageExtra
		err = json.Unmarshal([]byte(m.UserMessageExtraContent), &userMessageExtra)
		if err != nil {
			return err
		}
		m.UserMessageExtra = &userMessageExtra
	}

	if m.AssistantMessageExtraContent != "" {
		var assistantMessageExtra AssistantMessageExtra
		err = json.Unmarshal([]byte(m.AssistantMessageExtraContent), &assistantMessageExtra)
		if err != nil {
			return err
		}
		m.AssistantMessageExtra = &assistantMessageExtra
	}

	return
}

func (m *Message) before(tx *gorm.DB) (err error) {
	if m.UserMessageExtra != nil {
		bytes, err := json.Marshal(m.UserMessageExtra)
		if err != nil {
			return err
		}
		m.UserMessageExtraContent = string(bytes)
	}
	if m.AssistantMessageExtra != nil {
		bytes, err := json.Marshal(m.AssistantMessageExtra)
		if err != nil {
			return err
		}
		m.AssistantMessageExtraContent = string(bytes)
	}
	return
}
