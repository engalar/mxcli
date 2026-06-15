import { Component, ReactNode, createElement } from "react";
import { ValueStatus } from "mendix";
import { TicketStatusBadgeContainerProps } from "../typings/TicketStatusBadgeProps";

const STATUS_COLORS: Record<string, { bg: string; label: string }> = {
    Draft:      { bg: "#9E9E9E", label: "Draft" },
    Open:       { bg: "#FF9800", label: "Open" },
    InProgress: { bg: "#2196F3", label: "In Progress" },
    Resolved:   { bg: "#4CAF50", label: "Resolved" },
    Closed:     { bg: "#607D8B", label: "Closed" }
};

export class TicketStatusBadge extends Component<TicketStatusBadgeContainerProps> {
    render(): ReactNode {
        const { statusValue } = this.props;

        if (!statusValue || statusValue.status !== ValueStatus.Available) {
            return <span style={{ color: "#ccc" }}>—</span>;
        }

        const key = statusValue.value as string;
        const config = STATUS_COLORS[key] ?? { bg: "#ccc", label: key };

        return (
            <span
                style={{
                    display:       "inline-flex",
                    alignItems:    "center",
                    gap:           "6px",
                    padding:       "2px 10px",
                    borderRadius:  "12px",
                    background:    config.bg,
                    color:         "#fff",
                    fontSize:      "12px",
                    fontWeight:    600,
                    whiteSpace:    "nowrap"
                }}
            >
                <span
                    style={{
                        width:        "8px",
                        height:       "8px",
                        borderRadius: "50%",
                        background:   "rgba(255,255,255,0.7)"
                    }}
                />
                {config.label}
            </span>
        );
    }
}
