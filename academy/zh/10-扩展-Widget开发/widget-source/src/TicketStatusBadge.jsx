import { createElement } from 'react';

// 状态到颜色/标签的映射（与 业务需求.md 颜色规格一致）
const STATUS_COLORS = {
    Draft:      { bg: "#9E9E9E", label: "Draft" },
    Open:       { bg: "#FF9800", label: "Open" },
    InProgress: { bg: "#2196F3", label: "In Progress" },
    Resolved:   { bg: "#4CAF50", label: "Resolved" },
    Closed:     { bg: "#607D8B", label: "Closed" }
};

// Mendix Pluggable Widget 函数组件
// statusValue 是 AttributeValue<EnumerationValue>，由 Mendix 运行时注入
export function TicketStatusBadge({ statusValue }) {
    // 属性未加载或不可用时显示占位符
    if (!statusValue || statusValue.status !== "available") {
        return createElement("span", { style: { color: "#ccc" } }, "—");
    }

    const key = statusValue.value;
    const config = STATUS_COLORS[key] ?? { bg: "#ccc", label: key };

    return createElement("span", {
        style: {
            display:      "inline-flex",
            alignItems:   "center",
            gap:          "6px",
            padding:      "2px 10px",
            borderRadius: "12px",
            background:   config.bg,
            color:        "#fff",
            fontSize:     "12px",
            fontWeight:   600,
            whiteSpace:   "nowrap"
        }
    },
        // 左侧小圆点
        createElement("span", {
            style: {
                width:        "8px",
                height:       "8px",
                borderRadius: "50%",
                background:   "rgba(255,255,255,0.7)"
            }
        }),
        config.label
    );
}

export default TicketStatusBadge;
