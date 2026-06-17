import classNames from "classnames";

export function TicketStatusBadgeSample({ statusValue }) {
    return (
        <div className={classNames("widget-ticketstatusbadge")}>
            <span>statusValue ?? "TicketStatusBadge"</span>
        </div>
    );
}
