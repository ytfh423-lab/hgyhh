package oauth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func init() {
	Register("linuxdo", &LinuxDOProvider{})
}

// LinuxDOProvider implements OAuth for Linux DO
type LinuxDOProvider struct{}

type linuxdoUser struct {
	Id             int    `json:"id"`
	Username       string `json:"username"`
	Name           string `json:"name"`
	Active         bool   `json:"active"`
	TrustLevel     int    `json:"trust_level"`
	Silenced       bool   `json:"silenced"`
	AvatarTemplate string `json:"avatar_template"` // connect.linux.do 可能返回，也可能不返回
}

func (p *LinuxDOProvider) GetName() string {
	return "Linux DO"
}

func (p *LinuxDOProvider) IsEnabled() bool {
	return common.LinuxDOOAuthEnabled
}

func (p *LinuxDOProvider) ExchangeToken(ctx context.Context, code string, c *gin.Context) (*OAuthToken, error) {
	if code == "" {
		return nil, NewOAuthError(i18n.MsgOAuthInvalidCode, nil)
	}

	logger.LogDebug(ctx, "[OAuth-LinuxDO] ExchangeToken: code=%s...", code[:min(len(code), 10)])

	// Get access token using Basic auth
	tokenEndpoint := common.GetEnvOrDefaultString("LINUX_DO_TOKEN_ENDPOINT", "https://connect.linux.do/oauth2/token")
	credentials := common.LinuxDOClientId + ":" + common.LinuxDOClientSecret
	basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(credentials))

	// Get redirect URI from request
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	redirectURI := fmt.Sprintf("%s://%s/api/oauth/linuxdo", scheme, c.Request.Host)

	logger.LogDebug(ctx, "[OAuth-LinuxDO] ExchangeToken: token_endpoint=%s, redirect_uri=%s", tokenEndpoint, redirectURI)

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", basicAuth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := http.Client{Timeout: 5 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-LinuxDO] ExchangeToken error: %s", err.Error()))
		return nil, NewOAuthErrorWithRaw(i18n.MsgOAuthConnectFailed, map[string]any{"Provider": "Linux DO"}, err.Error())
	}
	defer res.Body.Close()

	logger.LogDebug(ctx, "[OAuth-LinuxDO] ExchangeToken response status: %d", res.StatusCode)

	var tokenRes struct {
		AccessToken string `json:"access_token"`
		Message     string `json:"message"`
	}
	if err := json.NewDecoder(res.Body).Decode(&tokenRes); err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-LinuxDO] ExchangeToken decode error: %s", err.Error()))
		return nil, err
	}

	if tokenRes.AccessToken == "" {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-LinuxDO] ExchangeToken failed: %s", tokenRes.Message))
		return nil, NewOAuthErrorWithRaw(i18n.MsgOAuthTokenFailed, map[string]any{"Provider": "Linux DO"}, tokenRes.Message)
	}

	logger.LogDebug(ctx, "[OAuth-LinuxDO] ExchangeToken success")

	return &OAuthToken{
		AccessToken: tokenRes.AccessToken,
	}, nil
}

// buildLinuxDOAvatarURL 构建 LinuxDO 头像 URL。
// 优先使用 connect.linux.do 返回的 avatar_template；
// 若为空则调用 linux.do 公开 JSON API 获取（超时 3s，失败静默忽略）。
func buildLinuxDOAvatarURL(avatarTemplate, username string) string {
	const size = "288"

	fromTemplate := func(tmpl string) string {
		if tmpl == "" {
			return ""
		}
		tmpl = strings.ReplaceAll(tmpl, "{size}", size)
		if strings.HasPrefix(tmpl, "/") {
			return "https://linux.do" + tmpl
		}
		return tmpl
	}

	if avatarTemplate != "" {
		return fromTemplate(avatarTemplate)
	}

	if username == "" {
		return ""
	}

	// 从 linux.do 公开接口获取 avatar_template
	type profileResp struct {
		User struct {
			AvatarTemplate string `json:"avatar_template"`
		} `json:"user"`
	}
	ctx2, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx2, "GET",
		"https://linux.do/u/"+username+".json", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 new-api/avatar-fetch")
	client := http.Client{Timeout: 3 * time.Second}
	res, err := client.Do(req)
	if err != nil || res.StatusCode != 200 {
		return ""
	}
	defer res.Body.Close()

	var p profileResp
	if err := json.NewDecoder(res.Body).Decode(&p); err != nil {
		return ""
	}
	return fromTemplate(p.User.AvatarTemplate)
}

func (p *LinuxDOProvider) GetUserInfo(ctx context.Context, token *OAuthToken) (*OAuthUser, error) {
	userEndpoint := common.GetEnvOrDefaultString("LINUX_DO_USER_ENDPOINT", "https://connect.linux.do/api/user")

	logger.LogDebug(ctx, "[OAuth-LinuxDO] GetUserInfo: user_endpoint=%s", userEndpoint)

	req, err := http.NewRequestWithContext(ctx, "GET", userEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/json")

	client := http.Client{Timeout: 5 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-LinuxDO] GetUserInfo error: %s", err.Error()))
		return nil, NewOAuthErrorWithRaw(i18n.MsgOAuthConnectFailed, map[string]any{"Provider": "Linux DO"}, err.Error())
	}
	defer res.Body.Close()

	logger.LogDebug(ctx, "[OAuth-LinuxDO] GetUserInfo response status: %d", res.StatusCode)

	var linuxdoUser linuxdoUser
	if err := json.NewDecoder(res.Body).Decode(&linuxdoUser); err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-LinuxDO] GetUserInfo decode error: %s", err.Error()))
		return nil, err
	}

	if linuxdoUser.Id == 0 {
		logger.LogError(ctx, "[OAuth-LinuxDO] GetUserInfo failed: invalid user id")
		return nil, NewOAuthError(i18n.MsgOAuthUserInfoEmpty, map[string]any{"Provider": "Linux DO"})
	}

	logger.LogDebug(ctx, "[OAuth-LinuxDO] GetUserInfo: id=%d, username=%s, name=%s, trust_level=%d, active=%v, silenced=%v",
		linuxdoUser.Id, linuxdoUser.Username, linuxdoUser.Name, linuxdoUser.TrustLevel, linuxdoUser.Active, linuxdoUser.Silenced)

	// Check trust level
	if linuxdoUser.TrustLevel < common.LinuxDOMinimumTrustLevel {
		logger.LogWarn(ctx, fmt.Sprintf("[OAuth-LinuxDO] GetUserInfo: trust level too low (required=%d, current=%d)",
			common.LinuxDOMinimumTrustLevel, linuxdoUser.TrustLevel))
		return nil, &TrustLevelError{
			Required: common.LinuxDOMinimumTrustLevel,
			Current:  linuxdoUser.TrustLevel,
		}
	}

	logger.LogDebug(ctx, "[OAuth-LinuxDO] GetUserInfo success: id=%d, username=%s", linuxdoUser.Id, linuxdoUser.Username)

	// 优先用 avatar_template（若 connect.linux.do 返回了的话）
	// 否则从 linux.do 公开用户信息接口拉取头像
	avatarUrl := buildLinuxDOAvatarURL(linuxdoUser.AvatarTemplate, linuxdoUser.Username)

	return &OAuthUser{
		ProviderUserID: strconv.Itoa(linuxdoUser.Id),
		Username:       linuxdoUser.Username,
		DisplayName:    linuxdoUser.Name,
		AvatarUrl:      avatarUrl,
		Extra: map[string]any{
			"trust_level": linuxdoUser.TrustLevel,
			"active":      linuxdoUser.Active,
			"silenced":    linuxdoUser.Silenced,
		},
	}, nil
}

func (p *LinuxDOProvider) IsUserIDTaken(providerUserID string) bool {
	return model.IsLinuxDOIdAlreadyTaken(providerUserID)
}

func (p *LinuxDOProvider) FillUserByProviderID(user *model.User, providerUserID string) error {
	user.LinuxDOId = providerUserID
	return user.FillUserByLinuxDOId()
}

func (p *LinuxDOProvider) SetProviderUserID(user *model.User, providerUserID string) {
	user.LinuxDOId = providerUserID
}

func (p *LinuxDOProvider) GetProviderPrefix() string {
	return "linuxdo_"
}

// TrustLevelError indicates the user's trust level is too low
type TrustLevelError struct {
	Required int
	Current  int
}

func (e *TrustLevelError) Error() string {
	return "trust level too low"
}
