import { createElement } from 'react';

const STATUS_COLORS = {
    Draft: { bg: "#9E9E9E", label: "Draft" },
    Open: { bg: "#FF9800", label: "Open" },
    InProgress: { bg: "#2196F3", label: "In Progress" },
    Resolved: { bg: "#4CAF50", label: "Resolved" },
    Closed: { bg: "#607D8B", label: "Closed" }
};

export function TicketStatusBadgeSample({ statusValue }) {
    if (!statusValue || statusValue.status !== "available") {
        return createElement("span", { className: "widget-badge widget-badge--empty" }, "\u00B7\u00B7\u00B7");
    }

    const config = STATUS_COLORS[statusValue.value] || {
        bg: "#9E9E9E",
        label: statusValue.value
    };

    return createElement("span", {
        className: "widget-badge",
        style: {
            backgroundColor: config.bg,
            color: "#fff",
            padding: "4px 8px",
            borderRadius: "4px",
            fontSize: "12px",
            fontWeight: "bold",
            display: "inline-block",
            lineHeight: "1.4"
        }
    }, config.label);
}
