import { TicketStatusBadgeSample } from "./components/TicketStatusBadgeSample";

export function preview({ statusValue }) {
    return <TicketStatusBadgeSample statusValue={statusValue} />;
}

export function getPreviewCss() {
    return require("./ui/TicketStatusBadge.css");
}
