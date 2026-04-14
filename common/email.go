package common

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"slices"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/setting/system_setting"
)

type emailAPIRequest struct {
	To       string `json:"to"`
	Subject  string `json:"subject"`
	HTML     string `json:"html"`
	SMTPHost string `json:"smtp_host"`
	SMTPPort int    `json:"smtp_port"`
	SMTPUser string `json:"smtp_user"`
	SMTPPass string `json:"smtp_pass"`
}

func generateMessageID() (string, error) {
	fromAddress := SMTPFrom
	if fromAddress == "" {
		fromAddress = SMTPAccount
	}
	split := strings.Split(fromAddress, "@")
	if len(split) < 2 {
		return "", fmt.Errorf("invalid SMTP account")
	}
	domain := split[1]
	return fmt.Sprintf("<%d.%s@%s>", time.Now().UnixNano(), GetRandomString(12), domain), nil
}

func validateSMTPConfig() error {
	if SMTPServer == "" || SMTPAccount == "" || SMTPToken == "" || SMTPPort <= 0 {
		return fmt.Errorf("SMTP 服务器未配置完整")
	}
	return nil
}

func SendEmail(subject string, receiver string, content string) error {
	emailSettings := system_setting.GetEmailSettings()
	mode := emailSettings.Mode
	if mode == "" {
		mode = "smtp"
	}
	if mode == "http_api" {
		return sendEmailViaHTTPAPI(emailSettings.ApiUrl, subject, receiver, content)
	}
	return sendEmailViaSMTP(subject, receiver, content)
}

func sendEmailViaHTTPAPI(apiURL string, subject string, receiver string, content string) error {
	if apiURL == "" {
		return fmt.Errorf("邮件 API 地址未配置")
	}
	if !strings.HasPrefix(apiURL, "http://") && !strings.HasPrefix(apiURL, "https://") {
		return fmt.Errorf("邮件 API 地址格式无效")
	}
	if err := validateSMTPConfig(); err != nil {
		return err
	}
	payload := emailAPIRequest{
		To:       receiver,
		Subject:  subject,
		HTML:     content,
		SMTPHost: SMTPServer,
		SMTPPort: SMTPPort,
		SMTPUser: SMTPAccount,
		SMTPPass: SMTPToken,
	}
	body, err := Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := http.Post(apiURL, "application/json", bytes.NewReader(body))
	if err != nil {
		SysError(fmt.Sprintf("failed to call email API for %s: %v", receiver, err))
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("邮件 API 调用失败: %s", strings.TrimSpace(string(respBody)))
		SysError(fmt.Sprintf("failed to call email API for %s: %v", receiver, err))
		return err
	}
	return nil
}

func sendEmailViaSMTP(subject string, receiver string, content string) error {
	fromAddress := SMTPFrom
	if fromAddress == "" {
		fromAddress = SMTPAccount
	}
	id, err2 := generateMessageID()
	if err2 != nil {
		return err2
	}
	if err := validateSMTPConfig(); err != nil {
		return err
	}
	encodedSubject := fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(subject)))
	mail := []byte(fmt.Sprintf("To: %s\r\n"+
		"From: %s <%s>\r\n"+
		"Subject: %s\r\n"+
		"Date: %s\r\n"+
		"Message-ID: %s\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n\r\n%s\r\n",
		receiver, SystemName, fromAddress, encodedSubject, time.Now().Format(time.RFC1123Z), id, content))
	auth := smtp.PlainAuth("", SMTPAccount, SMTPToken, SMTPServer)
	addr := fmt.Sprintf("%s:%d", SMTPServer, SMTPPort)
	to := strings.Split(receiver, ";")
	var err error
	if SMTPPort == 465 || SMTPSSLEnabled {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         SMTPServer,
		}
		conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", SMTPServer, SMTPPort), tlsConfig)
		if err != nil {
			return err
		}
		client, err := smtp.NewClient(conn, SMTPServer)
		if err != nil {
			return err
		}
		defer client.Close()
		if err = client.Auth(auth); err != nil {
			return err
		}
		if err = client.Mail(fromAddress); err != nil {
			return err
		}
		receiverEmails := strings.Split(receiver, ";")
		for _, receiver := range receiverEmails {
			if err = client.Rcpt(receiver); err != nil {
				return err
			}
		}
		w, err := client.Data()
		if err != nil {
			return err
		}
		_, err = w.Write(mail)
		if err != nil {
			return err
		}
		err = w.Close()
		if err != nil {
			return err
		}
	} else if isOutlookServer(SMTPAccount) || slices.Contains(EmailLoginAuthServerList, SMTPServer) {
		auth = LoginAuth(SMTPAccount, SMTPToken)
		err = smtp.SendMail(addr, auth, fromAddress, to, mail)
	} else {
		err = smtp.SendMail(addr, auth, fromAddress, to, mail)
	}
	if err != nil {
		SysError(fmt.Sprintf("failed to send email to %s: %v", receiver, err))
	}
	return err
}
