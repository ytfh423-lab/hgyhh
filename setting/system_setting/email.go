package system_setting

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

type EmailSettings struct {
	Mode   string `json:"mode"`
	ApiUrl string `json:"api_url"`
}

var defaultEmailSettings = EmailSettings{
	Mode: common.EmailMode,
}

func init() {
	config.GlobalConfig.Register("email", &defaultEmailSettings)
}

func GetEmailSettings() *EmailSettings {
	common.EmailMode = defaultEmailSettings.Mode
	common.EmailAPIUrl = defaultEmailSettings.ApiUrl
	return &defaultEmailSettings
}
