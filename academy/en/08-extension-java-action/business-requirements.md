# Module 08: Java Action Extension — Business Requirements

## Business Context

The system administrator reports that Helpdesk's current password-change feature uses plaintext comparison — an unacceptable security risk in any production environment.
Passwords must be encrypted (hashed) before being stored and compared.

Mendix's built-in business logic (microflows/nanoflows) cannot directly call an encryption algorithm library; this requires extension through a **Java Action**.

---

## User Stories

- As a system, I want to BCrypt-hash the password the user submits before storing it, so that even if the database is leaked, the passwords remain safe
- As a system, I want to compare the password the user enters against the stored hash, in order to securely verify login
- As a developer, I want to reuse the encryption logic by calling a Java Action from a microflow, instead of rewriting it in multiple places

---

## Business Rules

- Password minimum length 8, must contain digits and letters
- Hash algorithm: BCrypt (cost factor: 12)
- The same password produces a different hash each time (BCrypt includes a random salt)
- Verification: `BCrypt.checkpw(rawPassword, storedHash)` returns true/false

---

## Acceptance Criteria

- [ ] A microflow can call the JA_HashPassword Java Action, passing in the plaintext password, and obtain a hash string
- [ ] A microflow can call the JA_VerifyPassword Java Action, passing in the plaintext password and the hash, and obtain a boolean result
- [ ] All password-related operations go through a Java Action; no plaintext password comparison appears
