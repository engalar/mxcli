# Module 10: AI Collaboration Guide — Widget Development

## Tool Requirements

| Tool | Version | Purpose |
|------|---------|---------|
| Node.js | 18+ | Run `@mendix/pluggable-widgets-tools` build toolchain |
| mxcli | latest | Scaffold, build, install widgets; use widgets in pages |
| Mendix Studio Pro | 11.x | Import the .mpk, test the Widget |

## Two Learning Paths

### Path A: Use the Widget in a Page (5 minutes)

```bash
# The widget is already built and installed. Just use it in MDL:
PLUGGABLEWIDGET 'com.helpdesk.widget.TicketStatusBadge' wdgStatus (statusValue: Status)
```

### Path B: Develop the Widget from scratch

```bash
# 1. Scaffold from a clean template (not needed if using widget-source/)
mxcli widget new TicketStatusBadge --dir my-widget

# 2. Build, install, and run in one step
cd widget-source
mxcli widget build --install --project MyProject.mpr

# 3. Use it on a page via MDL
mxcli exec 参考实现/use-widget.mdl -p MyProject.mpr
```

### Using a proxy for restricted networks

```bash
mxcli widget build --https-proxy http://192.168.2.35:29758
mxcli widget build --registry http://npm-registry.internal:4873
```

## Collaborating with Claude

```
I have a Mendix Pluggable Widget called TicketStatusBadge,
whose def.json already defines the property statusValue (HD.TicketStatus enumeration).
Help me use MDL's PLUGGABLEWIDGET syntax to add it to the Status column of the
HD.Ticket_Overview ticket list, replacing the original plain-text display.
```

## MDL Syntax for Using the Widget

```mdl
-- Use a custom Widget inside a DataGrid column
column colStatus (attribute: Status, caption: 'Status', ShowContentAs: customContent, ColumnWidth: manual, Size: 140) {
  PLUGGABLEWIDGET 'com.helpdesk.widget.TicketStatusBadge' wdgStatus (statusValue: Status)
}
```
