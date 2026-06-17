const STATUS_COLOR = {
    Open:        { bg: "#dbeafe", text: "#1d4ed8" },
    InProgress:  { bg: "#fef9c3", text: "#854d0e" },
    Resolved:    { bg: "#dcfce7", text: "#166534" },
    Closed:      { bg: "#f3f4f6", text: "#374151" },
};

export function TicketStatusBadgeSample({ statusValue }) {
    const key     = statusValue?.value ?? "";
    const label   = statusValue?.displayValue ?? key ?? "—";
    const colors  = STATUS_COLOR[key] ?? { bg: "#e5e7eb", text: "#374151" };

    return (
        <span
            className="widget-ticketstatusbadge"
            style={{ background: colors.bg, color: colors.text }}
        >
            {label}
        </span>
    );
}
