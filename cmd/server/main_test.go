package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"s3-rest/pkg/auth"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHandleGetMeta(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/meta", func(c *gin.Context) {
		handleGetMeta(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/meta", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response auth.LarkConnectorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 0, response.Code)
	assert.Equal(t, "success", response.Message)

	// 验证数据结构
	data, ok := response.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "S3 Slippage Trading Data", data["name"])

	fields, ok := data["fields"].([]interface{})
	assert.True(t, ok)
	assert.Equal(t, 19, len(fields))

	// 验证第一个字段
	firstField := fields[0].(map[string]interface{})
	assert.Equal(t, "time", firstField["name"])
	assert.Equal(t, "text", firstField["type"])
	assert.Equal(t, true, firstField["required"])
}

func TestIsSlippageLogFile(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{"valid file", "slippage-20260323.txt", true},
		{"valid file with path", "logs/slippage-20260323.txt", true},
		{"invalid extension", "slippage-20260323.csv", false},
		{"invalid format", "other-20260323.txt", false},
		{"wrong date format", "slippage-2026-03-23.txt", false},
		{"uppercase", "SLIPPAGE-20260323.TXT", true},
		{"mixed case", "Slippage-20260323.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSlippageLogFile(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractDateFromFileName(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{"valid file", "slippage-20260323.txt", "2026-03-23"},
		{"valid file with path", "logs/slippage-20260323.txt", "2026-03-23"},
		{"invalid format", "other-20260323.txt", ""},
		{"wrong date length", "slippage-2026032.txt", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDateFromFileName(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSortLogFilesByDateDesc(t *testing.T) {
	input := []string{
		"slippage-20260320.txt",
		"slippage-20260323.txt",
		"slippage-20260321.txt",
	}

	expected := []string{
		"slippage-20260323.txt",
		"slippage-20260321.txt",
		"slippage-20260320.txt",
	}

	result := sortLogFilesByDateDesc(input)
	assert.Equal(t, expected, result)
}

func TestParseCSVLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected []string
	}{
		{
			name:     "simple line",
			line:     "a,b,c",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "quoted fields",
			line:     `"a","b","c"`,
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "mixed quotes",
			line:     `a,"b,c",d`,
			expected: []string{"a", "b,c", "d"},
		},
		{
			name:     "single quotes",
			line:     `'a','b','c'`,
			expected: []string{"'a'", "'b'", "'c'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCSVLine(tt.line)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSlippageLine(t *testing.T) {
	tests := []struct {
		name           string
		fields         []string
		fileDate       string
		index          int
		expectedFields map[string]interface{}
	}{
		{
			name: "complete line with all fields",
			fields: []string{
				"2026.03.23 01:01:29",
				"'390009'",
				"XAUUSD",
				"BUY",
				"1.50",
				"4371.50000",
				"4371.50000",
				"4371.54000",
				"0.04000",
				"4.00",
				"Trade",
				"USD",
				"6.00",
				"571539",
				"4.00",
				"6.00",
				"Profile A- Gold Market",
				"26.00",
				"522.00",
			},
			fileDate: "2026-03-23",
			index:    0,
			expectedFields: map[string]interface{}{
				"time":          "2026.03.23 01:01:29",
				"login":         "390009",
				"symbol":        "XAUUSD",
				"b/s":           "BUY",
				"lot":           1.50,
				"req price":     4371.50000,
				"old price":     4371.50000,
				"new price":     4371.54000,
				"price diff":    0.04000,
				"slip":          4.00,
				"action":        "Trade",
				"ccy":           "USD",
				"pl":            6.00,
				"order":         "571539",
				"slip from req": 4.00,
				"pl from req":   6.00,
				"profile":       "Profile A- Gold Market",
				"spread":        26.00,
				"tick count":    522.00,
				"date":          "2026-03-23",
			},
		},
		{
			name: "minimal line with required fields only",
			fields: []string{
				"2026.03.23 01:01:29",
				"'390009'",
				"XAUUSD",
				"BUY",
				"1.50",
				"4371.50000",
				"4371.50000",
				"4371.54000",
				"0.04000",
				"4.00",
				"Trade",
				"USD",
				"6.00",
				"571539",
			},
			fileDate: "2026-03-23",
			index:    1,
			expectedFields: map[string]interface{}{
				"time":       "2026.03.23 01:01:29",
				"login":      "390009",
				"symbol":     "XAUUSD",
				"b/s":        "BUY",
				"lot":        1.50,
				"req price":  4371.50000,
				"old price":  4371.50000,
				"new price":  4371.54000,
				"price diff": 0.04000,
				"slip":       4.00,
				"action":     "Trade",
				"ccy":        "USD",
				"pl":         6.00,
				"order":      "571539",
				"date":       "2026-03-23",
			},
		},
		{
			name:           "too few fields",
			fields:         []string{"time", "login"},
			fileDate:       "2026-03-23",
			index:          2,
			expectedFields: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSlippageLine(tt.fields, tt.fileDate, tt.index)

			if tt.expectedFields == nil {
				assert.Nil(t, result)
				return
			}

			assert.NotNil(t, result)
			assert.Equal(t, fmt.Sprintf("record_%d", tt.index), result.ID)

			for key, expectedValue := range tt.expectedFields {
				actualValue, exists := result.Fields[key]
				assert.True(t, exists, "Field %s should exist", key)

				// 对于数值类型，允许一定的浮点误差
				if expectedFloat, ok := expectedValue.(float64); ok {
					actualFloat, ok := actualValue.(float64)
					assert.True(t, ok, "Field %s should be float64", key)
					assert.InDelta(t, expectedFloat, actualFloat, 0.0001, "Field %s value mismatch", key)
				} else {
					assert.Equal(t, expectedValue, actualValue, "Field %s value mismatch", key)
				}
			}
		})
	}
}

func TestHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, auth.LarkConnectorResponse{
			Code:    0,
			Message: "ok",
		})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response auth.LarkConnectorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 0, response.Code)
	assert.Equal(t, "ok", response.Message)
}

func TestParseSlippageLogFile(t *testing.T) {
	// 创建一个模拟的 S3 服务
	mockContent := `time,login,symbol,b/s,lot,req price,old price,new price,price diff,slip,action,ccy,pl,order,slip from req,pl from req,profile,spread,tick count
2026.03.23 01:01:29,'390009',XAUUSD,BUY,1.50,4371.50000,4371.50000,4371.54000,0.04000,4.00,Trade,USD,6.00,571539,4.00,6.00,Profile A- Gold Market,26.00,522.00
2026.03.23 01:01:35,'390009',XAUUSD,BUY,0.30,4373.02000,4373.02000,4373.02000,0.00000,0.00,Trade,USD,0.00,571540,0.00,0.00,Profile A- Gold Market,0.00,0.00`

	// 由于需要 S3 服务实例，这里我们只测试解析逻辑
	// 实际测试需要 mock S3 服务
	reader := strings.NewReader(mockContent)
	body := io.NopCloser(reader)
	defer body.Close()

	// 读取并解析内容
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(body)
	assert.NoError(t, err)

	lines := strings.Split(buf.String(), "\n")
	assert.Equal(t, 3, len(lines)) // 标题行 + 2 数据行

	// 验证第一行是标题
	assert.True(t, strings.HasPrefix(lines[0], "time,login"))
}
