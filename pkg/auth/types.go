package auth

// LarkConnectorRequest 飞书连接器请求格式
type LarkConnectorRequest struct {
	Action            string                 `json:"action"`
	VerificationToken string                 `json:"verification_token"`
	Params            map[string]interface{} `json:"params"`
}

// LarkConnectorResponse 飞书连接器响应格式
type LarkConnectorResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// TableMeta 表格元数据
type TableMeta struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Fields      []FieldMeta `json:"fields"`
}

// FieldMeta 字段元数据
type FieldMeta struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

// ReadDataRequest 读取数据请求
type ReadDataRequest struct {
	Filter    map[string]interface{} `json:"filter,omitempty"`
	PageSize  int                    `json:"page_size,omitempty"`
	PageToken string                 `json:"page_token,omitempty"`
}

// ReadDataResponse 读取数据响应
type ReadDataResponse struct {
	Records   []Record `json:"records"`
	HasMore   bool     `json:"has_more"`
	PageToken string   `json:"page_token,omitempty"`
}

// Record 数据记录
type Record struct {
	ID     string                 `json:"id"`
	Fields map[string]interface{} `json:"fields"`
}

// SlippageTradeRecord 滑点交易记录
type SlippageTradeRecord struct {
	Time        string  `json:"时间"`
	Login       string  `json:"账户"`
	Symbol      string  `json:"品种"`
	Direction   string  `json:"方向"`
	Lot         float64 `json:"手数"`
	ReqPrice    float64 `json:"请求价格"`
	OldPrice    float64 `json:"旧价格"`
	NewPrice    float64 `json:"新价格"`
	PriceDiff   float64 `json:"价格差"`
	Slip        float64 `json:"滑点"`
	Action      string  `json:"动作"`
	Currency    string  `json:"货币"`
	PL          float64 `json:"盈亏"`
	OrderID     string  `json:"订单号"`
	SlipFromReq float64 `json:"请求滑点"`
	PLFromReq   float64 `json:"请求盈亏"`
	Profile     string  `json:"配置"`
	Spread      float64 `json:"点差"`
	TickCount   float64 `json:"tick数"`
}
