# Module 10: AI Collaboration Guide — Widget Development

## Tool Requirements

| Tool | Version | Purpose |
|------|---------|---------|
| Node.js | 18+ | Run the Widget toolchain |
| pluggable-widgets-tools | latest | Compile and package the Widget |
| Mendix Studio Pro | 11.x | Import the .mpk, test the Widget |

## Two Learning Paths

### Path A: Use the Widget only (5 minutes)

1. Drag the prebuilt `TicketStatusBadge.mpk` (the build artifact provided with this module) into Studio Pro
2. Run `参考实现/use-widget.mdl` to add the Widget to a page
3. Start the app and check the result

### Path B: Develop the Widget from scratch (full path)

```bash
# 1. Clone the Widget template
npx @mendix/pluggable-widgets-tools@latest create-widget TicketStatusBadge

# 2. Replace the source (copy from widget-source/src/)
cp widget-source/src/TicketStatusBadge.tsx src/TicketStatusBadge.tsx

# 3. Compile + package
npm run build
# Output: dist/TicketStatusBadge.mpk

# 4. Import the .mpk in Studio Pro
# App → Import module package → select the .mpk

# 5. Use it on a page via MDL
mxcli exec 参考实现/use-widget.mdl -p MyProject.mpr
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
column colStatus (caption: 'Status', ColumnWidth: manual, Size: 120, ShowContentAs: customContent) {
  PLUGGABLEWIDGET 'helpdesk.TicketStatusBadge' wdgStatus (
    statusValue: attribute Status
  )
}
```
