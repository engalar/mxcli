# Marketplace CLI Usage Guide

How to discover, download, and install Mendix Marketplace content from the command line with `mxcli marketplace`.

## Authentication

Marketplace access needs a Mendix Personal Access Token (PAT):

```bash
mxcli auth login                     # prompts for the PAT
# or non-interactively (CI):
mxcli auth login --token <PAT>
# or via the environment:
export MENDIX_PAT=<PAT>

mxcli auth status                    # confirm it validates
```

Credentials are stored at `~/.mxcli/auth.json` (mode `0600`).

## Discover Content

```bash
# search by name/publisher
mxcli marketplace search "database connector"

# show one item's details (by content id)
mxcli marketplace info 2888

# list available versions, optionally filtered by Mendix compatibility
mxcli marketplace versions 2888
mxcli marketplace versions 2888 --min-mendix 10.24.0
```

Search results are served from the marketplace API. Each item has a numeric **content id** (shown by `search`/`info`); you pass it to `download` and `install`.

## Download a `.mpk`

```bash
# latest version, saved to {name}_{version}.mpk in the current directory
mxcli marketplace download 2888

# a specific version, to a chosen path
mxcli marketplace download 170 --version 11.5.0 -o ./modules/commons.mpk
```

The download is atomic (written to a temp file and renamed), so a cancelled run never leaves a truncated `.mpk`.

## Install into a Project

`install` downloads the content and installs it via `mx module-import`:

```bash
# install a module
mxcli marketplace install 2888 -p app.mpr
mxcli marketplace install 170 --version 11.5.0 -p app.mpr
```

**Update safety**: If the module is already present in the project, the command reports the installed and target versions and stops:

```
Module "DatabaseConnector" is already installed (version 7.0.1).
Target version: 7.0.3.
In-place module updates are not applied automatically (they can discard
local edits and change persistent entity IDs, which loses data).
Update via Studio Pro.
```

## List Installed Modules

```bash
# show all marketplace modules installed in the project
mxcli marketplace list -p app.mpr

# JSON output
mxcli marketplace list -p app.mpr --json
```

## Check for Updates

```bash
# check a specific module
mxcli marketplace update 2888 -p app.mpr

# check all installed modules
mxcli marketplace update -p app.mpr
```

Updates are **reported only** — automatic in-place updates are not applied (see safety note above). Use Studio Pro to perform an ID-preserving merge.

## JSON Output

All display commands support `--json` for machine-readable output:

```bash
mxcli marketplace search "database connector" --json
mxcli marketplace info 2888 --json
mxcli marketplace versions 170 --json
mxcli marketplace list -p app.mpr --json
```

## Common Content IDs

| Module | ID |
|--------|----|
| Database Connector | 2888 |
| Community Commons | 170 |
