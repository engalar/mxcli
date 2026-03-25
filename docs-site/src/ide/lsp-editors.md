# Integration with Other Editors

The MDL language server works with any editor that supports the Language Server Protocol. This page covers setup for editors other than VS Code.

## General Setup

All editors need to:

1. Start `mxcli lsp --stdio` as a subprocess
2. Communicate via JSON-RPC over stdin/stdout
3. Associate the language server with `.mdl` files

## Neovim

### Using nvim-lspconfig

Add the following to your Neovim configuration:

```lua
local lspconfig = require('lspconfig')
local configs = require('lspconfig.configs')

-- Define the MDL language server
if not configs.mdl then
  configs.mdl = {
    default_config = {
      cmd = { 'mxcli', 'lsp', '--stdio' },
      filetypes = { 'mdl' },
      root_dir = function(fname)
        return lspconfig.util.find_git_ancestor(fname)
          or lspconfig.util.root_pattern('*.mpr')(fname)
      end,
      settings = {},
    },
  }
end

lspconfig.mdl.setup({})
```

### File Type Detection

Add MDL file type recognition:

```lua
vim.filetype.add({
  extension = {
    mdl = 'mdl',
  },
})
```

### Syntax Highlighting

For basic syntax highlighting without a full TreeSitter grammar, create `~/.config/nvim/syntax/mdl.vim` with keyword highlights for MDL reserved words, or use the TextMate grammar from the VS Code extension via a plugin like `nvim-textmate`.

## Emacs

### Using lsp-mode

```elisp
(require 'lsp-mode)

(add-to-list 'lsp-language-id-configuration '(mdl-mode . "mdl"))

(lsp-register-client
 (make-lsp-client
  :new-connection (lsp-stdio-connection '("mxcli" "lsp" "--stdio"))
  :activation-fn (lsp-activate-on "mdl")
  :server-id 'mdl-ls))

(add-hook 'mdl-mode-hook #'lsp)
```

### Using eglot (Built-in since Emacs 29)

```elisp
(add-to-list 'eglot-server-programs
             '(mdl-mode . ("mxcli" "lsp" "--stdio")))

(add-hook 'mdl-mode-hook #'eglot-ensure)
```

### File Mode

Define a basic major mode for `.mdl` files:

```elisp
(define-derived-mode mdl-mode prog-mode "MDL"
  "Major mode for editing MDL (Mendix Definition Language) files."
  (setq-local comment-start "-- ")
  (setq-local comment-end ""))

(add-to-list 'auto-mode-alist '("\\.mdl\\'" . mdl-mode))
```

## Sublime Text

### Using LSP Package

Install the [LSP](https://packagecontrol.io/packages/LSP) package, then add to your LSP settings:

```json
{
  "clients": {
    "mdl": {
      "enabled": true,
      "command": ["mxcli", "lsp", "--stdio"],
      "selector": "source.mdl",
      "schemes": ["file"]
    }
  }
}
```

## Helix

Add to `~/.config/helix/languages.toml`:

```toml
[[language]]
name = "mdl"
scope = "source.mdl"
file-types = ["mdl"]
language-servers = ["mdl-ls"]
comment-token = "--"

[language-server.mdl-ls]
command = "mxcli"
args = ["lsp", "--stdio"]
```

## Generic LSP Client

For any editor with LSP support, the key configuration values are:

| Parameter | Value |
|-----------|-------|
| Command | `mxcli lsp --stdio` |
| Transport | stdio |
| Language ID | `mdl` |
| File extensions | `.mdl` |
| Trigger characters | `.`, ` ` |
