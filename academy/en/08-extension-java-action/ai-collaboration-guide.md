# Module 08: AI Collaboration Guide — Java Action Extension

## Tool Requirements

| Tool | Purpose | Minimum Version |
|------|---------|-----------------|
| Mendix Studio Pro | Create the Java Action skeleton (generates the `.java` file) | 11.x |
| JDK | Compile the Java code | 11+ |
| mxcli | The microflow (MDL) that calls the Java Action | latest |

> **Fast path (no Studio Pro):**
> Use the source provided in `java-source/JA_HashPassword.java`. After creating a Java Action with the same name in Studio Pro, paste the source into it. Or directly use the `CommunityCommons` module from the Marketplace, which already includes a `BCrypt` utility class.

## Two Paths

### Path A: Use CommunityCommons (Fastest, Recommended for Demos)

1. Install the CommunityCommons Marketplace module in Studio Pro
2. Call `CommunityCommons.BCryptHash` and `CommunityCommons.BCryptCheck` in a microflow
3. See `reference-implementation/call-java-action.mdl` for the corresponding MDL (how to call)

### Path B: Implement the Java Action Yourself (Full Learning Path)

1. Studio Pro → App Explorer → right-click module HD → Add Java Action
2. Name it `JA_HashPassword`, add a parameter `Password` (String), return type `String`
3. Open `javasource/helpdesk/actions/JA_HashPassword.java`
4. Paste the content of `java-source/JA_HashPassword.java` into it
5. Add the `bcrypt.jar` dependency in Studio Pro (or use Maven)
6. Use mxcli exec to run `reference-implementation/call-java-action.mdl` to create the calling microflow

## Collaborating with Claude

```
Help me implement a microflow HD.ACT_ChangePassword in MDL
that calls the Java Action JA_HashPassword (in the module, parameter: Password: string, returns string),
then stores the hash result into the current user's PasswordHash attribute.

Also implement HD.ACT_VerifyPassword,
calling JA_VerifyPassword (parameters: Password: string, HashedPassword: string, returns boolean)
```

## Key MDL Syntax

```mdl
-- Syntax for calling a Java Action
declare $Hash: string = call java action HD.JA_HashPassword (
  Password = $Password
);
```

## Common Pitfalls

| Pitfall | Solution |
|---------|----------|
| `call java action` cannot find the Action | The Java Action must be created in Studio Pro before mxcli can reference it |
| Return type mismatch | Confirm the Java Action's return type in Studio Pro matches the MDL declaration |
| BCrypt jar missing | Place the bcrypt jar in Studio Pro's `userlib/` directory, or use Maven |
