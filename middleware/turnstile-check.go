package middleware

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	json "github.com/QuantumNous/new-api/common"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	humanVerificationSessionKey   = "human_verification"
	legacyTurnstileSessionKey     = "turnstile"
	humanVerificationTokenKey     = "captcha"
	legacyTurnstileTokenKey       = "turnstile"
	turnstileVerifyURL            = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	recaptchaVerifyURL            = "https://www.google.com/recaptcha/api/siteverify"
)

type HumanVerificationResult struct {
	Provider       string
	Success        bool
	Score          float64
	Action         string
	Hostname       string
	ChallengeTS    string `json:"challenge_ts"`
	ErrorCodes     []string `json:"error-codes"`
	ExpectedAction string
	MinScore       float64
}

type HumanVerificationOptions struct {
	ExpectedAction string
	MinScore       float64
	// Version 指定 reCAPTCHA 版本：""（默认按 provider 处理）/"v3"/"v2"
	// v2 用于 v3 风控失败后的 fallback checkbox 验证，不做 score/action 校验
	Version string
}

type humanVerificationResponse struct {
	Success     bool     `json:"success"`
	Score       float64  `json:"score"`
	Action      string   `json:"action"`
	Hostname    string   `json:"hostname"`
	ChallengeTS string   `json:"challenge_ts"`
	ErrorCodes  []string `json:"error-codes"`
}

func TurnstileCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !common.TurnstileCheckEnabled {
			c.Next()
			return
		}

		session := sessions.Default(c)
		if session.Get(humanVerificationSessionKey) != nil || session.Get(legacyTurnstileSessionKey) != nil {
			c.Next()
			return
		}

		response := c.Query(humanVerificationTokenKey)
		if response == "" {
			response = c.Query(legacyTurnstileTokenKey)
		}
		if response == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": getHumanVerificationMissingMessage(),
			})
			c.Abort()
			return
		}

		verifyErr := verifyHumanVerificationWithV2Fallback(c.ClientIP(), response)
		if verifyErr != nil {
			common.SysLog(verifyErr.Error())
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": verifyErr.Error(),
			})
			c.Abort()
			return
		}

		session.Set(humanVerificationSessionKey, true)
		session.Set(legacyTurnstileSessionKey, true)
		if err := session.Save(); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"message": "无法保存会话信息，请重试",
				"success": false,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func getHumanVerificationMissingMessage() string {
	if common.HumanVerificationProvider == "recaptcha" {
		return "reCAPTCHA token 为空"
	}
	return "Turnstile token 为空"
}

func getHumanVerificationFailedMessage() string {
	if common.HumanVerificationProvider == "recaptcha" {
		return "reCAPTCHA 校验失败，请刷新重试！"
	}
	return "Turnstile 校验失败，请刷新重试！"
}

func VerifyHumanVerification(remoteIP, response string, options HumanVerificationOptions) (*HumanVerificationResult, error) {
	verifyURL := turnstileVerifyURL
	secretKey := common.TurnstileSecretKey
	provider := common.HumanVerificationProvider
	if provider == "recaptcha" {
		verifyURL = recaptchaVerifyURL
		secretKey = common.RecaptchaSecretKey
		// v2 fallback：用 v2 secret 校验
		if options.Version == "v2" {
			if common.RecaptchaV2SecretKey != "" {
				secretKey = common.RecaptchaV2SecretKey
			}
		}
	}
	if provider == "" {
		provider = "turnstile"
	}

	rawRes, err := http.PostForm(verifyURL, url.Values{
		"secret":   {secretKey},
		"response": {response},
		"remoteip": {remoteIP},
	})
	if err != nil {
		return nil, err
	}
	defer rawRes.Body.Close()

	var res humanVerificationResponse
	if err = json.DecodeJson(rawRes.Body, &res); err != nil {
		return nil, err
	}

	result := &HumanVerificationResult{
		Provider:       provider,
		Success:        res.Success,
		Score:          res.Score,
		Action:         res.Action,
		Hostname:       res.Hostname,
		ChallengeTS:    res.ChallengeTS,
		ErrorCodes:     res.ErrorCodes,
		ExpectedAction: options.ExpectedAction,
		MinScore:       options.MinScore,
	}

	if !res.Success {
		return result, &humanVerificationError{message: getHumanVerificationFailedMessage()}
	}
	if provider != "recaptcha" {
		return result, nil
	}
	// v2 fallback：只校验 Success，不做 score/action 校验（v2 没有 score）
	if options.Version == "v2" {
		return result, nil
	}
	if options.ExpectedAction != "" && res.Action != options.ExpectedAction {
		return result, &humanVerificationError{message: fmt.Sprintf("reCAPTCHA action 不匹配: expected=%s actual=%s", options.ExpectedAction, res.Action)}
	}
	if options.MinScore > 0 && res.Score < options.MinScore {
		return result, &humanVerificationError{message: fmt.Sprintf("reCAPTCHA score 过低: %.2f < %.2f", res.Score, options.MinScore)}
	}
	return result, nil
}

// verifyHumanVerificationWithV2Fallback 在默认 provider 校验失败时，若 provider=recaptcha
// 且后端同时配置了 v2 siteKey/secretKey，自动用 v2 secret 再校验一次。
//
// 场景：前端 HumanVerification 组件在后端同时配置 v2/v3 时会切到 v2 checkbox 模式，
// 拿到的是 v2 token。如果这里不 fallback，v2 token 会被 v3 secret 拒绝，用户
// 看到"校验失败"而不理解——因为他刚刚明明勾选了那个 checkbox。
//
// 性能：成功路径零影响（首次就过）；失败路径最多多一次 Google siteverify 请求。
func verifyHumanVerificationWithV2Fallback(remoteIP, response string) error {
	_, err := VerifyHumanVerification(remoteIP, response, HumanVerificationOptions{})
	if err == nil {
		return nil
	}
	if common.HumanVerificationProvider != "recaptcha" || !common.IsRecaptchaV2Configured() {
		return err
	}
	if _, err2 := VerifyHumanVerification(remoteIP, response, HumanVerificationOptions{Version: "v2"}); err2 == nil {
		return nil
	}
	return err
}

func ParseHumanVerificationFloat(value string, fallback float64) float64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

type humanVerificationError struct {
	message string
}

func (e *humanVerificationError) Error() string {
	return e.message
}
