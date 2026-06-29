package helper

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func ApplyDynamicModelMapping(c *gin.Context, info *relaycommon.RelayInfo, request dto.Request) (bool, error) {
	if info == nil {
		return false, nil
	}
	info.DynamicModelMappingApplied = false
	info.DynamicFieldTransforms = nil
	if info.ChannelMeta == nil || len(info.ChannelOtherSettings.DynamicModelMapping) == 0 {
		return false, nil
	}

	var body []byte
	var bodyLoaded bool
	for _, rule := range info.ChannelOtherSettings.DynamicModelMapping {
		if strings.TrimSpace(rule.From) != "" && strings.TrimSpace(rule.From) != info.OriginModelName {
			continue
		}
		if len(rule.When) > 0 || len(rule.FieldTransforms) > 0 {
			if !bodyLoaded {
				loadedBody, err := dynamicModelMappingBody(c)
				if err != nil {
					return false, err
				}
				body = loadedBody
				bodyLoaded = true
			}
		}
		if !dynamicModelMappingConditionsMatch(body, rule.When) {
			continue
		}

		upstreamModel := strings.TrimSpace(rule.To)
		if upstreamModel == "" {
			upstreamModel = info.OriginModelName
		}
		info.UpstreamModelName = upstreamModel
		info.IsModelMapped = upstreamModel != "" && upstreamModel != info.OriginModelName
		info.DynamicModelMappingApplied = true
		info.DynamicFieldTransforms = rule.FieldTransforms
		if request != nil {
			request.SetModelName(upstreamModel)
		}

		return true, nil
	}

	return false, nil
}

func dynamicModelMappingBody(c *gin.Context) ([]byte, error) {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, err
	}
	body, err := storage.Bytes()
	if err != nil {
		return nil, err
	}
	return body, nil
}

func dynamicModelMappingConditionsMatch(body []byte, conditions []dto.DynamicModelMappingCondition) bool {
	for _, condition := range conditions {
		if !dynamicModelMappingConditionMatch(body, condition) {
			return false
		}
	}
	return true
}

func dynamicModelMappingConditionMatch(body []byte, condition dto.DynamicModelMappingCondition) bool {
	path := strings.TrimSpace(condition.Path)
	op := strings.TrimSpace(condition.Op)
	if op == "" {
		op = "eq"
	}
	value := gjson.GetBytes(body, path)

	switch op {
	case "exists":
		return value.Exists()
	case "not_empty":
		return value.Exists() && dynamicValueLength(value) > 0
	case "eq":
		return dynamicValueEqual(value, condition.Value)
	case "neq":
		return !dynamicValueEqual(value, condition.Value)
	case "len_eq":
		return dynamicValueLength(value) == numericConditionValue(condition.Value)
	case "len_gte":
		return dynamicValueLength(value) >= numericConditionValue(condition.Value)
	case "len_lte":
		return dynamicValueLength(value) <= numericConditionValue(condition.Value)
	default:
		return false
	}
}

func dynamicValueEqual(value gjson.Result, expected any) bool {
	if !value.Exists() {
		return expected == nil
	}
	switch typed := expected.(type) {
	case string:
		return value.String() == typed
	case bool:
		return value.Bool() == typed
	case int:
		return value.Int() == int64(typed)
	case int64:
		return value.Int() == typed
	case float64:
		return value.Float() == typed
	default:
		return fmt.Sprint(value.Value()) == fmt.Sprint(expected)
	}
}

func dynamicValueLength(value gjson.Result) int {
	if !value.Exists() {
		return 0
	}
	if value.IsArray() {
		return len(value.Array())
	}
	if value.IsObject() {
		return len(value.Map())
	}
	if strings.TrimSpace(value.String()) == "" {
		return 0
	}
	return 1
}

func numericConditionValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(typed))
		return n
	default:
		return 0
	}
}

func applyDynamicFieldTransforms(body []byte, transforms []dto.DynamicFieldTransform) ([]byte, error) {
	nextBody := string(body)
	var err error
	for _, transform := range transforms {
		path := strings.TrimSpace(transform.Path)
		to := strings.TrimSpace(transform.To)
		if path == "" || to == "" {
			continue
		}
		if to == "delete" {
			nextBody, err = sjson.Delete(nextBody, path)
			if err != nil {
				return nil, err
			}
			continue
		}

		value := gjson.Get(nextBody, path)
		if !value.Exists() {
			continue
		}
		converted, ok := convertDynamicFieldValue(value, to)
		if !ok {
			return nil, fmt.Errorf("unsupported dynamic field transform type: %s", to)
		}
		nextBody, err = sjson.Set(nextBody, path, converted)
		if err != nil {
			return nil, err
		}
	}
	return []byte(nextBody), nil
}

func convertDynamicFieldValue(value gjson.Result, to string) (any, bool) {
	switch to {
	case "string":
		if value.IsArray() {
			for _, item := range value.Array() {
				if strings.TrimSpace(item.String()) != "" {
					return item.String(), true
				}
			}
			return "", true
		}
		return value.String(), true
	case "array":
		if value.IsArray() {
			return value.Value(), true
		}
		return []any{value.Value()}, true
	case "integer":
		return int64(math.Round(value.Float())), true
	case "number":
		return value.Float(), true
	case "boolean":
		return value.Bool(), true
	default:
		return nil, false
	}
}

func replaceRequestBody(c *gin.Context, body []byte) error {
	storage, err := common.CreateBodyStorage(body)
	if err != nil {
		return err
	}
	if oldStorage, exists := c.Get(common.KeyBodyStorage); exists && oldStorage != nil {
		if closer, ok := oldStorage.(io.Closer); ok {
			_ = closer.Close()
		}
	}
	c.Set(common.KeyBodyStorage, storage)
	c.Set(common.KeyRequestBody, body)
	c.Request.Body = io.NopCloser(storage)
	c.Request.ContentLength = int64(len(body))
	c.Request.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	if c.Request.Header.Get("Content-Type") == "" {
		c.Request.Header.Set("Content-Type", "application/json")
	}
	c.Request.Header.Set("Content-Length", strconv.Itoa(len(body)))
	return nil
}

func ApplyDynamicFieldTransformsToJSON(jsonData []byte, info *relaycommon.RelayInfo) ([]byte, error) {
	if info == nil || len(info.DynamicFieldTransforms) == 0 {
		return jsonData, nil
	}
	return applyDynamicFieldTransforms(jsonData, info.DynamicFieldTransforms)
}

func ApplyDynamicFieldTransformsToRequestBody(c *gin.Context, info *relaycommon.RelayInfo) error {
	if info == nil || len(info.DynamicFieldTransforms) == 0 {
		return nil
	}
	if !strings.HasPrefix(c.GetHeader("Content-Type"), "application/json") {
		return nil
	}
	body, err := dynamicModelMappingBody(c)
	if err != nil {
		return err
	}
	nextBody, err := applyDynamicFieldTransforms(body, info.DynamicFieldTransforms)
	if err != nil {
		return err
	}
	if bytes.Equal(nextBody, body) {
		return nil
	}
	return replaceRequestBody(c, nextBody)
}
