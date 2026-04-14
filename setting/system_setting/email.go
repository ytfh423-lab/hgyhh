package system_setting

import "github.com/QuantumNous/new-api/setting/config"

type EmailSettings struct {
	Mode   string `json:"mode"`
	ApiUrl string `json:"api_url"`
}

var defaultEmailSettings = EmailSettings{
	Mode: "smtp",
}

func init() {
	config.GlobalConfig.Register("email", &defaultEmailSettings)
}

func GetEmailSettings() *EmailSettings {
	return &defaultEmailSettings
}
