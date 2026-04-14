package common

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
)

type emailAPIRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

type EmailAction struct {
	Label string
	URL   string
}

type EmailTemplateData struct {
	Eyebrow      string
	Title        string
	Greeting     string
	Message      string
	Highlight    string
	Action       *EmailAction
	FallbackText string
	Footer       string
}

func nl2br(text string) string {
	escaped := html.EscapeString(strings.TrimSpace(text))
	return strings.ReplaceAll(escaped, "\n", "<br/>")
}

func normalizeEmailHTML(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "<html") || strings.Contains(lower, "<body") || strings.Contains(lower, "<table") || strings.Contains(lower, "<div") || strings.Contains(lower, "<p") {
		return trimmed
	}
	return "<p style=\"margin:0;color:#475569;font-size:15px;line-height:1.8;\">" + nl2br(trimmed) + "</p>"
}

func RenderEmailTemplate(data EmailTemplateData) string {
	systemName := html.EscapeString(strings.TrimSpace(SystemName))
	if systemName == "" {
		systemName = "系统通知"
	}
	title := html.EscapeString(strings.TrimSpace(data.Title))
	eyebrow := html.EscapeString(strings.TrimSpace(data.Eyebrow))
	greeting := html.EscapeString(strings.TrimSpace(data.Greeting))
	messageHTML := normalizeEmailHTML(data.Message)
	highlight := html.EscapeString(strings.TrimSpace(data.Highlight))
	fallbackText := html.EscapeString(strings.TrimSpace(data.FallbackText))
	footer := html.EscapeString(strings.TrimSpace(data.Footer))
	if footer == "" {
		footer = "此邮件由系统自动发送，请勿直接回复。"
	}

	var builder strings.Builder
	builder.WriteString("<!DOCTYPE html><html lang=\"zh-CN\"><head><meta charset=\"UTF-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\"><title>")
	builder.WriteString(title)
	builder.WriteString("</title></head><body style=\"margin:0;padding:0;background:#f8fafc;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;color:#0f172a;\">")
	builder.WriteString("<div style=\"padding:32px 12px;\"><div style=\"max-width:640px;margin:0 auto;background:#ffffff;border:1px solid #e2e8f0;border-radius:24px;overflow:hidden;box-shadow:0 20px 60px rgba(15,23,42,0.08);\">")
	builder.WriteString("<div style=\"padding:40px 40px 24px;background:linear-gradient(135deg,#2563eb 0%,#7c3aed 100%);color:#ffffff;\">")
	if eyebrow != "" {
		builder.WriteString("<div style=\"font-size:12px;letter-spacing:0.16em;text-transform:uppercase;opacity:0.9;margin-bottom:14px;\">")
		builder.WriteString(eyebrow)
		builder.WriteString("</div>")
	}
	builder.WriteString("<div style=\"font-size:28px;font-weight:700;line-height:1.35;\">")
	builder.WriteString(title)
	builder.WriteString("</div><div style=\"font-size:14px;opacity:0.9;margin-top:10px;\">")
	builder.WriteString(systemName)
	builder.WriteString("</div></div>")
	builder.WriteString("<div style=\"padding:36px 40px 40px;\">")
	if greeting != "" {
		builder.WriteString("<p style=\"margin:0 0 18px;font-size:16px;line-height:1.75;color:#0f172a;\">")
		builder.WriteString(greeting)
		builder.WriteString("</p>")
	}
	if messageHTML != "" {
		builder.WriteString(messageHTML)
	}
	if highlight != "" {
		builder.WriteString("<div style=\"margin-top:24px;padding:20px 24px;border-radius:18px;background:#eff6ff;border:1px solid #bfdbfe;text-align:center;\"><div style=\"font-size:13px;color:#1d4ed8;letter-spacing:0.08em;text-transform:uppercase;margin-bottom:10px;\">关键信息</div><div style=\"font-size:32px;font-weight:800;letter-spacing:0.18em;color:#1e3a8a;\">")
		builder.WriteString(highlight)
		builder.WriteString("</div></div>")
	}
	if data.Action != nil && strings.TrimSpace(data.Action.Label) != "" && strings.TrimSpace(data.Action.URL) != "" {
		builder.WriteString("<div style=\"margin-top:28px;text-align:center;\"><a href=\"")
		builder.WriteString(html.EscapeString(strings.TrimSpace(data.Action.URL)))
		builder.WriteString("\" style=\"display:inline-block;padding:14px 28px;border-radius:999px;background:#2563eb;color:#ffffff;text-decoration:none;font-size:15px;font-weight:700;box-shadow:0 10px 30px rgba(37,99,235,0.28);\">")
		builder.WriteString(html.EscapeString(strings.TrimSpace(data.Action.Label)))
		builder.WriteString("</a></div>")
	}
	if fallbackText != "" {
		builder.WriteString("<div style=\"margin-top:24px;padding:18px 20px;border-radius:16px;background:#f8fafc;border:1px solid #e2e8f0;\"><div style=\"font-size:13px;font-weight:600;color:#334155;margin-bottom:8px;\">备用说明</div><div style=\"font-size:13px;line-height:1.8;color:#475569;word-break:break-all;\">")
		builder.WriteString(strings.ReplaceAll(fallbackText, "\n", "<br/>"))
		builder.WriteString("</div></div>")
	}
	builder.WriteString("<div style=\"margin-top:30px;padding-top:20px;border-top:1px solid #e2e8f0;font-size:12px;line-height:1.8;color:#94a3b8;\">")
	builder.WriteString(footer)
	builder.WriteString("</div></div></div></div></body></html>")
	return builder.String()
}

func SendEmail(subject string, receiver string, content string) error {
	requestURL := "http://38.92.15.13:5000/send-email"
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
