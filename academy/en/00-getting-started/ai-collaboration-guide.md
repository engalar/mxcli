# Module 00: AI Collaboration Guide — Getting Started

## What You Need to Prepare

### 1. Install mxcli

**Windows (PowerShell, recommended):**

```powershell
irm https://github.com/engalar/mxcli/raw/refs/heads/dev/install.ps1 | iex
```

**Windows (Git Bash) / Linux / Mac:**

```bash
curl -fsSL https://github.com/engalar/mxcli/raw/refs/heads/dev/install.sh | bash
```

Verify after installation:

```bash
mxcli --version
```

### 2. Install Claude Code

Go to [claude.ai/code](https://claude.ai/code) to download the desktop app, or install the CLI via npm:

```bash
npm install -g @anthropic-ai/claude-code
claude --version
```

### 3. Prepare a Mendix Project

**Option A: Create a new project (recommended)**

```bash
# Automatically downloads Mendix 11.6.6 and creates a blank project
mxcli new MyHelpdesk --version 11.6.6
cd MyHelpdesk
```

**Option B: Use an existing project**

```bash
# Confirm the project can be opened
mxcli -p MyProject.mpr -c "show structure"
```

### 4. Prepare the mx check Tool

mxcli automatically locates `mx` in the following order, **with no manual configuration required**:

| Platform | Auto-discovery path (by priority) |
|----------|----------------------------------|
| Windows | `C:\Program Files\Mendix\{version}\modeler\mx.exe` (Studio Pro install directory) |
| Windows | `~/.mxcli/mxbuild/{version}/modeler/mx.exe` (mxbuild cache) |
| Linux/Mac | `~/.mxcli/mxbuild/{version}/modeler/mx` |

**If you already have Mendix Studio Pro installed, mx is available — skip this step.**

If Studio Pro is not installed, let mxcli download mxbuild automatically:

```bash
mxcli setup mxbuild -p MyProject.mpr
```

---

## Your First AI Collaboration

Open Claude Code (inside the project directory):

```bash
claude
```

Try entering:

```
Use MDL to create a simple module HD for me, containing a Customer entity with two attributes, Name and Email
```

Claude will generate MDL code. Save the generated code as `test.mdl`, then:

```bash
# Syntax check
mxcli check test.mdl

# Execute against the project
mxcli exec test.mdl -p MyProject.mpr

# Mendix platform validation (mxcli locates mx automatically, no manual path needed)
mxcli docker check -p MyProject.mpr
```

> **Windows users**: If Studio Pro is installed, you can also call it directly:
> ```
> "C:\Program Files\Mendix\11.6.6\modeler\mx.exe" check MyProject.mpr
> ```

**0 errors = success!**

---

## Understanding the Three-Step Validation Loop

```
1. mxcli check file.mdl              → Is the syntax correct?
2. mxcli exec file.mdl -p app.mpr    → Can it be written to the project?
3. mxcli docker check -p app.mpr     → Does the Mendix platform accept it?
   (Windows Studio Pro users can also run directly: mx.exe check app.mpr)
```

Run these three steps every time you finish a piece of MDL. This is the validation rhythm that runs throughout this course.

---

## Reference Implementation

If you get stuck, check `参考实现/hello-world.mdl`.

**Rule: try it yourself first, and only look at the reference implementation when you're truly stuck.**
