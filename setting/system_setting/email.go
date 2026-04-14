package system_setting

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

type EmailSettings struct {
	ApiUrl string `json:"api_url"`
}

var defaultEmailSettings = EmailSettings{}

func init() {
	config.GlobalConfig.Register("email", &defaultEmailSettings)
}

func GetEmailSettings() *EmailSettings {
	common.EmailAPIUrl = defaultEmailSettings.ApiUrl
	return &defaultEmailSettings
}
