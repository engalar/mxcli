# Module 03: AI Collaboration Guide — Nanoflows and the Client

## Microflow vs Nanoflow

| Feature | Microflow | Nanoflow |
|---------|-----------|----------|
| Runtime environment | Server | Client (browser) |
| Database access | Supported (full) | Supported (limited) |
| Best for | Complex business logic | Simple operations, fast response |
| MDL syntax | `create or modify microflow` | `create or modify nanoflow` |

## Steps for Collaborating with Claude

```
Help me implement three nanoflows in MDL:
1. HD.NF_Ticket_QuickCreate: parameters $Customer (HD.Customer) and $Subject (string),
   create a draft ticket and commit, return HD.Ticket
2. HD.NF_TicketSearch_Apply: parameter $Search (HD.TicketSearch),
   retrieve tickets from the database (limit 100), return an HD.Ticket list
3. HD.NF_Priority_GetLabel: parameter $Priority (HD.TicketPriority),
   return a string (Critical→'🔴 紧急', High→'🟠 高', Normal→'🟡 普通', Low→'🟢 低')
```

## Common Pitfalls

- Nanoflow `retrieve` does not support complex XPath references; use a simple `where [Status = 'Open']`
- Nanoflows do not support `show message`, only `return`
- When a nanoflow returns a list, the type is written as: `returns list of HD.Ticket as $Tickets`
