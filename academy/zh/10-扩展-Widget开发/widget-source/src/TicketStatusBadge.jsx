import { createElement } from 'react';
import { TicketStatusBadgeSample } from "./components/TicketStatusBadgeSample";
import "./ui/TicketStatusBadge.css";

export function TicketStatusBadge({ statusValue }) {
    return createElement(TicketStatusBadgeSample, { statusValue: statusValue });
}

export default TicketStatusBadge;
