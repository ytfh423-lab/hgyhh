package controller

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
)

func GetTopUpInfo(c *gin.Context) {
	// 获取支付方式
	payMethods := operation_setting.PayMethods

	// 如果启用了 Stripe 支付，添加到支付方法列表
	if setting.StripeApiSecret != "" && setting.StripeWebhookSecret != "" && setting.StripePriceId != "" {
		// 检查是否已经包含 Stripe
		hasStripe := false
		for _, method := range payMethods {
			if method["type"] == "stripe" {
				hasStripe = true
				break
			}
		}

		if !hasStripe {
			stripeMethod := map[string]string{
				"name":      "Stripe",
				"type":      "stripe",
				"color":     "rgba(var(--semi-purple-5), 1)",
				"min_topup": strconv.Itoa(setting.StripeMinTopUp),
			}
			payMethods = append(payMethods, stripeMethod)
		}
	}

	data := gin.H{
		"enable_online_topup": operation_setting.PayAddress != "" && operation_setting.EpayId != "" && operation_setting.EpayKey != "",
		"enable_stripe_topup": setting.StripeApiSecret != "" && setting.StripeWebhookSecret != "" && setting.StripePriceId != "",
		"enable_creem_topup":  setting.CreemApiKey != "" && setting.CreemProducts != "[]",
		"creem_products":      setting.CreemProducts,
		"pay_methods":         payMethods,
		"min_topup":           operation_setting.MinTopUp,
		"stripe_min_topup":    setting.StripeMinTopUp,
		"amount_options":      operation_setting.GetPaymentSetting().AmountOptions,
		"discount":            operation_setting.GetPaymentSetting().AmountDiscount,
	}
	common.ApiSuccess(c, data)
}

type EpayRequest struct {
	Amount        int64  `json:"amount"`
	PaymentMethod string `json:"payment_method"`
}

type AmountRequest struct {
	Amount int64 `json:"amount"`
}

type LinuxDoEpayRequest struct {
	Amount int64  `json:"amount"`
	Name   string `json:"name"`
}

type LinuxDoOrderQueryRequest struct {
	TradeNo string `json:"trade_no"`
}

type LinuxDoRefundRequest struct {
	TradeNo string `json:"trade_no"`
}

type linuxDoOrderQueryResponse struct {
	Code       int    `json:"code"`
	Msg        string `json:"msg"`
	TradeNo    string `json:"trade_no"`
	OutTradeNo string `json:"out_trade_no"`
	Type       string `json:"type"`
	Pid        string `json:"pid"`
	AddTime    string `json:"addtime"`
	EndTime    string `json:"endtime"`
	Name       string `json:"name"`
	Money      string `json:"money"`
	Status     int    `json:"status"`
}

type linuxDoCommonResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func GetEpayClient() *epay.Client {
	if operation_setting.PayAddress == "" || operation_setting.EpayId == "" || operation_setting.EpayKey == "" {
		return nil
	}
	withUrl, err := epay.NewClient(&epay.Config{
		PartnerID: operation_setting.EpayId,
		Key:       operation_setting.EpayKey,
	}, operation_setting.PayAddress)
	if err != nil {
		return nil
	}
	return withUrl
}

func getPayMoney(amount int64, group string) float64 {
	dAmount := decimal.NewFromInt(amount)
	// 充值金额以“展示类型”为准：
	// - USD/CNY: 前端传 amount 为金额单位；TOKENS: 前端传 tokens，需要换成 USD 金额
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		dAmount = dAmount.Div(dQuotaPerUnit)
	}

	topupGroupRatio := common.GetTopupGroupRatio(group)
	if topupGroupRatio == 0 {
		topupGroupRatio = 1
	}

	dTopupGroupRatio := decimal.NewFromFloat(topupGroupRatio)
	dPrice := decimal.NewFromFloat(operation_setting.Price)
	// apply optional preset discount by the original request amount (if configured), default 1.0
	discount := 1.0
	if ds, ok := operation_setting.GetPaymentSetting().AmountDiscount[int(amount)]; ok {
		if ds > 0 {
			discount = ds
		}
	}
	dDiscount := decimal.NewFromFloat(discount)

	payMoney := dAmount.Mul(dPrice).Mul(dTopupGroupRatio).Mul(dDiscount)

	return payMoney.InexactFloat64()
}

func getMinTopup() int64 {
	minTopup := operation_setting.MinTopUp
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dMinTopup := decimal.NewFromInt(int64(minTopup))
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		minTopup = int(dMinTopup.Mul(dQuotaPerUnit).IntPart())
	}
	return int64(minTopup)
}

func RequestEpay(c *gin.Context) {
	var req EpayRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if req.Amount < getMinTopup() {
		c.JSON(200, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getMinTopup())})
		return
	}

	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getPayMoney(req.Amount, group)
	if payMoney < 0.01 {
		c.JSON(200, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	if !operation_setting.ContainsPayMethod(req.PaymentMethod) {
		c.JSON(200, gin.H{"message": "error", "data": "支付方式不存在"})
		return
	}

	callBackAddress := service.GetCallbackAddress()
	returnUrl, _ := url.Parse(system_setting.ServerAddress + "/console/log")
	notifyUrl, _ := url.Parse(callBackAddress + "/api/user/epay/notify")
	tradeNo := fmt.Sprintf("%s%d", common.GetRandomString(6), time.Now().Unix())
	tradeNo = fmt.Sprintf("USR%dNO%s", id, tradeNo)
	client := GetEpayClient()
	if client == nil {
		c.JSON(200, gin.H{"message": "error", "data": "当前管理员未配置支付信息"})
		return
	}
	uri, params, err := client.Purchase(&epay.PurchaseArgs{
		Type:           req.PaymentMethod,
		ServiceTradeNo: tradeNo,
		Name:           fmt.Sprintf("TUC%d", req.Amount),
		Money:          strconv.FormatFloat(payMoney, 'f', 2, 64),
		Device:         epay.PC,
		NotifyUrl:      notifyUrl,
		ReturnUrl:      returnUrl,
	})
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}
	amount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dAmount := decimal.NewFromInt(int64(amount))
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		amount = dAmount.Div(dQuotaPerUnit).IntPart()
	}
	topUp := &model.TopUp{
		UserId:        id,
		Amount:        amount,
		Money:         payMoney,
		TradeNo:       tradeNo,
		PaymentMethod: req.PaymentMethod,
		CreateTime:    time.Now().Unix(),
		Status:        "pending",
	}
	err = topUp.Insert()
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}
	c.JSON(200, gin.H{"message": "success", "data": params, "url": uri})
}

func buildLinuxDoSign(params map[string]string, secret string) string {
	keys := make([]string, 0, len(params))
	for key, value := range params {
		if key == "sign" || key == "sign_type" || value == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	builder := strings.Builder{}
	for index, key := range keys {
		if index > 0 {
			builder.WriteString("&")
		}
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(params[key])
	}
	builder.WriteString(secret)

	h := md5.Sum([]byte(builder.String()))
	return hex.EncodeToString(h[:])
}

func getLinuxDoEpaySubmitURL() (string, error) {
	payAddress := strings.TrimSpace(operation_setting.PayAddress)
	if payAddress == "" {
		return "", errors.New("当前管理员未配置支付网关地址")
	}
	payAddress = strings.TrimRight(payAddress, "/")
	return payAddress + "/pay/submit.php", nil
}

func getLinuxDoEpayAPIURL() (string, error) {
	payAddress := strings.TrimSpace(operation_setting.PayAddress)
	if payAddress == "" {
		return "", errors.New("当前管理员未配置支付网关地址")
	}
	payAddress = strings.TrimRight(payAddress, "/")
	return payAddress + "/api.php", nil
}

func queryLinuxDoOrder(outTradeNo string) (*linuxDoOrderQueryResponse, error) {
	pid := strings.TrimSpace(operation_setting.EpayId)
	secret := strings.TrimSpace(operation_setting.EpayKey)
	if pid == "" || secret == "" {
		return nil, errors.New("当前管理员未配置支付信息")
	}
	apiURL, err := getLinuxDoEpayAPIURL()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	query := req.URL.Query()
	query.Set("act", "order")
	query.Set("pid", pid)
	query.Set("key", secret)
	query.Set("out_trade_no", outTradeNo)
	req.URL.RawQuery = query.Encode()

	client := service.GetHttpClient()
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	result := &linuxDoOrderQueryResponse{}
	if err := common.Unmarshal(body, result); err != nil {
		return nil, err
	}

	if result.Code != 1 {
		if result.Msg == "" {
			result.Msg = "订单不存在或已完成"
		}
		return nil, errors.New(result.Msg)
	}
	return result, nil
}

func refundLinuxDoOrder(tradeNo string, outTradeNo string, money string) error {
	pid := strings.TrimSpace(operation_setting.EpayId)
	secret := strings.TrimSpace(operation_setting.EpayKey)
	if pid == "" || secret == "" {
		return errors.New("当前管理员未配置支付信息")
	}
	apiURL, err := getLinuxDoEpayAPIURL()
	if err != nil {
		return err
	}

	formValues := url.Values{}
	formValues.Set("pid", pid)
	formValues.Set("key", secret)
	formValues.Set("trade_no", tradeNo)
	formValues.Set("money", money)
	if outTradeNo != "" {
		formValues.Set("out_trade_no", outTradeNo)
	}

	client := service.GetHttpClient()
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.PostForm(apiURL, formValues)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	result := &linuxDoCommonResponse{}
	if err := common.Unmarshal(body, result); err != nil {
		return err
	}
	if result.Code != 1 {
		if result.Msg == "" {
			result.Msg = "退款失败"
		}
		return errors.New(result.Msg)
	}
	return nil
}

func RequestLinuxDoEpay(c *gin.Context) {
	var req LinuxDoEpayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if req.Amount < getMinTopup() {
		c.JSON(200, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getMinTopup())})
		return
	}

	pid := strings.TrimSpace(operation_setting.EpayId)
	secret := strings.TrimSpace(operation_setting.EpayKey)
	if pid == "" || secret == "" {
		c.JSON(200, gin.H{"message": "error", "data": "当前管理员未配置支付信息"})
		return
	}

	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getPayMoney(req.Amount, group)
	if payMoney <= 0 {
		c.JSON(200, gin.H{"message": "error", "data": "金额必须大于0"})
		return
	}

	payMoneyStr := strconv.FormatFloat(payMoney, 'f', 2, 64)
	if strings.HasPrefix(payMoneyStr, "0.00") {
		c.JSON(200, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	callBackAddress := service.GetCallbackAddress()
	returnUrl := system_setting.ServerAddress + "/console/log"
	notifyUrl := callBackAddress + "/api/user/linuxdo/notify"

	tradeNo := fmt.Sprintf("USR%dNO%s%d", id, common.GetRandomString(6), time.Now().Unix())
	title := strings.TrimSpace(req.Name)
	if title == "" {
		title = fmt.Sprintf("TUC%d", req.Amount)
	}

	formParams := map[string]string{
		"pid":          pid,
		"type":         "epay",
		"out_trade_no": tradeNo,
		"name":         title,
		"money":        payMoneyStr,
		"notify_url":   notifyUrl,
		"return_url":   returnUrl,
		"sign_type":    "MD5",
	}
	formParams["sign"] = buildLinuxDoSign(formParams, secret)

	endpoint, err := getLinuxDoEpaySubmitURL()
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": err.Error()})
		return
	}

	formValues := url.Values{}
	for key, value := range formParams {
		formValues.Set(key, value)
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	httpResp, err := client.PostForm(endpoint, formValues)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}
	defer httpResp.Body.Close()

	redirectURL := httpResp.Header.Get("Location")
	if (httpResp.StatusCode == http.StatusFound || httpResp.StatusCode == http.StatusSeeOther) && redirectURL != "" {
		amount := req.Amount
		if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
			dAmount := decimal.NewFromInt(amount)
			dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
			amount = dAmount.Div(dQuotaPerUnit).IntPart()
		}

		topUp := &model.TopUp{
			UserId:        id,
			Amount:        amount,
			Money:         payMoney,
			TradeNo:       tradeNo,
			PaymentMethod: "epay",
			CreateTime:    time.Now().Unix(),
			Status:        common.TopUpStatusPending,
		}
		if err = topUp.Insert(); err != nil {
			c.JSON(200, gin.H{"message": "error", "data": "创建订单失败"})
			return
		}

		c.JSON(200, gin.H{"message": "success", "url": redirectURL, "trade_no": tradeNo})
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
	c.JSON(200, gin.H{"message": "error", "data": failResp.ErrorMsg})
}

func LinuxDoEpayNotify(c *gin.Context) {
	var params map[string]string
	if c.Request.Method == http.MethodPost {
		if err := c.Request.ParseForm(); err != nil {
			_, _ = c.Writer.Write([]byte("fail"))
			return
		}
		params = lo.Reduce(lo.Keys(c.Request.PostForm), func(result map[string]string, key string, _ int) map[string]string {
			result[key] = c.Request.PostForm.Get(key)
			return result
		}, map[string]string{})
	} else {
		params = lo.Reduce(lo.Keys(c.Request.URL.Query()), func(result map[string]string, key string, _ int) map[string]string {
			result[key] = c.Request.URL.Query().Get(key)
			return result
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

	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	if topUp.Status == common.TopUpStatusPending {
		topUp.Status = common.TopUpStatusSuccess
		topUp.CompleteTime = time.Now().Unix()
		if err := topUp.Update(); err != nil {
			_, _ = c.Writer.Write([]byte("fail"))
			return
		}

		dAmount := decimal.NewFromInt(topUp.Amount)
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		quotaToAdd := int(dAmount.Mul(dQuotaPerUnit).IntPart())
		if err := model.IncreaseUserQuota(topUp.UserId, quotaToAdd, true); err != nil {
			_, _ = c.Writer.Write([]byte("fail"))
			return
		}

		model.RecordLog(topUp.UserId, model.LogTypeTopup, fmt.Sprintf("使用 LinuxDO 积分充值成功，充值金额: %v，支付金额：%f", logger.LogQuota(quotaToAdd), topUp.Money))
	}

	_, _ = c.Writer.Write([]byte("success"))
}

func QueryLinuxDoOrder(c *gin.Context) {
	var req LinuxDoOrderQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.TradeNo) == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	tradeNo := strings.TrimSpace(req.TradeNo)
	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil {
		common.ApiErrorMsg(c, "订单不存在")
		return
	}
	if topUp.UserId != c.GetInt("id") {
		common.ApiErrorMsg(c, "无权访问该订单")
		return
	}

	queryResp, err := queryLinuxDoOrder(tradeNo)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	common.ApiSuccess(c, queryResp)
}

func RefundLinuxDoOrder(c *gin.Context) {
	if c.GetInt("role") < common.RoleAdminUser {
		common.ApiErrorMsg(c, "仅管理员可操作退款")
		return
	}

	var req LinuxDoRefundRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.TradeNo) == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	tradeNo := strings.TrimSpace(req.TradeNo)
	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)

	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil {
		common.ApiErrorMsg(c, "订单不存在")
		return
	}
	if topUp.Status == common.TopUpStatusRefunded {
		common.ApiSuccess(c, gin.H{"status": common.TopUpStatusRefunded})
		return
	}
	if topUp.Status != common.TopUpStatusSuccess {
		common.ApiErrorMsg(c, "仅支持对已成功订单退款")
		return
	}

	queryResp, err := queryLinuxDoOrder(tradeNo)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	if queryResp.Status != 1 {
		common.ApiErrorMsg(c, "订单不存在或已完成")
		return
	}

	money := strconv.FormatFloat(topUp.Money, 'f', 2, 64)
	remoteTradeNo := strings.TrimSpace(queryResp.TradeNo)
	if remoteTradeNo == "" {
		remoteTradeNo = tradeNo
	}

	if err := refundLinuxDoOrder(remoteTradeNo, tradeNo, money); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	if err := model.RefundTopUp(tradeNo); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	common.ApiSuccess(c, gin.H{"status": common.TopUpStatusRefunded})
}

// tradeNo lock
var orderLocks sync.Map
var createLock sync.Mutex

// LockOrder 尝试对给定订单号加锁
func LockOrder(tradeNo string) {
	lock, ok := orderLocks.Load(tradeNo)
	if !ok {
		createLock.Lock()
		defer createLock.Unlock()
		lock, ok = orderLocks.Load(tradeNo)
		if !ok {
			lock = new(sync.Mutex)
			orderLocks.Store(tradeNo, lock)
		}
	}
	lock.(*sync.Mutex).Lock()
}

// UnlockOrder 释放给定订单号的锁
func UnlockOrder(tradeNo string) {
	lock, ok := orderLocks.Load(tradeNo)
	if ok {
		lock.(*sync.Mutex).Unlock()
	}
}

func EpayNotify(c *gin.Context) {
	var params map[string]string

	if c.Request.Method == "POST" {
		// POST 请求：从 POST body 解析参数
		if err := c.Request.ParseForm(); err != nil {
			log.Println("易支付回调POST解析失败:", err)
			_, _ = c.Writer.Write([]byte("fail"))
			return
		}
		params = lo.Reduce(lo.Keys(c.Request.PostForm), func(r map[string]string, t string, i int) map[string]string {
			r[t] = c.Request.PostForm.Get(t)
			return r
		}, map[string]string{})
	} else {
		// GET 请求：从 URL Query 解析参数
		params = lo.Reduce(lo.Keys(c.Request.URL.Query()), func(r map[string]string, t string, i int) map[string]string {
			r[t] = c.Request.URL.Query().Get(t)
			return r
		}, map[string]string{})
	}

	if len(params) == 0 {
		log.Println("易支付回调参数为空")
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	client := GetEpayClient()
	if client == nil {
		log.Println("易支付回调失败 未找到配置信息")
		_, err := c.Writer.Write([]byte("fail"))
		if err != nil {
			log.Println("易支付回调写入失败")
		}
		return
	}
	verifyInfo, err := client.Verify(params)
	if err == nil && verifyInfo.VerifyStatus {
		_, err := c.Writer.Write([]byte("success"))
		if err != nil {
			log.Println("易支付回调写入失败")
		}
	} else {
		_, err := c.Writer.Write([]byte("fail"))
		if err != nil {
			log.Println("易支付回调写入失败")
		}
		log.Println("易支付回调签名验证失败")
		return
	}

	if verifyInfo.TradeStatus == epay.StatusTradeSuccess {
		log.Println(verifyInfo)
		LockOrder(verifyInfo.ServiceTradeNo)
		defer UnlockOrder(verifyInfo.ServiceTradeNo)
		topUp := model.GetTopUpByTradeNo(verifyInfo.ServiceTradeNo)
		if topUp == nil {
			log.Printf("易支付回调未找到订单: %v", verifyInfo)
			return
		}
		if topUp.Status == "pending" {
			topUp.Status = "success"
			err := topUp.Update()
			if err != nil {
				log.Printf("易支付回调更新订单失败: %v", topUp)
				return
			}
			//user, _ := model.GetUserById(topUp.UserId, false)
			//user.Quota += topUp.Amount * 500000
			dAmount := decimal.NewFromInt(int64(topUp.Amount))
			dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
			quotaToAdd := int(dAmount.Mul(dQuotaPerUnit).IntPart())
			err = model.IncreaseUserQuota(topUp.UserId, quotaToAdd, true)
			if err != nil {
				log.Printf("易支付回调更新用户失败: %v", topUp)
				return
			}
			log.Printf("易支付回调更新用户成功 %v", topUp)
			model.RecordLog(topUp.UserId, model.LogTypeTopup, fmt.Sprintf("使用在线充值成功，充值金额: %v，支付金额：%f", logger.LogQuota(quotaToAdd), topUp.Money))
		}
	} else {
		log.Printf("易支付异常回调: %v", verifyInfo)
	}
}

func RequestAmount(c *gin.Context) {
	var req AmountRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "参数错误"})
		return
	}

	if req.Amount < getMinTopup() {
		c.JSON(200, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getMinTopup())})
		return
	}
	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getPayMoney(req.Amount, group)
	if payMoney <= 0.01 {
		c.JSON(200, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}
	c.JSON(200, gin.H{"message": "success", "data": strconv.FormatFloat(payMoney, 'f', 2, 64)})
}

func GetUserTopUps(c *gin.Context) {
	userId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")

	var (
		topups []*model.TopUp
		total  int64
		err    error
	)
	if keyword != "" {
		topups, total, err = model.SearchUserTopUps(userId, keyword, pageInfo)
	} else {
		topups, total, err = model.GetUserTopUps(userId, pageInfo)
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(topups)
	common.ApiSuccess(c, pageInfo)
}

// GetAllTopUps 管理员获取全平台充值记录
func GetAllTopUps(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")

	var (
		topups []*model.TopUp
		total  int64
		err    error
	)
	if keyword != "" {
		topups, total, err = model.SearchAllTopUps(keyword, pageInfo)
	} else {
		topups, total, err = model.GetAllTopUps(pageInfo)
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(topups)
	common.ApiSuccess(c, pageInfo)
}

type AdminCompleteTopupRequest struct {
	TradeNo string `json:"trade_no"`
}

// AdminCompleteTopUp 管理员补单接口
func AdminCompleteTopUp(c *gin.Context) {
	var req AdminCompleteTopupRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.TradeNo == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	// 订单级互斥，防止并发补单
	LockOrder(req.TradeNo)
	defer UnlockOrder(req.TradeNo)

	if err := model.ManualCompleteTopUp(req.TradeNo); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
