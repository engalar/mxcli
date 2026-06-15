# Module 12: AI Collaboration Patterns

## Purpose of This Module

This is not about teaching you which command to use, but about teaching you **how to think**: when you receive a requirement, how do you collaborate efficiently with AI, going through the full loop from design to implementation to validation without wasting time?

---

## Core Principles

### 1. Requirements first, code second

Have the AI help you clarify requirements first, then have it write code.
Skipping design and asking for code directly often leads to many revisions.

**A good workflow:**
```
You: Read requirements.md — what do you understand the core business rules of this feature to be?
AI: [explains the business logic]
You: Correct. Now implement this microflow in MDL.
AI: [generates the MDL]
You: Execute it, check for errors, come back and fix.
```

**A bad workflow:**
```
You: Write me a ticket microflow
AI: [guesses and generates, usually missing the status check]
You: No, you need to add status validation
AI: [revises, then misses something else]
...(round and round)
```

### 2. Validation-driven: run check at every step

Don't wait until all the MDL is written to validate — every time you finish a microflow or page, run:

```bash
mxcli check my-file.mdl
mxcli exec  my-file.mdl -p app.mpr
```

The earlier you catch an error, the lower the cost to fix it.

### 3. Error messages are part of the conversation

When mxcli or mx check reports an error, paste the error message directly to Claude:

```
After execution it reported this error:
[paste the error message]
Help me analyze the cause and fix it
```

Don't guess on your own — AI analyzes error messages much faster than most people.

### 4. Divide and conquer: do one thing at a time

Don't have the AI generate 500 lines of code at once — first have it do the domain model, validate, then do the microflows, then the pages.

Commit after each stage is complete, so you can roll back if something goes wrong.

---

## Five Typical Scenarios

| Scenario | Recommended Prompt Template |
|----------|----------------------------|
| Create an entity | `prompt-create-entity.md` |
| Debug an mxcli error | `prompt-debug-errors.md` |
| Review generated MDL | `prompt-review-code.md` |
| Design a new feature from scratch | `案例分析/从需求到实现-完整案例.md` |

---

## Common Pitfalls and How to Avoid Them

| Pitfall | Symptom | How to Avoid |
|---------|---------|--------------|
| Asking the AI to do too much at once | The generated code has many small errors | Step by step: entity first, then microflows, then pages |
| Not providing context | The generated code doesn't match the existing model | Run `show structure` or `describe entity` first |
| Skipping validation | By the time you find an error, you've already written many lines | Run mxcli check at every step |
| Accepting the AI's first draft | Missing edge-case handling | Ask "which edge cases does this implementation not handle?" |
