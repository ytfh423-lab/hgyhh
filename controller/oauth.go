package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/oauth"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	oauthPendingProviderKey       = "oauth_pending_provider"
	oauthPendingProviderUserIDKey = "oauth_pending_provider_user_id"
	oauthPendingUsernameKey       = "oauth_pending_username"
	oauthPendingDisplayNameKey    = "oauth_pending_display_name"
	oauthPendingEmailKey          = "oauth_pending_email"
	oauthPendingLegacyIDKey       = "oauth_pending_legacy_id"
)

type OAuthCompleteRegistrationRequest struct {
	RegistrationCode string `json:"registration_code"`
}

// providerParams returns map with Provider key for i18n templates
func providerParams(name string) map[string]any {
	return map[string]any{"Provider": name}
}

// GenerateOAuthCode generates a state code for OAuth CSRF protection
func GenerateOAuthCode(c *gin.Context) {
	session := sessions.Default(c)
	state := common.GetRandomString(12)
	affCode := c.Query("aff")
	if affCode != "" {
		session.Set("aff", affCode)
	}
	session.Set("oauth_state", state)
	err := session.Save()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    state,
	})
}

// HandleOAuth handles OAuth callback for all standard OAuth providers
func HandleOAuth(c *gin.Context) {
	providerName := c.Param("provider")
	provider := oauth.GetProvider(providerName)
	if provider == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgOAuthUnknownProvider),
		})
		return
	}

	session := sessions.Default(c)

	state := c.Query("state")
	if state == "" || session.Get("oauth_state") == nil || state != session.Get("oauth_state").(string) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgOAuthStateInvalid),
		})
		return
	}

	username := session.Get("username")
	if username != nil {
		handleOAuthBind(c, provider)
		return
	}

	if !provider.IsEnabled() {
		common.ApiErrorI18n(c, i18n.MsgOAuthNotEnabled, providerParams(provider.GetName()))
		return
	}

	errorCode := c.Query("error")
	if errorCode != "" {
		errorDescription := c.Query("error_description")
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": errorDescription,
		})
		return
	}

	code := c.Query("code")
	token, err := provider.ExchangeToken(c.Request.Context(), code, c)
	if err != nil {
		handleOAuthError(c, err)
		return
	}

	oauthUser, err := provider.GetUserInfo(c.Request.Context(), token)
	if err != nil {
		handleOAuthError(c, err)
		return
	}

	user, exists, err := findExistingOAuthUser(provider, oauthUser)
	if err != nil {
		switch err.(type) {
		case *OAuthUserDeletedError:
			common.ApiErrorI18n(c, i18n.MsgOAuthUserDeleted)
		default:
			common.ApiError(c, err)
		}
		return
	}
	if exists {
		if user.Status != common.UserStatusEnabled {
			common.ApiErrorI18n(c, i18n.MsgOAuthUserBanned)
			return
		}
		setupLogin(user, c)
		return
	}

	if !common.RegisterEnabled {
		common.ApiErrorI18n(c, i18n.MsgUserRegisterDisabled)
		return
	}

	if strings.EqualFold(providerName, "linuxdo") {
		setPendingOAuthRegistration(session, providerName, oauthUser)
		if err := session.Save(); err != nil {
			common.ApiError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
			"data": gin.H{
				"registration_required": true,
				"provider":              providerName,
			},
		})
		return
	}

	createdUser, err := createOAuthUser(provider, oauthUser, session, "")
	if err != nil {
		handleOAuthError(c, err)
		return
	}
	if createdUser.Status != common.UserStatusEnabled {
		common.ApiErrorI18n(c, i18n.MsgOAuthUserBanned)
		return
	}
	setupLogin(createdUser, c)
}

func CompleteOAuthRegistration(c *gin.Context) {
	providerName := c.Param("provider")
	provider := oauth.GetProvider(providerName)
	if provider == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgOAuthUnknownProvider),
		})
		return
	}
	if !provider.IsEnabled() {
		common.ApiErrorI18n(c, i18n.MsgOAuthNotEnabled, providerParams(provider.GetName()))
		return
	}

	var req OAuthCompleteRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if req.RegistrationCode == "" {
		common.ApiErrorI18n(c, i18n.MsgUserRegistrationCodeRequired)
		return
	}

	session := sessions.Default(c)
	pendingProvider, pendingOAuthUser, ok := getPendingOAuthRegistration(session)
	if !ok || !strings.EqualFold(pendingProvider, providerName) {
		common.ApiErrorI18n(c, i18n.MsgOAuthRegistrationSessionInvalid)
		return
	}

	user, err := createOAuthUser(provider, pendingOAuthUser, session, req.RegistrationCode)
	if err != nil {
		if errors.Is(err, model.ErrRedeemFailed) {
			common.ApiErrorI18n(c, i18n.MsgUserRegistrationCodeInvalid)
			return
		}
		common.ApiError(c, err)
		return
	}

	clearPendingOAuthRegistration(session)
	if err = session.Save(); err != nil {
		common.ApiError(c, err)
		return
	}

	if user.Status != common.UserStatusEnabled {
		common.ApiErrorI18n(c, i18n.MsgOAuthUserBanned)
		return
	}
	setupLogin(user, c)
}

// handleOAuthBind handles binding OAuth account to existing user
func handleOAuthBind(c *gin.Context, provider oauth.Provider) {
	if !provider.IsEnabled() {
		common.ApiErrorI18n(c, i18n.MsgOAuthNotEnabled, providerParams(provider.GetName()))
		return
	}

	code := c.Query("code")
	token, err := provider.ExchangeToken(c.Request.Context(), code, c)
	if err != nil {
		handleOAuthError(c, err)
		return
	}

	oauthUser, err := provider.GetUserInfo(c.Request.Context(), token)
	if err != nil {
		handleOAuthError(c, err)
		return
	}

	if provider.IsUserIDTaken(oauthUser.ProviderUserID) {
		common.ApiErrorI18n(c, i18n.MsgOAuthAlreadyBound, providerParams(provider.GetName()))
		return
	}
	if legacyID, ok := oauthUser.Extra["legacy_id"].(string); ok && legacyID != "" {
		if provider.IsUserIDTaken(legacyID) {
			common.ApiErrorI18n(c, i18n.MsgOAuthAlreadyBound, providerParams(provider.GetName()))
			return
		}
	}

	session := sessions.Default(c)
	id := session.Get("id")
	user := model.User{Id: id.(int)}
	err = user.FillUserById()
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if genericProvider, ok := provider.(*oauth.GenericOAuthProvider); ok {
		err = model.UpdateUserOAuthBinding(user.Id, genericProvider.GetProviderId(), oauthUser.ProviderUserID)
		if err != nil {
			common.ApiError(c, err)
			return
		}
	} else {
		provider.SetProviderUserID(&user, oauthUser.ProviderUserID)
		err = user.Update(false)
		if err != nil {
			common.ApiError(c, err)
			return
		}
	}

	common.ApiSuccessI18n(c, i18n.MsgOAuthBindSuccess, nil)
}

func findExistingOAuthUser(provider oauth.Provider, oauthUser *oauth.OAuthUser) (*model.User, bool, error) {
	user := &model.User{}

	if provider.IsUserIDTaken(oauthUser.ProviderUserID) {
		err := provider.FillUserByProviderID(user, oauthUser.ProviderUserID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				common.SysLog(fmt.Sprintf("[OAuth] provider id exists but user not found, fallback to registration: provider=%s, provider_user_id=%s", provider.GetName(), oauthUser.ProviderUserID))
			} else {
				return nil, false, err
			}
		} else {
			if user.Id == 0 {
				return nil, false, &OAuthUserDeletedError{}
			}
			return user, true, nil
		}
	}

	if legacyID, ok := oauthUser.Extra["legacy_id"].(string); ok && legacyID != "" {
		if provider.IsUserIDTaken(legacyID) {
			err := provider.FillUserByProviderID(user, legacyID)
			if err != nil {
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, false, err
				}
			} else if user.Id != 0 {
				common.SysLog(fmt.Sprintf("[OAuth] Migrating user %d from legacy_id=%s to new_id=%s", user.Id, legacyID, oauthUser.ProviderUserID))
				if err := user.UpdateGitHubId(oauthUser.ProviderUserID); err != nil {
					common.SysError(fmt.Sprintf("[OAuth] Failed to migrate user %d: %s", user.Id, err.Error()))
				}
				return user, true, nil
			}
		}
	}

	return nil, false, nil
}

func createOAuthUser(provider oauth.Provider, oauthUser *oauth.OAuthUser, session sessions.Session, registrationCode string) (*model.User, error) {
	user := &model.User{}
	user.Username = provider.GetProviderPrefix() + strconv.Itoa(model.GetMaxUserId()+1)

	if oauthUser.Username != "" {
		if exists, err := model.CheckUserExistOrDeleted(oauthUser.Username, ""); err == nil && !exists {
			if len(oauthUser.Username) <= model.UserNameMaxLength {
				user.Username = oauthUser.Username
			}
		}
	}

	if oauthUser.DisplayName != "" {
		user.DisplayName = oauthUser.DisplayName
	} else if oauthUser.Username != "" {
		user.DisplayName = oauthUser.Username
	} else {
		user.DisplayName = provider.GetName() + " User"
	}
	if oauthUser.Email != "" {
		user.Email = oauthUser.Email
	}
	user.Role = common.RoleCommonUser
	user.Status = common.UserStatusEnabled

	affCode := session.Get("aff")
	inviterId := 0
	if affCode != nil {
		inviterId, _ = model.GetUserIdByAffCode(affCode.(string))
	}

	if genericProvider, ok := provider.(*oauth.GenericOAuthProvider); ok {
		err := model.DB.Transaction(func(tx *gorm.DB) error {
			if err := user.InsertWithTx(tx, inviterId); err != nil {
				return err
			}
			binding := &model.UserOAuthBinding{
				UserId:         user.Id,
				ProviderId:     genericProvider.GetProviderId(),
				ProviderUserId: oauthUser.ProviderUserID,
			}
			if err := model.CreateUserOAuthBindingWithTx(tx, binding); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		user.FinalizeOAuthUserCreation(inviterId)
	} else {
		err := model.DB.Transaction(func(tx *gorm.DB) error {
			if err := user.InsertWithTx(tx, inviterId); err != nil {
				return err
			}

			provider.SetProviderUserID(user, oauthUser.ProviderUserID)
			if err := tx.Model(user).Updates(map[string]interface{}{
				"github_id":   user.GitHubId,
				"discord_id":  user.DiscordId,
				"oidc_id":     user.OidcId,
				"linux_do_id": user.LinuxDOId,
				"wechat_id":   user.WeChatId,
				"telegram_id": user.TelegramId,
			}).Error; err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		user.FinalizeOAuthUserCreation(inviterId)
	}

	if registrationCode != "" {
		if _, err := model.ConsumeRedemptionCodeForRegistration(registrationCode, user.Id); err != nil {
			_ = model.DeleteUserById(user.Id)
			return nil, err
		}
	}

	return user, nil
}

func setPendingOAuthRegistration(session sessions.Session, providerName string, oauthUser *oauth.OAuthUser) {
	session.Set(oauthPendingProviderKey, providerName)
	session.Set(oauthPendingProviderUserIDKey, oauthUser.ProviderUserID)
	session.Set(oauthPendingUsernameKey, oauthUser.Username)
	session.Set(oauthPendingDisplayNameKey, oauthUser.DisplayName)
	session.Set(oauthPendingEmailKey, oauthUser.Email)
	legacyID := ""
	if val, ok := oauthUser.Extra["legacy_id"].(string); ok {
		legacyID = val
	}
	session.Set(oauthPendingLegacyIDKey, legacyID)
}

func getPendingOAuthRegistration(session sessions.Session) (string, *oauth.OAuthUser, bool) {
	providerVal := session.Get(oauthPendingProviderKey)
	providerUserIDVal := session.Get(oauthPendingProviderUserIDKey)
	if providerVal == nil || providerUserIDVal == nil {
		return "", nil, false
	}

	providerName, ok := providerVal.(string)
	if !ok || providerName == "" {
		return "", nil, false
	}
	providerUserID, ok := providerUserIDVal.(string)
	if !ok || providerUserID == "" {
		return "", nil, false
	}

	oauthUser := &oauth.OAuthUser{
		ProviderUserID: providerUserID,
		Extra:          map[string]any{},
	}
	if username, ok := session.Get(oauthPendingUsernameKey).(string); ok {
		oauthUser.Username = username
	}
	if displayName, ok := session.Get(oauthPendingDisplayNameKey).(string); ok {
		oauthUser.DisplayName = displayName
	}
	if email, ok := session.Get(oauthPendingEmailKey).(string); ok {
		oauthUser.Email = email
	}
	if legacyID, ok := session.Get(oauthPendingLegacyIDKey).(string); ok && legacyID != "" {
		oauthUser.Extra["legacy_id"] = legacyID
	}
	return providerName, oauthUser, true
}

func clearPendingOAuthRegistration(session sessions.Session) {
	session.Delete(oauthPendingProviderKey)
	session.Delete(oauthPendingProviderUserIDKey)
	session.Delete(oauthPendingUsernameKey)
	session.Delete(oauthPendingDisplayNameKey)
	session.Delete(oauthPendingEmailKey)
	session.Delete(oauthPendingLegacyIDKey)
}

// Error types for OAuth
type OAuthUserDeletedError struct{}

func (e *OAuthUserDeletedError) Error() string {
	return "user has been deleted"
}

// handleOAuthError handles OAuth errors and returns translated message
func handleOAuthError(c *gin.Context, err error) {
	switch e := err.(type) {
	case *oauth.OAuthError:
		if e.Params != nil {
			common.ApiErrorI18n(c, e.MsgKey, e.Params)
		} else {
			common.ApiErrorI18n(c, e.MsgKey)
		}
	case *oauth.AccessDeniedError:
		common.ApiErrorMsg(c, e.Message)
	case *oauth.TrustLevelError:
		common.ApiErrorI18n(c, i18n.MsgOAuthTrustLevelLow)
	default:
		common.ApiError(c, err)
	}
}
