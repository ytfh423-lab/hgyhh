package model

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"time"
)

// ========== AI 自动审核配置 ==========

// FarmBetaAIConfig AI 审核配置（单例，仅一条记录）
type FarmBetaAIConfig struct {
	Id                    int     `json:"id" gorm:"primaryKey"`
	Enabled               bool    `json:"enabled" gorm:"default:false"`
	ApiBaseUrl            string  `json:"api_base_url" gorm:"type:varchar(512)"`
	ModelName             string  `json:"model_name" gorm:"type:varchar(128)"`
	ApiKeyEncrypted       string  `json:"-" gorm:"type:text;column:api_key_encrypted"`
	SystemPrompt          string  `json:"system_prompt" gorm:"type:text"`
	AutoApproveConfidence int     `json:"auto_approve_confidence" gorm:"default:85"`
	AutoRejectConfidence  int     `json:"auto_reject_confidence" gorm:"default:80"`
	AllowAutoApplyResult  bool    `json:"allow_auto_apply_result" gorm:"default:true"`
	LogRawResponse        bool    `json:"log_raw_response" gorm:"default:true"`
	TimeoutMs             int     `json:"timeout_ms" gorm:"default:30000"`
	JsonMode              bool    `json:"json_mode" gorm:"default:true"`
	PromptVersion         int     `json:"prompt_version" gorm:"default:1"`
	DailyQuota            int     `json:"daily_quota" gorm:"default:0"`
	UpdatedBy             int     `json:"updated_by" gorm:"default:0"`
	CreatedAt             int64   `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt             int64   `json:"updated_at" gorm:"autoUpdateTime"`
}

func (FarmBetaAIConfig) TableName() string {
	return "farm_beta_ai_configs"
}

// FarmBetaAIReviewLog AI 审核日志
type FarmBetaAIReviewLog struct {
	Id                   int     `json:"id" gorm:"primaryKey;autoIncrement"`
	ApplicationId        int     `json:"application_id" gorm:"index"`
	UserId               int     `json:"user_id" gorm:"index"`
	SystemPromptSnapshot string  `json:"system_prompt_snapshot" gorm:"type:text"`
	ModelName            string  `json:"model_name" gorm:"type:varchar(128)"`
	ApiBaseUrl           string  `json:"api_base_url" gorm:"type:varchar(512)"`
	RequestPayload       string  `json:"request_payload" gorm:"type:text"`
	AiDecision           string  `json:"ai_decision" gorm:"type:varchar(20)"`
	AiConfidence         float64 `json:"ai_confidence"`
	AiScore              float64 `json:"ai_score"`
	AiSummary            string  `json:"ai_summary" gorm:"type:text"`
	AiReasons            string  `json:"ai_reasons" gorm:"type:text"`
	AiRawResponse        string  `json:"ai_raw_response" gorm:"type:text"`
	FinalAction          string  `json:"final_action" gorm:"type:varchar(30)"`
	Status               string  `json:"status" gorm:"type:varchar(20);default:'completed'"`
	ErrorMessage         string  `json:"error_message" gorm:"type:text"`
	PromptVersion        int     `json:"prompt_version"`
	CreatedAt            int64   `json:"created_at" gorm:"autoCreateTime"`
}

func (FarmBetaAIReviewLog) TableName() string {
	return "farm_beta_ai_review_logs"
}

// ========== 加密工具 ==========

var aiEncryptionKey []byte

func getAIEncryptionKey() []byte {
	if aiEncryptionKey != nil {
		return aiEncryptionKey
	}
	key := os.Getenv("ENCRYPTION_KEY")
	if key == "" {
		key = os.Getenv("SESSION_SECRET")
	}
	if key == "" {
		key = "farm-beta-ai-default-key-32b!!"
	}
	b := []byte(key)
	if len(b) >= 32 {
		aiEncryptionKey = b[:32]
	} else {
		padded := make([]byte, 32)
		copy(padded, b)
		aiEncryptionKey = padded
	}
	return aiEncryptionKey
}

func EncryptAPIKey(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	key := getAIEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func DecryptAPIKey(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}
	key := getAIEncryptionKey()
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// ========== AI 配置 CRUD ==========

func GetFarmBetaAIConfig() (*FarmBetaAIConfig, error) {
	var config FarmBetaAIConfig
	err := DB.First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func SaveFarmBetaAIConfig(config *FarmBetaAIConfig) error {
	var existing FarmBetaAIConfig
	err := DB.First(&existing).Error
	if err != nil {
		config.Id = 1
		return DB.Create(config).Error
	}
	config.Id = existing.Id
	return DB.Save(config).Error
}

func IsBetaAIReviewEnabled() bool {
	config, err := GetFarmBetaAIConfig()
	if err != nil {
		return false
	}
	return config.Enabled
}

func GetBetaAIDailyQuota() int {
	config, err := GetFarmBetaAIConfig()
	if err != nil {
		return 0
	}
	return config.DailyQuota
}

// CountTodayApprovedBetaApplications 统计今日通过的申请数
func CountTodayApprovedBetaApplications() int64 {
	var count int64
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
	DB.Model(&FarmBetaApplication{}).Where("status = ? AND reviewed_at >= ?", "approved", todayStart).Count(&count)
	return count
}

// ========== AI 审核日志 CRUD ==========

func CreateAIReviewLog(log *FarmBetaAIReviewLog) error {
	return DB.Create(log).Error
}

func GetAIReviewLogsByApplicationId(appId int) ([]*FarmBetaAIReviewLog, error) {
	var logs []*FarmBetaAIReviewLog
	err := DB.Where("application_id = ?", appId).Order("id desc").Find(&logs).Error
	return logs, err
}

func GetAIReviewLogById(id int) (*FarmBetaAIReviewLog, error) {
	var log FarmBetaAIReviewLog
	err := DB.Where("id = ?", id).First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func GetAIReviewLogList(page, pageSize int) ([]*FarmBetaAIReviewLog, int64, error) {
	var logs []*FarmBetaAIReviewLog
	var total int64
	DB.Model(&FarmBetaAIReviewLog{}).Count(&total)
	err := DB.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error
	return logs, total, err
}

// DefaultBetaAISystemPrompt 默认前置提示词模板
const DefaultBetaAISystemPrompt = `你是农场内测资格审核助手。请根据用户的申请信息，判断是否应该给予内测资格。

## 审核标准
1. **申请理由质量**：理由是否真诚、具体，表达了对农场玩法的兴趣和参与意愿
2. **内容有效性**：理由不能是无意义的重复文字、乱码或敷衍内容
3. **LinuxDo 社区参与**：如果提供了 LinuxDo 论坛链接，可作为加分项
4. **申请历史**：多次被拒绝后重新申请需要更充分的理由

## 评分规则
- 90-100分：理由详细具体，表达了明确的参与意愿和反馈承诺 → approve
- 70-89分：理由基本合理，但不够详细 → manual_review
- 50-69分：理由模糊或过于简短 → manual_review
- 0-49分：明显无意义、乱码、恶意内容 → reject

## 输出要求
请严格按以下 JSON 格式输出，不要输出任何其他内容：
{
  "decision": "approve 或 reject 或 manual_review",
  "confidence": 0.0到1.0的置信度,
  "score": 0到100的评分,
  "summary": "一句话总结审核结论",
  "reasons": ["理由1", "理由2"],
  "risk_flags": ["风险标记，如无则为空数组"],
  "suggested_review_note": "建议的审核备注"
}`
