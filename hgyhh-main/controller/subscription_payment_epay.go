package controller

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

type SubscriptionEpayPayRequest struct {
	PlanId        int    `json:"plan_id"`
	PaymentMethod string `json:"payment_method"`
}

func SubscriptionRequestEpay(c *gin.Context) {
	var req SubscriptionEpayPayRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	plan, err := model.GetSubscriptionPlanById(req.PlanId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !plan.Enabled {
		common.ApiErrorMsg(c, "套餐未启用")
		return
	}
	if plan.PriceAmount < 0.01 {
		common.ApiErrorMsg(c, "套餐金额过低")
		return
	}

	pid := strings.TrimSpace(operation_setting.EpayId)
	secret := strings.TrimSpace(operation_setting.EpayKey)
	if pid == "" || secret == "" {
		common.ApiErrorMsg(c, "当前管理员未配置支付信息")
		return
	}

	userId := c.GetInt("id")
	if plan.MaxPurchasePerUser > 0 {
		count, err := model.CountUserSubscriptionsByPlan(userId, plan.Id)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if count >= int64(plan.MaxPurchasePerUser) {
			common.ApiErrorMsg(c, "已达到该套餐购买上限")
			return
		}
	}

	callBackAddress := service.GetCallbackAddress()
	returnUrl := callBackAddress + "/api/subscription/epay/return"
	notifyUrl := callBackAddress + "/api/subscription/epay/notify"

	tradeNo := fmt.Sprintf("SUBUSR%dNO%s%d", userId, common.GetRandomString(6), time.Now().Unix())
	payMoneyStr := strconv.FormatFloat(plan.PriceAmount, 'f', 2, 64)

	formParams := map[string]string{
		"pid":          pid,
		"type":         "epay",
		"out_trade_no": tradeNo,
		"name":         fmt.Sprintf("SUB:%s", plan.Title),
		"money":        payMoneyStr,
		"notify_url":   notifyUrl,
		"return_url":   returnUrl,
		"sign_type":    "MD5",
	}
	formParams["sign"] = buildLinuxDoSign(formParams, secret)

	endpoint, err := getLinuxDoEpaySubmitURL()
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	formValues := url.Values{}
	for key, value := range formParams {
		formValues.Set(key, value)
	}

	httpClient := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	httpResp, err := httpClient.PostForm(endpoint, formValues)
	if err != nil {
		common.ApiErrorMsg(c, "拉起支付失败")
		return
	}
	defer httpResp.Body.Close()

	redirectURL := httpResp.Header.Get("Location")
	if (httpResp.StatusCode == http.StatusFound || httpResp.StatusCode == http.StatusSeeOther) && redirectURL != "" {
		order := &model.SubscriptionOrder{
			UserId:        userId,
			PlanId:        plan.Id,
			Money:         plan.PriceAmount,
			TradeNo:       tradeNo,
			PaymentMethod: "epay",
			CreateTime:    time.Now().Unix(),
			Status:        common.TopUpStatusPending,
		}
		if err := order.Insert(); err != nil {
			common.ApiErrorMsg(c, "创建订单失败")
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success", "url": redirectURL, "trade_no": tradeNo})
		return
	}

	body, _ := io.ReadAll(httpResp.Body)
	var failResp struct {
		ErrorMsg string `json:"error_msg"`
	}
	if len(body) > 0 {
		_ = common.Unmarshal(body, &failResp)
	}
	if failResp.ErrorMsg == "" {
		failResp.ErrorMsg = "拉起支付失败"
	}
	common.ApiErrorMsg(c, failResp.ErrorMsg)
}

func SubscriptionEpayNotify(c *gin.Context) {
	var params map[string]string

	if c.Request.Method == "POST" {
		if err := c.Request.ParseForm(); err != nil {
			_, _ = c.Writer.Write([]byte("fail"))
			return
		}
		params = lo.Reduce(lo.Keys(c.Request.PostForm), func(r map[string]string, t string, i int) map[string]string {
			r[t] = c.Request.PostForm.Get(t)
			return r
		}, map[string]string{})
	} else {
		params = lo.Reduce(lo.Keys(c.Request.URL.Query()), func(r map[string]string, t string, i int) map[string]string {
			r[t] = c.Request.URL.Query().Get(t)
			return r
		}, map[string]string{})
	}

	if len(params) == 0 {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	secret := strings.TrimSpace(operation_setting.EpayKey)
	if secret == "" {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	expectedSign := buildLinuxDoSign(params, secret)
	receivedSign := strings.ToLower(strings.TrimSpace(params["sign"]))
	if expectedSign != receivedSign {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	if params["trade_status"] != "TRADE_SUCCESS" {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	tradeNo := strings.TrimSpace(params["out_trade_no"])
	if tradeNo == "" {
		tradeNo = strings.TrimSpace(params["trade_no"])
	}
	if tradeNo == "" {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)

	if err := model.CompleteSubscriptionOrder(tradeNo, common.GetJsonString(params)); err != nil {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	_, _ = c.Writer.Write([]byte("success"))
}

// SubscriptionEpayReturn handles browser return after payment.
// It verifies the payload and completes the order, then redirects to console.
func SubscriptionEpayReturn(c *gin.Context) {
	var params map[string]string

	if c.Request.Method == "POST" {
		if err := c.Request.ParseForm(); err != nil {
			c.Redirect(http.StatusFound, system_setting.ServerAddress+"/console/subscription?pay=fail")
			return
		}
		params = lo.Reduce(lo.Keys(c.Request.PostForm), func(r map[string]string, t string, i int) map[string]string {
			r[t] = c.Request.PostForm.Get(t)
			return r
		}, map[string]string{})
	} else {
		params = lo.Reduce(lo.Keys(c.Request.URL.Query()), func(r map[string]string, t string, i int) map[string]string {
			r[t] = c.Request.URL.Query().Get(t)
			return r
		}, map[string]string{})
	}

	if len(params) == 0 {
		c.Redirect(http.StatusFound, system_setting.ServerAddress+"/console/subscription?pay=fail")
		return
	}

	secret := strings.TrimSpace(operation_setting.EpayKey)
	if secret == "" {
		c.Redirect(http.StatusFound, system_setting.ServerAddress+"/console/subscription?pay=fail")
		return
	}

	expectedSign := buildLinuxDoSign(params, secret)
	receivedSign := strings.ToLower(strings.TrimSpace(params["sign"]))
	if expectedSign != receivedSign {
		c.Redirect(http.StatusFound, system_setting.ServerAddress+"/console/subscription?pay=fail")
		return
	}

	if params["trade_status"] == "TRADE_SUCCESS" {
		tradeNo := strings.TrimSpace(params["out_trade_no"])
		if tradeNo == "" {
			tradeNo = strings.TrimSpace(params["trade_no"])
		}
		if tradeNo == "" {
			c.Redirect(http.StatusFound, system_setting.ServerAddress+"/console/subscription?pay=fail")
			return
		}
		LockOrder(tradeNo)
		defer UnlockOrder(tradeNo)
		if err := model.CompleteSubscriptionOrder(tradeNo, common.GetJsonString(params)); err != nil {
			c.Redirect(http.StatusFound, system_setting.ServerAddress+"/console/subscription?pay=fail")
			return
		}
		c.Redirect(http.StatusFound, system_setting.ServerAddress+"/console/subscription?pay=success")
		return
	}
	c.Redirect(http.StatusFound, system_setting.ServerAddress+"/console/subscription?pay=pending")
}
