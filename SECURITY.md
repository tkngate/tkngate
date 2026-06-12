# Security Policy

## Supported Versions

Currently, only the latest active branch is supported with security updates.

| Version | Supported          |
| ------- | ------------------ |
| 0.6.x   | :white_check_mark: |
| 0.5.x   | :x:                |
| < 0.4.x | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability within tkngate, please do not disclose it publicly. 

Instead, send an email to the repository owner or open a private security advisory on GitHub. We will respond as quickly as possible to acknowledge the report and outline the next steps.

Please provide:
- A description of the vulnerability.
- Steps to reproduce it.
- Potential impact on users or infrastructure.

## Scope of Security
tkngate relies on local SQLite storage and AES-256-GCM encryption for managing master keys. Any vulnerabilities found in the `internal/crypto` blind key pooling engine or the `internal/proxy` transport intercepts will be treated as Critical priority.
