# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in this project, please report it responsibly.

**Do NOT open a public GitHub issue for security vulnerabilities.**

Instead, please use one of the following methods:

1. **GitHub Security Advisories** (preferred): [Report a vulnerability](https://github.com/mendixlabs/mxcli/security/advisories/new)
2. **Email**: Send details to the repository maintainers via the email addresses listed in their GitHub profiles

### What to include

- Description of the vulnerability
- Steps to reproduce
- Affected versions
- Potential impact
- Suggested fix (if any)

### What to expect

- **Acknowledgment** within 3 business days
- **Assessment** within 10 business days
- **Fix or mitigation** for confirmed vulnerabilities, coordinated with you before public disclosure

## Supported Versions

| Version | Supported |
|---------|-----------|
| Latest release | Yes |
| Nightly builds | Best-effort |
| Older releases | No |

## Security Practices

- Static analysis via CodeQL (Go) on every push
- Go vulnerabilities are scanned with `govulncheck` in CI
- Dependencies are monitored via Dependabot
- CycloneDX SBOM is available via `make sbom`
- Release binaries are built with `CGO_ENABLED=0` (no C dependencies)
