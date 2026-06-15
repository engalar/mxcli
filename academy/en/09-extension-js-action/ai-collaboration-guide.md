# Module 09: AI Collaboration Guide — JS Action Extension

## JS Action vs Java Action

| Feature | JS Action | Java Action |
|---------|-----------|-------------|
| Runtime environment | Client (browser) | Server |
| MDL syntax | `call javascript action` | `call java action` |
| Suitable scenarios | Browser APIs, UI interaction | Server-side computation, database operations |
| How to create | Studio Pro or inline MDL | Studio Pro only |

## Collaborating with Claude

```
Help me implement three JavaScript Action nanoflows in MDL:

1. HD.NF_CopyToClipboard($Text: string):
   Use navigator.clipboard.writeText() to write $Text to the clipboard
   On success, call mx.ui.info('Copied') to show a message

2. HD.NF_NotifyHighPriority($Subject: string):
   Check Notification.permission; if it is granted, create a notification
   Title: 'High Priority Ticket', body: $Subject

3. HD.NF_FormatRelativeTime($DateTime: DateTime) returns string:
   Compute the difference (in milliseconds) of now - datetime, and convert it to "just now / N minutes ago / N hours ago / N days ago"
```

## How to Write a JS Action in MDL

```mdl
-- Inline JavaScript Action (Mendix 10.6+ supports inline JS in nanoflows)
create or modify nanoflow HD.NF_CopyToClipboard
  ($Text: string)
  returns void
  folder 'UI'
{
  call javascript action (
    code = 'navigator.clipboard.writeText($Text).then(() => mx.ui.info("已复制"));',
    parameters = { $Text: string }
  );
}
/
```

## Common Pitfalls

| Pitfall | Solution |
|---------|----------|
| The clipboard API requires HTTPS | Local development on localhost is auto-authorized; production must use HTTPS |
| Notification permission not granted | Call `Notification.requestPermission()` first, then check permission |
| Passing DateTime into JS | A Mendix DateTime is a JS Date object; call `.getTime()` directly to get milliseconds |
