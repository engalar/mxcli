import { createElement } from 'react';
import { TicketStatusBadgeSample } from "./components/TicketStatusBadgeSample";

export function preview() {
    return createElement(TicketStatusBadgeSample, {
        statusValue: { value: "Open", status: "available" }
    });
}

export function getPreviewCss() {
    return require("./ui/TicketStatusBadge.css");
}
