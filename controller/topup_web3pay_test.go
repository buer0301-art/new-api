package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
)

func TestWeb3PaySignatureCanonicalizesSortedNonEmptyParams(t *testing.T) {
	params := map[string]string{
		"orderCurrency":   "CNY",
		"notifyUrl":       "https://merchant.example.com/pay/notify",
		"amount":          "100.00",
		"merchantOrderNo": "CNY_001",
		"returnUrl":       "",
	}

	canonical := web3PayCanonicalString(params, "1776400000")

	if canonical != "amount=100.00&merchantOrderNo=CNY_001&notifyUrl=https://merchant.example.com/pay/notify&orderCurrency=CNY&timestamp=1776400000" {
		t.Fatalf("canonical mismatch: %s", canonical)
	}
	if got := web3PaySign(params, "1776400000", "sk_test_123456"); got != "1c3b7d9febe6e656fa91eae5001d09ef" {
		t.Fatalf("sign mismatch: %s", got)
	}
}

func TestWeb3PayVerifyCallbackSignIgnoresSignAndEmptyValues(t *testing.T) {
	payload := map[string]string{
		"event":           "order.success",
		"orderId":         "397118246400032768",
		"merchantOrderNo": "CNY_001",
		"amount":          "13.89",
		"payCurrency":     "USDT",
		"txHash":          "",
		"timestamp":       "1776400000",
	}
	payload["sign"] = web3PaySignCallback(payload, "sk_test_123456")

	if !web3PayVerifyCallbackSign(payload, "sk_test_123456") {
		t.Fatal("expected callback signature to verify")
	}

	payload["amount"] = "13.88"
	if web3PayVerifyCallbackSign(payload, "sk_test_123456") {
		t.Fatal("expected tampered callback signature to fail")
	}
}

func TestWeb3PayGatewayAPIBaseUsesConfiguredValueAndDefaults(t *testing.T) {
	original := setting.Web3PayGatewayAPIBase
	t.Cleanup(func() {
		setting.Web3PayGatewayAPIBase = original
	})

	setting.Web3PayGatewayAPIBase = " https://pay.example.com/ "
	if got := web3PayGatewayAPIBase(); got != "https://pay.example.com/api/gateway/v1" {
		t.Fatalf("configured gateway base mismatch: %s", got)
	}

	setting.Web3PayGatewayAPIBase = " https://pay.example.com/api/gateway/v1/orders "
	if got := web3PayGatewayAPIBase(); got != "https://pay.example.com/api/gateway/v1" {
		t.Fatalf("configured order endpoint mismatch: %s", got)
	}

	setting.Web3PayGatewayAPIBase = ""
	if got := web3PayGatewayAPIBase(); got != setting.DefaultWeb3PayGatewayAPIBase {
		t.Fatalf("default gateway base mismatch: %s", got)
	}
}

func TestCreateWeb3PayOrderSendsCNYOrderCurrency(t *testing.T) {
	originalBase := setting.Web3PayGatewayAPIBase
	originalAppKey := setting.Web3PayAppKey
	originalSecret := setting.Web3PayApiSecret
	t.Cleanup(func() {
		setting.Web3PayGatewayAPIBase = originalBase
		setting.Web3PayAppKey = originalAppKey
		setting.Web3PayApiSecret = originalSecret
	})

	setting.Web3PayAppKey = "pk_test"
	setting.Web3PayApiSecret = "sk_test"

	var requestPayload web3PayCreateOrderRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := common.DecodeJson(r.Body, &requestPayload); err != nil {
			t.Fatalf("decode request payload failed: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"0","data":{"orderNo":"397118246400032768","merchantOrderNo":"W3P1NOABC1776400000","orderAmount":"10.00","orderCurrency":"CNY","payAmount":"1.39","payCurrency":"USDT","paymentOptions":[{"code":"USDT","chain":[{"address":"0xabc","chainCode":"BSC","chainName":"BSC"}]}]}}`))
	}))
	defer server.Close()
	setting.Web3PayGatewayAPIBase = server.URL

	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/api/user/web3-pay/pay", nil)

	if _, err := createWeb3PayOrder(context, "W3P1NOABC1776400000", 10); err != nil {
		t.Fatalf("create order failed: %v", err)
	}

	if requestPayload.OrderCurrency != "CNY" {
		t.Fatalf("order currency mismatch: %s", requestPayload.OrderCurrency)
	}
	if requestPayload.Amount != "10.00" {
		t.Fatalf("amount mismatch: %s", requestPayload.Amount)
	}
}
