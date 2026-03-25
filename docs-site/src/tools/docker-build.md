# mxcli docker build

The `mxcli docker build` command builds a Mendix application using mxbuild in Docker, including support for PAD (Platform-Agnostic Deployment) patching.

## Usage

```bash
mxcli docker build -p app.mpr
```

## What It Does

1. **Downloads mxbuild** -- Automatically downloads the correct mxbuild version for your project if not already available
2. **Runs mxbuild** -- Executes the Mendix build toolchain in a Docker container
3. **PAD patching** -- Applies Platform-Agnostic Deployment patches (Phase 1) to the build output
4. **Produces artifact** -- Generates a deployable Mendix deployment package (MDA file)

## PAD Patching

PAD (Platform-Agnostic Deployment) patching modifies the build output to be compatible with container-based deployment platforms. This is essential for deploying Mendix applications to Kubernetes, Cloud Foundry, or other container orchestrators.

## Validation with mx check

To validate a project without building:

```bash
mxcli docker check -p app.mpr
```

This runs `mx check` against the project and reports any errors:

```
Checking app for errors...
The app contains: 0 errors.
```

## Auto-Download mxbuild

mxcli automatically downloads the correct mxbuild version for your project:

```bash
# Explicit setup
mxcli setup mxbuild -p app.mpr

# Or let docker build handle it automatically
mxcli docker build -p app.mpr
```

The downloaded mxbuild is cached at `~/.mxcli/mxbuild/{version}/` for reuse.
