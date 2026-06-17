import { TicketStatusBadgeSample } from "./components/TicketStatusBadgeSample";

export function preview() {
    return <TicketStatusBadgeSample statusValue={"Open"} />;
}

export function getPreviewCss() {
    return require("./ui/TicketStatusBadge.css");
}
