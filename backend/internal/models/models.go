package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// DeletedAt is a custom type for swagger documentation
// swagger:model DeletedAt
type DeletedAt struct {
	Time  time.Time `json:"time"`
	Valid bool      `json:"valid"`
}

// Scan implements the sql.Scanner interface for DeletedAt
func (d *DeletedAt) Scan(value interface{}) error {
	if value == nil {
		d.Time, d.Valid = time.Time{}, false
		return nil
	}
	d.Valid = true
	switch v := value.(type) {
	case time.Time:
		d.Time = v
	case string:
		var err error
		d.Time, err = time.Parse(time.RFC3339, v)
		if err != nil {
			return err
		}
	default:
		d.Valid = false
	}
	return nil
}

// Value implements the driver.Valuer interface for DeletedAt
func (d DeletedAt) Value() (driver.Value, error) {
	if !d.Valid {
		return nil, nil
	}
	return d.Time, nil
}

// AuthType defines the authentication method for an email account.
type AuthType string

const (
	AuthTypePassword AuthType = "password"
	AuthTypeToken    AuthType = "token"
	AuthTypeOAuth2   AuthType = "oauth2"
)

// MailProviderType defines the type of email provider.
type MailProviderType string

const (
	ProviderTypeGmail   MailProviderType = "gmail"
	ProviderTypeOutlook MailProviderType = "outlook"
	ProviderTypeCustom  MailProviderType = "custom"
)

// AccountErrorStatus 账户错误状态
type AccountErrorStatus string

const (
	ErrorStatusNormal        AccountErrorStatus = "normal"         // 正常状态
	ErrorStatusOAuthExpired  AccountErrorStatus = "oauth_expired"  // OAuth Token过期
	ErrorStatusAuthRevoked   AccountErrorStatus = "auth_revoked"   // 授权被撤销
	ErrorStatusAPIDisabled   AccountErrorStatus = "api_disabled"   // API被禁用
	ErrorStatusNetworkError  AccountErrorStatus = "network_error"  // 网络错误
	ErrorStatusQuotaExceeded AccountErrorStatus = "quota_exceeded" // 配额超限
	ErrorStatusServerError   AccountErrorStatus = "server_error"   // 服务器错误
)

// MailProvider stores the configuration for a specific email provider.
type MailProvider struct {
	ID         uint             `gorm:"primaryKey" json:"id"`
	Name       string           `gorm:"unique;not null" json:"name"` // e.g., "Gmail", "Outlook"
	Type       MailProviderType `gorm:"not null" json:"type"`
	IMAPServer string           `gorm:"not null" json:"imapServer"`
	IMAPPort   int              `gorm:"not null" json:"imapPort"`
	SMTPServer string           `json:"smtpServer"`
	SMTPPort   int              `json:"smtpPort"`
	CreatedAt  time.Time        `json:"createdAt"`
	UpdatedAt  time.Time        `json:"updatedAt"`
	DeletedAt  DeletedAt        `gorm:"index" json:"deletedAt,omitempty"`
}

// StringSlice is a custom type for storing string arrays in database
type StringSlice []string

// Scan implements the sql.Scanner interface
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, s)
}

// Value implements the driver.Valuer interface
func (s StringSlice) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}
	return json.Marshal(s)
}

// JSONMap is a custom type for storing map[string]string in database
type JSONMap map[string]string

// Scan implements the sql.Scanner interface
func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = make(map[string]string)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		// Try string type as well
		if str, isString := value.(string); isString {
			bytes = []byte(str)
		} else {
			return fmt.Errorf("cannot convert value to bytes, got type %T", value)
		}
	}

	if len(bytes) == 0 {
		*m = make(map[string]string)
		return nil
	}

	if err := json.Unmarshal(bytes, m); err != nil {
		return err
	}
	return nil
}

// Value implements the driver.Valuer interface
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	return json.Marshal(m)
}

// EmailAccount represents a user's email account credentials and settings.
type EmailAccount struct {
	ID               uint                `gorm:"primaryKey" json:"id"`
	EmailAddress     string              `gorm:"uniqueIndex;not null;type:varchar(255)" json:"emailAddress"`
	AuthType         AuthType            `gorm:"not null;default:'password'" json:"authType"`
	Password         string              `json:"password,omitempty"`                                                                                         // For AuthTypePassword
	Token            string              `json:"token,omitempty"`                                                                                            // For AuthTypeToken
	MailProviderID   *uint               `gorm:"index" json:"mailProviderId,omitempty"`                                                                      // Make optional - only for accounts that need predefined providers
	MailProvider     *MailProvider       `gorm:"foreignKey:MailProviderID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"mailProvider,omitempty"`     // Make optional
	OAuth2ProviderID *uint               `gorm:"index" json:"oauth2ProviderId,omitempty"`                                                                    // For OAuth2 authentication, references OAuth2GlobalConfig
	OAuth2Provider   *OAuth2GlobalConfig `gorm:"foreignKey:OAuth2ProviderID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"oauth2Provider,omitempty"` // OAuth2配置关联
	Proxy            string              `json:"proxy,omitempty"`                                                                                            // e.g., "socks5://user:pass@host:port"
	IsDomainMail     bool                `gorm:"default:false" json:"isDomainMail"`
	Domain           string              `gorm:"index" json:"domain,omitempty"` // For domain-specific email
	CustomSettings   JSONMap             `gorm:"type:json" json:"customSettings"`
	LastSyncAt       *time.Time          `json:"lastSyncAt,omitempty"`
	IsVerified       bool                `gorm:"default:false" json:"isVerified"`
	VerifiedAt       *time.Time          `json:"verifiedAt,omitempty"`

	// 错误状态管理字段
	ErrorStatus    string     `gorm:"default:'normal'" json:"errorStatus"` // normal, oauth_expired, auth_revoked, api_disabled, network_error
	ErrorMessage   string     `gorm:"type:text" json:"errorMessage"`       // 详细错误信息
	ErrorTimestamp *time.Time `json:"errorTimestamp,omitempty"`            // 最后错误发生时间
	ErrorCount     int        `gorm:"default:0" json:"errorCount"`         // 累计错误次数
	AutoDisabledAt *time.Time `json:"autoDisabledAt,omitempty"`            // 自动禁用时间

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	DeletedAt DeletedAt `gorm:"index" json:"deletedAt,omitempty"`
}

// Email represents a single email message.
type Email struct {
	ID             uint         `gorm:"primaryKey"`
	MessageID      string       `gorm:"index"` // RFC Message-ID
	AccountID      uint         `gorm:"not null"`
	Account        EmailAccount `gorm:"foreignKey:AccountID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Subject        string
	From           StringSlice `gorm:"type:json"`
	To             StringSlice `gorm:"type:json"`
	Cc             StringSlice `gorm:"type:json"`
	Bcc            StringSlice `gorm:"type:json"`
	Date           time.Time   `gorm:"index"`
	ReceivedAt     time.Time   `gorm:"index"` // 接收时间
	Body           string      `gorm:"type:text"`
	TextBody       string      `gorm:"type:text"` // 纯文本内容
	HTMLBody       string      `gorm:"type:text"`
	RawMessage     string      `gorm:"type:longtext"` // 存储原始邮件报文
	InReplyTo      string      // In-Reply-To header
	References     StringSlice `gorm:"type:json"` // References header
	Headers        JSONMap     `gorm:"type:json"` // 其他邮件头
	Attachments    []Attachment
	HasAttachments bool        // 是否有附件
	MailboxName    string      `gorm:"index"`     // IMAP mailbox name
	Flags          StringSlice `gorm:"type:json"` // IMAP flags
	Size           int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *DeletedAt `gorm:"index"`
}

// Attachment represents an email attachment.
type Attachment struct {
	ID          uint   `gorm:"primaryKey"`
	EmailID     uint   `gorm:"not null"`
	Email       Email  `gorm:"foreignKey:EmailID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Filename    string `gorm:"not null"`
	Content     []byte `gorm:"type:blob"`
	MIMEType    string
	ContentType string // 内容类型
	Size        int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Mailbox represents a mailbox on the IMAP server.
type Mailbox struct {
	ID        uint         `gorm:"primaryKey"`
	Name      string       `gorm:"not null"`
	AccountID uint         `gorm:"not null"`
	Account   EmailAccount `gorm:"foreignKey:AccountID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Delimiter string
	Flags     StringSlice `gorm:"type:json"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// IncrementalSyncRecord represents the incremental sync state for an account
type IncrementalSyncRecord struct {
	ID                uint         `gorm:"primaryKey" json:"id"`
	AccountID         uint         `gorm:"not null;uniqueIndex:idx_account_mailbox" json:"account_id"`
	Account           EmailAccount `gorm:"foreignKey:AccountID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"account"`
	MailboxName       string       `gorm:"not null;uniqueIndex:idx_account_mailbox;type:varchar(255)" json:"mailbox_name"`
	LastSyncEndTime   time.Time    `gorm:"not null" json:"last_sync_end_time"`
	LastSyncStartTime time.Time    `gorm:"not null" json:"last_sync_start_time"`
	EmailsProcessed   int          `gorm:"default:0" json:"emails_processed"`
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`
}

// ExtractorTemplateConfig represents a single extractor configuration within a template
type ExtractorTemplateConfig struct {
	Field   string  `json:"field"`           // Field to extract from: ALL, from, to, cc, subject, body, html_body, headers
	Type    string  `json:"type"`            // Type of extraction: regex, js, gotemplate
	Match   *string `json:"match,omitempty"` // Optional match configuration (returns {matched: boolean, reason?: string})
	Extract string  `json:"extract"`         // Extract configuration (returns string or null)
}

// ExtractorTemplateConfigs is a custom type for storing ExtractorTemplateConfig array in database
type ExtractorTemplateConfigs []ExtractorTemplateConfig

// Scan implements the sql.Scanner interface
func (e *ExtractorTemplateConfigs) Scan(value interface{}) error {
	if value == nil {
		*e = []ExtractorTemplateConfig{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, e)
}

// Value implements the driver.Valuer interface
func (e ExtractorTemplateConfigs) Value() (driver.Value, error) {
	if len(e) == 0 {
		return "[]", nil
	}
	return json.Marshal(e)
}

// ExtractorTemplate represents a saved email extraction template
type ExtractorTemplate struct {
	ID          uint                     `gorm:"primaryKey" json:"id"`
	Name        string                   `gorm:"not null;uniqueIndex;type:varchar(255)" json:"name"` // Custom name for the template
	Description string                   `json:"description,omitempty"`                              // Optional description
	Extractors  ExtractorTemplateConfigs `gorm:"type:json;not null" json:"extractors"`               // Array of extractor configurations
	CreatedAt   time.Time                `json:"createdAt"`
	UpdatedAt   time.Time                `json:"updatedAt"`
	DeletedAt   DeletedAt                `gorm:"index" json:"deletedAt,omitempty"`
}

// AIChannelType defines the type of AI provider
type AIChannelType string

const (
	AIChannelOpenAI AIChannelType = "openai"
	AIChannelGemini AIChannelType = "gemini"
	AIChannelClaude AIChannelType = "claude"
)

// OpenAIConfig represents the OpenAI configuration settings
type OpenAIConfig struct {
	ID          uint          `gorm:"primaryKey" json:"id"`
	Name        string        `gorm:"not null;uniqueIndex;type:varchar(255)" json:"name"` // Configuration name (e.g., "default", "production")
	ChannelType AIChannelType `gorm:"not null;default:'openai'" json:"channel_type"`      // AI provider type
	BaseURL     string        `gorm:"not null" json:"base_url"`                           // API base URL
	APIKey      string        `gorm:"not null" json:"api_key"`                            // Encrypted API key
	Model       string        `gorm:"not null" json:"model"`                              // Default model
	Headers     JSONMap       `gorm:"type:json" json:"headers"`                           // Additional headers
	IsActive    bool          `gorm:"default:false" json:"is_active"`                     // Whether this config is active
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	DeletedAt   DeletedAt     `gorm:"index" json:"deleted_at,omitempty"`
}

// AIPromptTemplate represents system prompts for different AI scenarios
type AIPromptTemplate struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Scenario     string    `gorm:"not null;uniqueIndex;type:varchar(255)" json:"scenario"` // Scenario identifier (e.g., "email_template_generation")
	Name         string    `gorm:"not null" json:"name"`                                   // Human-readable name
	Description  string    `json:"description,omitempty"`                                  // Description of the scenario
	SystemPrompt string    `gorm:"type:text;not null" json:"system_prompt"`                // System prompt template
	UserPrompt   string    `gorm:"type:text" json:"user_prompt,omitempty"`                 // User prompt template (optional)
	Variables    JSONMap   `gorm:"type:json" json:"variables"`                             // Available variables for the template
	MaxTokens    int       `gorm:"default:1000" json:"max_tokens"`                         // Maximum tokens for response
	Temperature  float64   `gorm:"default:0.7" json:"temperature"`                         // Temperature setting
	IsActive     bool      `gorm:"default:true" json:"is_active"`                          // Whether this template is active
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	DeletedAt    DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// AIGeneratedTemplate represents an AI-generated email template
type AIGeneratedTemplate struct {
	ID               uint                     `gorm:"primaryKey" json:"id"`
	Name             string                   `gorm:"not null" json:"name"`                 // Template name
	Description      string                   `json:"description,omitempty"`                // Template description
	PromptTemplateID uint                     `gorm:"not null" json:"prompt_template_id"`   // Reference to the prompt template used
	PromptTemplate   AIPromptTemplate         `gorm:"foreignKey:PromptTemplateID" json:"-"` // Prompt template relation
	UserInput        string                   `gorm:"type:text" json:"user_input"`          // User's input for generation
	GeneratedContent string                   `gorm:"type:text" json:"generated_content"`   // AI-generated content
	ExtractorConfig  ExtractorTemplateConfigs `gorm:"type:json" json:"extractor_config"`    // Generated extractor configuration
	Model            string                   `json:"model"`                                // Model used for generation
	TokensUsed       int                      `json:"tokens_used"`                          // Tokens consumed
	CreatedBy        string                   `json:"created_by,omitempty"`                 // User who created this
	CreatedAt        time.Time                `json:"created_at"`
	UpdatedAt        time.Time                `json:"updated_at"`
	DeletedAt        DeletedAt                `gorm:"index" json:"deleted_at,omitempty"`
}

// PluginInfo 插件信息
type PluginInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Schema      map[string]interface{} `json:"schema"`
}

// ConditionInfo 条件信息
type ConditionInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Type        string                 `json:"type"`
	Schema      map[string]interface{} `json:"schema"`
}

// Trigger 触发器（V1版本，用于兼容）
type Trigger struct {
	ID                uint                   `gorm:"primaryKey" json:"id"`
	UserID            uint                   `gorm:"not null;index" json:"user_id"`
	Name              string                 `gorm:"not null" json:"name"`
	Description       string                 `json:"description"`
	Status            TriggerStatus          `gorm:"default:'active'" json:"status"`
	Enabled           bool                   `gorm:"default:true" json:"enabled"`
	Conditions        TriggerConditionConfig `gorm:"type:json" json:"conditions"`
	Actions           []TriggerActionConfig  `gorm:"type:json" json:"actions"`
	Priority          int                    `gorm:"default:0" json:"priority"`
	TotalExecutions   int64                  `gorm:"default:0" json:"total_executions"`
	SuccessExecutions int64                  `gorm:"default:0" json:"success_executions"`
	LastExecutedAt    *time.Time             `json:"last_executed_at,omitempty"`
	LastError         string                 `json:"last_error,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// OAuth2GlobalConfig represents global OAuth2 configuration for different providers
type OAuth2GlobalConfig struct {
	ID           uint             `gorm:"primaryKey" json:"id"`
	Name         string           `gorm:"uniqueIndex;type:varchar(255)" json:"name"`            // 配置名称，用于区分不同的OAuth2配置
	ProviderType MailProviderType `gorm:"type:varchar(50);not null;index" json:"provider_type"` // 去掉唯一约束，改为普通索引
	ClientID     string           `gorm:"not null" json:"client_id"`
	ClientSecret string           `gorm:"not null" json:"client_secret"`
	RedirectURI  string           `gorm:"not null" json:"redirect_uri"`
	Scopes       StringSlice      `gorm:"type:json" json:"scopes"`
	IsEnabled    bool             `gorm:"default:true" json:"is_enabled"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
	DeletedAt    DeletedAt        `gorm:"index" json:"deleted_at,omitempty"`
}
