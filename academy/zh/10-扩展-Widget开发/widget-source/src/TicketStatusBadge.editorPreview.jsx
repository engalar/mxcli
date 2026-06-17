import { createElement } from 'react';
import { TicketStatusBadgeSample } from "./components/TicketStatusBadgeSample";

export function preview({ placeholder }) {
    return createElement(TicketStatusBadgeSample, { sampleText: placeholder });
}

export function getPreviewCss() {
    return require("./ui/TicketStatusBadge.css");
}
