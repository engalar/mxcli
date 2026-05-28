// SPDX-License-Identifier: Apache-2.0

package daemon

// Request type constants — always set ValidateRequest.Type to one of these.
const (
	ReqPing     = "ping"     // status probe; server replies with PingResponse
	ReqValidate = "validate" // expression validation; server replies with ValidateResponse
)

// ValidateRequest 是客户端发往 daemon 的所有请求的通用信封。
// Type 字段区分请求类型（ReqPing / ReqValidate）；省略 Type 时退化为旧版
// 空 MprPath = ping 约定（兼容旧客户端）。
type ValidateRequest struct {
	Type     string `json:"type,omitempty"`
	MprPath  string `json:"mprPath"`
	Filter   string `json:"filter,omitempty"`
	Severity string `json:"severity,omitempty"`
}

// ValidationItem 是单条验证结果。
type ValidationItem struct {
	UnitID   string `json:"unitID"`
	UnitType string `json:"unitType"`
	UnitPath string `json:"unitPath,omitempty"` // relative path from mprcontents/
	Location string `json:"location,omitempty"` // human-readable "Module.MicroflowName"
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

// PingResponse 返回 daemon 状态。
type PingResponse struct {
	OK          bool   `json:"ok"`
	MprPath     string `json:"mprPath"`
	IndexAge    string `json:"indexAge"`
	EntityCount int    `json:"entityCount"`
	EnumCount   int    `json:"enumCount"`
}
