"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getCustomCaption = function (props) {
    return "TicketStatusBadge";
};
exports.getPreview = function (props, isDarkMode) {
    return {
        type: "RowLayout",
        columnSize: "grow",
        children: [{
            type: "Text",
            content: "TicketStatusBadge",
            fontColor: isDarkMode ? "#cba6f7" : "#89b4fa",
        }]
    };
};
