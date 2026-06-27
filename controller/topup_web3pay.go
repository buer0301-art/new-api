package controller

import (
	"bytes"
	"crypto/md5"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

const (
	web3PayGatewayPath     = "/api/gateway/v1"
	web3PayCreateOrderPath = "/orders"
	web3PayOrderStatusOK   = "0"
	web3PaySuccessStatus   = "SUCCESS"
)

type Web3PayRequest struct {
	Amount int64 `json:"amount"`
}

type web3PayCreateOrderRequest struct {
	MerchantOrderNo string `json:"merchantOrderNo"`
	Amount          string `json:"amount"`
	OrderCurrency   string `json:"orderCurrency,omitempty"`
	NotifyUrl       string `json:"notifyUrl"`
	ReturnUrl       string `json:"returnUrl,omitempty"`
	Attach          string `json:"attach,omitempty"`
}

type web3PayChainOption struct {
	Address       string `json:"address"`
	ChainCode     string `json:"chainCode"`
	ChainName     string `json:"chainName"`
	Contract      string `json:"contract"`
	InConfirm     int    `json:"inConfirm"`
	Logo          string `json:"logo"`
	PaymentNotice string `json:"paymentNotice"`
}

type web3PayPaymentOption struct {
	Code  string               `json:"code"`
	Logo  string               `json:"logo"`
	Chain []web3PayChainOption `json:"chain"`
}

type web3PayOrderData struct {
	OrderNo         string                 `json:"orderNo"`
	MerchantOrderNo string                 `json:"merchantOrderNo"`
	OrderAmount     string                 `json:"orderAmount"`
	OrderCurrency   string                 `json:"orderCurrency"`
	PayAmount       string                 `json:"payAmount"`
	PayCurrency     string                 `json:"payCurrency"`
	PayUrl          string                 `json:"payUrl"`
	ExpireTime      string                 `json:"expireTime"`
	Status          string                 `json:"status"`
	Attach          string                 `json:"attach"`
	CreatedAt       string                 `json:"createdAt"`
	PaymentOptions  []web3PayPaymentOption `json:"paymentOptions"`
}

type web3PayAPIResponse struct {
	Code    string           `json:"code"`
	Message string           `json:"message"`
	Data    web3PayOrderData `json:"data"`
}

type web3PayCallback struct {
	Event           string          `json:"event"`
	OrderID         string          `json:"orderId"`
	MerchantOrderNo string          `json:"merchantOrderNo"`
	ChainCode       string          `json:"chainCode"`
	TokenAddress    string          `json:"tokenAddress"`
	Amount          string          `json:"amount"`
	AmountReadable  string          `json:"amountReadable"`
	OrderAmount     string          `json:"orderAmount"`
	OrderCurrency   string          `json:"orderCurrency"`
	PayAmount       string          `json:"payAmount"`
	PayCurrency     string          `json:"payCurrency"`
	TxHash          string          `json:"txHash"`
	PayAddress      string          `json:"payAddress"`
	Attach          string          `json:"attach"`
	SuccessAt       string          `json:"successAt"`
	Timestamp       json.RawMessage `json:"timestamp"`
	Sign            string          `json:"sign"`
}

func isWeb3PayTopUpEnabled() bool {
	return setting.Web3PayEnabled &&
		strings.TrimSpace(setting.Web3PayAppKey) != "" &&
		strings.TrimSpace(setting.Web3PayApiSecret) != ""
}

func getWeb3PayMinTopup() int64 {
	minTopup := setting.Web3PayMinTopUp
	if minTopup <= 0 {
		minTopup = 1
	}
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		minTopup = minTopup * int(common.QuotaPerUnit)
	}
	return int64(minTopup)
}

func getWeb3PayMoney(amount float64, group string) float64 {
	originalAmount := amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		amount = amount / common.QuotaPerUnit
	}
	topupGroupRatio := common.GetTopupGroupRatio(group)
	if topupGroupRatio == 0 {
		topupGroupRatio = 1
	}
	discount := 1.0
	if ds, ok := operation_setting.GetPaymentSetting().AmountDiscount[int(originalAmount)]; ok {
		if ds > 0 {
			discount = ds
		}
	}
	payMoney := decimal.NewFromFloat(amount).
		Mul(decimal.NewFromFloat(setting.Web3PayUnitPrice)).
		Mul(decimal.NewFromFloat(topupGroupRatio)).
		Mul(decimal.NewFromFloat(discount))
	return payMoney.InexactFloat64()
}

func web3PayCanonicalString(params map[string]string, timestamp string) string {
	canonicalParams := make(map[string]string, len(params)+1)
	for key, value := range params {
		value = strings.TrimSpace(value)
		if key == "" || key == "sign" || value == "" {
			continue
		}
		canonicalParams[key] = value
	}
	if strings.TrimSpace(timestamp) != "" {
		canonicalParams["timestamp"] = strings.TrimSpace(timestamp)
	}

	keys := make([]string, 0, len(canonicalParams))
	for key := range canonicalParams {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+canonicalParams[key])
	}
	return strings.Join(parts, "&")
}

func web3PaySign(params map[string]string, timestamp string, apiSecret string) string {
	hash := md5.Sum([]byte(web3PayCanonicalString(params, timestamp) + apiSecret))
	return hex.EncodeToString(hash[:])
}

func web3PaySignCallback(payload map[string]string, apiSecret string) string {
	timestamp := payload["timestamp"]
	params := make(map[string]string, len(payload))
	for key, value := range payload {
		if key == "timestamp" || key == "sign" {
			continue
		}
		params[key] = value
	}
	return web3PaySign(params, timestamp, apiSecret)
}

func web3PayVerifyCallbackSign(payload map[string]string, apiSecret string) bool {
	signature := payload["sign"]
	if signature == "" || apiSecret == "" {
		return false
	}
	expected := web3PaySignCallback(payload, apiSecret)
	return subtle.ConstantTimeCompare([]byte(strings.ToLower(signature)), []byte(expected)) == 1
}

func RequestWeb3PayAmount(c *gin.Context) {
	var req AmountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if req.Amount < getWeb3PayMinTopup() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getWeb3PayMinTopup())})
		return
	}
	group, err := model.GetUserGroup(c.GetInt("id"), true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getWeb3PayMoney(float64(req.Amount), group)
	if payMoney <= 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": strconv.FormatFloat(payMoney, 'f', 2, 64)})
}

func RequestWeb3Pay(c *gin.Context) {
	var req Web3PayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if !isWeb3PayTopUpEnabled() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "当前管理员未配置 Web3 Pay"})
		return
	}
	if req.Amount < getWeb3PayMinTopup() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getWeb3PayMinTopup())})
		return
	}

	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getWeb3PayMoney(float64(req.Amount), group)
	if payMoney < 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	tradeNo := fmt.Sprintf("W3P%dNO%s%d", id, common.GetRandomString(6), time.Now().Unix())
	topUp := &model.TopUp{
		UserId:          id,
		Amount:          normalizeWeb3PayTopupAmount(req.Amount),
		Money:           payMoney,
		TradeNo:         tradeNo,
		PaymentMethod:   model.PaymentMethodWeb3Pay,
		PaymentProvider: model.PaymentProviderWeb3Pay,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Web3 Pay 创建充值订单失败 user_id=%d trade_no=%s amount=%d error=%q", id, tradeNo, req.Amount, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}

	order, err := createWeb3PayOrder(c, tradeNo, payMoney)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Web3 Pay 创建网关订单失败 user_id=%d trade_no=%s amount=%d money=%.2f error=%q", id, tradeNo, req.Amount, payMoney, err.Error()))
		_ = model.UpdatePendingTopUpStatus(tradeNo, model.PaymentProviderWeb3Pay, common.TopUpStatusFailed)
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("Web3 Pay 充值订单创建成功 user_id=%d trade_no=%s order_no=%s amount=%d money=%.2f", id, tradeNo, order.OrderNo, req.Amount, payMoney))
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": order})
}

func GetWeb3PayOrderStatus(c *gin.Context) {
	tradeNo := strings.TrimSpace(c.Param("trade_no"))
	if tradeNo == "" {
		common.ApiErrorMsg(c, "订单号不能为空")
		return
	}

	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil || topUp.UserId != c.GetInt("id") || topUp.PaymentProvider != model.PaymentProviderWeb3Pay {
		common.ApiErrorMsg(c, "订单不存在")
		return
	}

	common.ApiSuccess(c, gin.H{
		"trade_no":      topUp.TradeNo,
		"status":        topUp.Status,
		"complete_time": topUp.CompleteTime,
	})
}

func normalizeWeb3PayTopupAmount(amount int64) int64 {
	if operation_setting.GetQuotaDisplayType() != operation_setting.QuotaDisplayTypeTokens {
		return amount
	}
	dAmount := decimal.NewFromInt(amount)
	dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
	return dAmount.Div(dQuotaPerUnit).IntPart()
}

func createWeb3PayOrder(c *gin.Context, tradeNo string, payMoney float64) (*web3PayOrderData, error) {
	callBackAddress := service.GetCallbackAddress()
	payload := web3PayCreateOrderRequest{
		MerchantOrderNo: tradeNo,
		Amount:          strconv.FormatFloat(payMoney, 'f', 2, 64),
		OrderCurrency:   "CNY",
		NotifyUrl:       callBackAddress + "/api/web3-pay/webhook",
		ReturnUrl:       paymentReturnPath("/console/topup?show_history=true"),
		Attach:          tradeNo,
	}

	params := map[string]string{
		"merchantOrderNo": payload.MerchantOrderNo,
		"amount":          payload.Amount,
		"orderCurrency":   payload.OrderCurrency,
		"notifyUrl":       payload.NotifyUrl,
		"returnUrl":       payload.ReturnUrl,
		"attach":          payload.Attach,
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	body, err := common.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("序列化请求数据失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, web3PayGatewayAPIBase()+web3PayCreateOrderPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Api-Key", setting.Web3PayAppKey)
	httpReq.Header.Set("X-Timestamp", timestamp)
	httpReq.Header.Set("X-Sign", web3PaySign(params, timestamp, setting.Web3PayApiSecret))

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("Web3 Pay API http status %d body_len=%d", resp.StatusCode, len(respBody))
	}

	var apiResp web3PayAPIResponse
	if err := common.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if apiResp.Code != web3PayOrderStatusOK {
		return nil, fmt.Errorf("Web3 Pay API error code=%s message=%s", apiResp.Code, apiResp.Message)
	}
	if apiResp.Data.OrderNo == "" || len(apiResp.Data.PaymentOptions) == 0 {
		return nil, errors.New("Web3 Pay API 响应缺少支付信息")
	}
	normalizeWeb3PayOrderAssetURLs(&apiResp.Data)
	return &apiResp.Data, nil
}

func normalizeWeb3PayOrderAssetURLs(order *web3PayOrderData) {
	for optionIndex := range order.PaymentOptions {
		order.PaymentOptions[optionIndex].Logo = normalizeWeb3PayAssetURL(order.PaymentOptions[optionIndex].Logo)
		for chainIndex := range order.PaymentOptions[optionIndex].Chain {
			order.PaymentOptions[optionIndex].Chain[chainIndex].Logo = normalizeWeb3PayAssetURL(order.PaymentOptions[optionIndex].Chain[chainIndex].Logo)
		}
	}
}

func normalizeWeb3PayAssetURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	if strings.HasPrefix(value, "/") {
		return web3PayGatewayAPIBase() + value
	}
	return value
}

func web3PayGatewayAPIBase() string {
	apiBase := strings.TrimSpace(setting.Web3PayGatewayAPIBase)
	if apiBase == "" {
		return setting.DefaultWeb3PayGatewayAPIBase
	}
	apiBase = strings.TrimRight(apiBase, "/")
	if strings.HasSuffix(apiBase, web3PayCreateOrderPath) {
		apiBase = strings.TrimSuffix(apiBase, web3PayCreateOrderPath)
	}
	if !strings.HasSuffix(apiBase, web3PayGatewayPath) {
		apiBase += web3PayGatewayPath
	}
	return apiBase
}

func Web3PayWebhook(c *gin.Context) {
	if !isWeb3PayTopUpEnabled() {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("Web3 Pay webhook 被拒绝 reason=webhook_disabled path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Web3 Pay webhook 读取请求体失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	var raw map[string]interface{}
	if err := common.Unmarshal(body, &raw); err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("Web3 Pay webhook 解析原始请求失败 path=%q client_ip=%s body_len=%d error=%q", c.Request.RequestURI, c.ClientIP(), len(body), err.Error()))
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	signPayload := web3PayStringMap(raw)
	if !web3PayVerifyCallbackSign(signPayload, setting.Web3PayApiSecret) {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("Web3 Pay webhook 验签失败 path=%q client_ip=%s body_len=%d", c.Request.RequestURI, c.ClientIP(), len(body)))
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var callback web3PayCallback
	if err := common.Unmarshal(body, &callback); err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("Web3 Pay webhook 解析请求失败 path=%q client_ip=%s body_len=%d error=%q", c.Request.RequestURI, c.ClientIP(), len(body), err.Error()))
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if callback.Event != "order.success" {
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("Web3 Pay webhook 忽略事件 event=%s order_id=%s merchant_order_no=%s", callback.Event, callback.OrderID, callback.MerchantOrderNo))
		c.JSON(http.StatusOK, gin.H{"code": "0"})
		return
	}

	tradeNo := callback.MerchantOrderNo
	if tradeNo == "" {
		tradeNo = callback.Attach
	}
	if tradeNo == "" {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("Web3 Pay webhook 缺少商户订单号 order_id=%s", callback.OrderID))
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)
	if err := model.RechargeWeb3Pay(tradeNo, c.ClientIP()); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Web3 Pay 充值处理失败 trade_no=%s order_id=%s client_ip=%s error=%q", tradeNo, callback.OrderID, c.ClientIP(), err.Error()))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("Web3 Pay 充值成功 trade_no=%s order_id=%s pay_currency=%s chain=%s tx_hash=%s", tradeNo, callback.OrderID, callback.PayCurrency, callback.ChainCode, callback.TxHash))
	c.JSON(http.StatusOK, gin.H{"code": "0"})
}

func web3PayStringMap(raw map[string]interface{}) map[string]string {
	result := make(map[string]string, len(raw))
	for key, value := range raw {
		switch v := value.(type) {
		case string:
			result[key] = v
		case float64:
			if v == float64(int64(v)) {
				result[key] = strconv.FormatInt(int64(v), 10)
			} else {
				result[key] = strconv.FormatFloat(v, 'f', -1, 64)
			}
		case bool:
			result[key] = strconv.FormatBool(v)
		case nil:
			result[key] = ""
		default:
			b, err := common.Marshal(v)
			if err == nil {
				result[key] = string(b)
			}
		}
	}
	return result
}
