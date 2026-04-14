package common

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type emailAPIRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

func SendEmail(subject string, receiver string, content string) error {
	if EmailAPIUrl == "" {
		return fmt.Errorf("邮件 API 地址未配置")
	}
	parsedURL, err := url.ParseRequestURI(EmailAPIUrl)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		return fmt.Errorf("邮件 API 地址格式无效")
	}
	requestURL := EmailAPIUrl
	if parsedURL.Path == "" || parsedURL.Path == "/" {
		requestURL = strings.TrimRight(EmailAPIUrl, "/") + "/send-email"
	}
	payload := emailAPIRequest{
		To:      receiver,
		Subject: subject,
		HTML:    content,
	}
	body, err := Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := http.Post(requestURL, "application/json", bytes.NewReader(body))
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
