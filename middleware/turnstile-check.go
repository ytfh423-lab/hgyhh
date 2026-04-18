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

		if _, err := VerifyHumanVerification(c.ClientIP(), response, HumanVerificationOptions{}); err != nil {
			common.SysLog(err.Error())
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
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
	if options.ExpectedAction != "" && res.Action != options.ExpectedAction {
		return result, &humanVerificationError{message: fmt.Sprintf("reCAPTCHA action 不匹配: expected=%s actual=%s", options.ExpectedAction, res.Action)}
	}
	if options.MinScore > 0 && res.Score < options.MinScore {
		return result, &humanVerificationError{message: fmt.Sprintf("reCAPTCHA score 过低: %.2f < %.2f", res.Score, options.MinScore)}
	}
	return result, nil
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
