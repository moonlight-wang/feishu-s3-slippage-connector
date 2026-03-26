package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"s3-rest/pkg/auth"
	"s3-rest/pkg/config"
	"s3-rest/pkg/s3"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	s3Service, err := s3.NewS3Service(
		cfg.S3Endpoint,
		cfg.S3Region,
		cfg.S3Bucket,
		cfg.S3AccessKey,
		cfg.S3SecretKey,
	)
	if err != nil {
		log.Fatalf("Failed to initialize S3 service: %v", err)
	}

	larkAuth := auth.NewLarkAuth(cfg.LarkVerificationToken)

	r := gin.Default()

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, auth.LarkConnectorResponse{
			Code:    0,
			Message: "ok",
		})
	})

	// 飞书连接器主接口
	r.POST("/connector", auth.LarkConnectorMiddleware(larkAuth), func(c *gin.Context) {
		larkReq, exists := c.Get("lark_request")
		if !exists {
			c.JSON(http.StatusOK, auth.LarkConnectorResponse{
				Code:    400,
				Message: "Request not found",
			})
			return
		}

		req := larkReq.(auth.LarkConnectorRequest)

		switch req.Action {
		case "get_meta":
			handleGetMeta(c)
		case "read_data":
			handleReadData(c, s3Service, req.Params)
		default:
			c.JSON(http.StatusOK, auth.LarkConnectorResponse{
				Code:    400,
				Message: "Unknown action: " + req.Action,
			})
		}
	})

	addr := fmt.Sprintf(":%d", cfg.ServerPort)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// handleGetMeta 获取表格元数据
func handleGetMeta(c *gin.Context) {
	meta := auth.TableMeta{
		Name:        "S3 Slippage Trading Data",
		Description: "Slippage trading records synchronized from S3 storage",
		Fields: []auth.FieldMeta{
			{Name: "time", Type: "text", Required: true},
			{Name: "login", Type: "text", Required: true},
			{Name: "symbol", Type: "text", Required: true},
			{Name: "b/s", Type: "text", Required: true},
			{Name: "lot", Type: "number", Required: true},
			{Name: "req price", Type: "number", Required: true},
			{Name: "old price", Type: "number", Required: false},
			{Name: "new price", Type: "number", Required: true},
			{Name: "price diff", Type: "number", Required: false},
			{Name: "slip", Type: "number", Required: false},
			{Name: "action", Type: "text", Required: false},
			{Name: "ccy", Type: "text", Required: false},
			{Name: "pl", Type: "number", Required: false},
			{Name: "order", Type: "text", Required: false},
			{Name: "slip from req", Type: "number", Required: false},
			{Name: "pl from req", Type: "number", Required: false},
			{Name: "profile", Type: "text", Required: false},
			{Name: "spread", Type: "number", Required: false},
			{Name: "tick count", Type: "number", Required: false},
		},
	}

	c.JSON(http.StatusOK, auth.LarkConnectorResponse{
		Code:    0,
		Message: "success",
		Data:    meta,
	})
}

// handleReadData 读取数据
func handleReadData(c *gin.Context, s3Service *s3.S3Service, params map[string]interface{}) {
	// 解析分页参数
	pageSize := 100
	if ps, ok := params["page_size"].(float64); ok {
		pageSize = int(ps)
	}
	if pageSize <= 0 || pageSize > 500 {
		pageSize = 100
	}

	pageToken := ""
	if pt, ok := params["page_token"].(string); ok {
		pageToken = pt
	}

	// 解析过滤条件
	dateFilter := ""
	if filter, ok := params["filter"].(map[string]interface{}); ok {
		if d, ok := filter["date"].(string); ok {
			dateFilter = d
		}
	}

	// 列出 S3 对象，只获取 slippage-YYYYMMDD.txt 格式的文件
	objects, err := s3Service.ListObjects("")
	if err != nil {
		c.JSON(http.StatusOK, auth.LarkConnectorResponse{
			Code:    500,
			Message: "Failed to list objects: " + err.Error(),
		})
		return
	}

	// 过滤滑点日志文件
	var logFiles []string
	for _, obj := range objects {
		if isSlippageLogFile(obj) {
			// 如果有过滤条件，检查日期是否匹配
			if dateFilter != "" {
				date := extractDateFromFileName(obj)
				if strings.Contains(date, dateFilter) {
					logFiles = append(logFiles, obj)
				}
			} else {
				logFiles = append(logFiles, obj)
			}
		}
	}

	// 按日期倒序排序（最新的在前面）
	logFiles = sortLogFilesByDateDesc(logFiles)

	// 读取所有文件的交易记录
	var allRecords []auth.Record
	recordIndex := 0

	for _, fileKey := range logFiles {
		fileDate := extractDateFromFileName(fileKey)

		records := parseSlippageLogFile(s3Service, fileKey, fileDate, &recordIndex)
		allRecords = append(allRecords, records...)
	}

	// 分页处理
	startIndex := 0
	if pageToken != "" {
		startIndex, _ = strconv.Atoi(pageToken)
	}

	totalRecords := len(allRecords)
	endIndex := startIndex + pageSize
	hasMore := false
	if endIndex < totalRecords {
		hasMore = true
	} else {
		endIndex = totalRecords
	}

	// 获取当前页的记录
	var pageRecords []auth.Record
	if startIndex < totalRecords {
		pageRecords = allRecords[startIndex:endIndex]
	}

	// 构建下一页 token
	nextPageToken := ""
	if hasMore {
		nextPageToken = strconv.Itoa(endIndex)
	}

	response := auth.ReadDataResponse{
		Records:   pageRecords,
		HasMore:   hasMore,
		PageToken: nextPageToken,
	}

	c.JSON(http.StatusOK, auth.LarkConnectorResponse{
		Code:    0,
		Message: "success",
		Data:    response,
	})
}

// isSlippageLogFile 判断是否为滑点日志文件
func isSlippageLogFile(key string) bool {
	// 匹配 slippage-YYYYMMDD.txt 格式
	pattern := `slippage-\d{8}\.txt$`
	matched, _ := regexp.MatchString(pattern, strings.ToLower(key))
	return matched
}

// extractDateFromFileName 从文件名提取日期
func extractDateFromFileName(key string) string {
	// 提取 YYYYMMDD 部分
	re := regexp.MustCompile(`slippage-(\d{8})\.txt`)
	matches := re.FindStringSubmatch(strings.ToLower(key))
	if len(matches) >= 2 {
		dateStr := matches[1]
		// 格式化为 YYYY-MM-DD
		if len(dateStr) == 8 {
			return dateStr[:4] + "-" + dateStr[4:6] + "-" + dateStr[6:]
		}
	}
	return ""
}

// sortLogFilesByDateDesc 按日期倒序排序日志文件
func sortLogFilesByDateDesc(files []string) []string {
	// 简单的冒泡排序，按文件名中的日期排序
	result := make([]string, len(files))
	copy(result, files)

	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			dateI := extractDateFromFileName(result[i])
			dateJ := extractDateFromFileName(result[j])
			// 倒序排列，新的日期在前
			if dateI < dateJ {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}

// parseSlippageLogFile 解析滑点日志文件
func parseSlippageLogFile(s3Service *s3.S3Service, key, fileDate string, recordIndex *int) []auth.Record {
	var records []auth.Record

	body, err := s3Service.GetObject(key)
	if err != nil {
		return records
	}
	defer body.Close()

	// 使用 CSV 解析器读取文件
	reader := csv.NewReader(body)
	reader.FieldsPerRecord = -1 // 允许可变字段数
	reader.TrimLeadingSpace = true

	lines, err := reader.ReadAll()
	if err != nil {
		// 如果 CSV 解析失败，尝试按行解析
		return parseSlippageLogFileFallback(s3Service, key, fileDate, recordIndex)
	}

	// 跳过标题行（第一行）
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if len(line) < 10 {
			continue
		}

		record := parseSlippageLine(line, fileDate, *recordIndex)
		if record != nil {
			records = append(records, *record)
			*recordIndex++
		}
	}

	return records
}

// parseSlippageLogFileFallback 备用解析方法
func parseSlippageLogFileFallback(s3Service *s3.S3Service, key, fileDate string, recordIndex *int) []auth.Record {
	var records []auth.Record

	body, err := s3Service.GetObject(key)
	if err != nil {
		return records
	}
	defer body.Close()

	scanner := bufio.NewScanner(body)
	isFirstLine := true

	for scanner.Scan() {
		line := scanner.Text()

		// 跳过标题行
		if isFirstLine {
			isFirstLine = false
			continue
		}

		// 跳过空行
		if strings.TrimSpace(line) == "" {
			continue
		}

		// 解析 CSV 行
		fields := parseCSVLine(line)
		if len(fields) < 10 {
			continue
		}

		record := parseSlippageLine(fields, fileDate, *recordIndex)
		if record != nil {
			records = append(records, *record)
			*recordIndex++
		}
	}

	return records
}

// parseCSVLine 解析 CSV 行（处理引号内的逗号）
func parseCSVLine(line string) []string {
	var fields []string
	var current strings.Builder
	inQuotes := false

	for i, r := range line {
		switch r {
		case '"':
			inQuotes = !inQuotes
		case ',':
			if !inQuotes {
				fields = append(fields, strings.TrimSpace(current.String()))
				current.Reset()
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}

		// 处理最后一个字符
		if i == len(line)-1 {
			fields = append(fields, strings.TrimSpace(current.String()))
		}
	}

	return fields
}

// parseSlippageLine 解析单行滑点数据
func parseSlippageLine(fields []string, fileDate string, index int) *auth.Record {
	if len(fields) < 10 {
		return nil
	}

	// 清理字段值（去除引号）
	cleanField := func(s string) string {
		s = strings.TrimSpace(s)
		s = strings.Trim(s, "'")
		s = strings.Trim(s, `"`)
		return s
	}

	parseFloat := func(s string) float64 {
		s = cleanField(s)
		v, _ := strconv.ParseFloat(s, 64)
		return v
	}

	record := auth.Record{
		ID: fmt.Sprintf("record_%d", index),
		Fields: map[string]interface{}{
			"time":          cleanField(fields[0]),
			"login":         cleanField(fields[1]),
			"symbol":        cleanField(fields[2]),
			"b/s":           cleanField(fields[3]),
			"lot":           parseFloat(fields[4]),
			"req price":     parseFloat(fields[5]),
			"old price":     parseFloat(fields[6]),
			"new price":     parseFloat(fields[7]),
			"price diff":    parseFloat(fields[8]),
			"slip":          parseFloat(fields[9]),
			"action":        cleanField(fields[10]),
			"ccy":           cleanField(fields[11]),
			"pl":            parseFloat(fields[12]),
			"order":         cleanField(fields[13]),
			"date":          fileDate,
			"sync_time":     time.Now().Format("2006-01-02 15:04:05"),
		},
	}

	// 可选字段（根据行长度判断）
	if len(fields) > 14 {
		record.Fields["slip from req"] = parseFloat(fields[14])
	}
	if len(fields) > 15 {
		record.Fields["pl from req"] = parseFloat(fields[15])
	}
	if len(fields) > 16 {
		record.Fields["profile"] = cleanField(fields[16])
	}
	if len(fields) > 17 {
		record.Fields["spread"] = parseFloat(fields[17])
	}
	if len(fields) > 18 {
		record.Fields["tick count"] = parseFloat(fields[18])
	}

	return &record
}
