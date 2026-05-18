// SPDX-License-Identifier: Apache-2.0

package daemon

// ValidateRequest 是客户端发往 daemon 的验证请求。
type ValidateRequest struct {
	MprPath  string `json:"mprPath"`
	Filter   string `json:"filter,omitempty"`
	Severity string `json:"severity,omitempty"`
}

// ValidationItem 是单条验证结果。
type ValidationItem struct {
	UnitID   string `json:"unitID"`
	UnitType string `json:"unitType"`
	Field    string `json:"field"`
	Raw      string `json:"raw"`
	RuleID   string `json:"ruleID"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Fix      string `json:"fix,omitempty"`
}

// ValidateResponse 是 daemon 返回的验证响应。
type ValidateResponse struct {
	IndexAge string           `json:"indexAge"`
	Results  []ValidationItem `json:"results"`
	Error    string           `json:"error,omitempty"`
}

// PingRequest 用于检活。
type PingRequest struct{}

// PingResponse 返回 daemon 状态。
type PingResponse struct {
	OK          bool   `json:"ok"`
	MprPath     string `json:"mprPath"`
	IndexAge    string `json:"indexAge"`
	EntityCount int    `json:"entityCount"`
	EnumCount   int    `json:"enumCount"`
}
