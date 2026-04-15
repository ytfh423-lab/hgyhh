package middleware

import (
	"net/http"
	"net/url"

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

type humanVerificationResponse struct {
	Success bool `json:"success"`
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

		if err := verifyHumanVerification(c.ClientIP(), response); err != nil {
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

func verifyHumanVerification(remoteIP, response string) error {
	verifyURL := turnstileVerifyURL
	secretKey := common.TurnstileSecretKey
	if common.HumanVerificationProvider == "recaptcha" {
		verifyURL = recaptchaVerifyURL
		secretKey = common.RecaptchaSecretKey
	}

	rawRes, err := http.PostForm(verifyURL, url.Values{
		"secret":   {secretKey},
		"response": {response},
		"remoteip": {remoteIP},
	})
	if err != nil {
		return err
	}
	defer rawRes.Body.Close()

	var res humanVerificationResponse
	if err = json.DecodeJson(rawRes.Body, &res); err != nil {
		return err
	}
	if !res.Success {
		return &humanVerificationError{message: getHumanVerificationFailedMessage()}
	}
	return nil
}

type humanVerificationError struct {
	message string
}

func (e *humanVerificationError) Error() string {
	return e.message
}
