import { createElement } from 'react';

export function TicketStatusBadgeSample({ statusValue: statusValue }) {
    return createElement('div', { className: "widget-ticketstatusbadge" },
        createElement('span', null, 'TicketStatusBadge'),
    );
}
