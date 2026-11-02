# Security Policy — BramandScript

> **Specification Standard**: Responsible Vulnerability Disclosure and Security Maintenance Protocol.

The BramandScript core maintainers take security and runtime predictability seriously. BramandScript is designed with zero external third-party dependencies, drastically minimizing the supply-chain attack surface. However, virtual machine runtimes, compilers, and sandboxes require rigorous security scrutiny.

---

## 1. Supported Versions

Security updates and patches are actively maintained for the following versions:

| Version | Release Date | Supported          | Status             |
|:--------|:-------------|:-------------------|:-------------------|
| 0.1.x   | November 2026| [OK] Supported     | Active Maintenance |
| < 0.1.0 | N/A          | [NO] Not Supported | Deprecated         |

---

## 2. Reporting a Vulnerability

If you discover a potential security vulnerability in BramandScript (including Bytecode VM memory corruption, buffer overflows, arbitrary execution escapes, denial-of-service vulnerabilities, or cryptographic issues), please report it responsibly.

### Disclosure Channels
- **GitHub Private Vulnerability Reporting**: Submit through the GitHub repository's [Security Advisories tab](https://github.com/Codeleafly/bramandScript/security/advisories/new).
- **Direct Maintainer Contact**: Email the security response team directly at:  
  `security@codeleafly.dev` (Subject: `[SECURITY] BramandScript Vulnerability Report`)

Please include:
1. Clear description of the vulnerability and its potential impact.
2. Minimal reproduction script (`.bmd` or test harness).
3. System environment details (OS, architecture, Go compiler version).
4. Any proposed patch or mitigation strategy (if available).

Please do **NOT** open public issues or discuss suspected vulnerabilities on public social forums before private triage and coordinate release.

---

## 3. Response Timelines & SLAs

The BramandScript maintainers commit to the following response timeline:

- **Initial Acknowledgement**: Within 48 hours of initial receipt.
- **Triage & Severity Assessment**: Within 5 business days, confirming whether the report is valid and assigning a severity rating (CVSS).
- **Remediation & Patching**: Dedicated patch development and local regression verification on a private branch.
- **Public Disclosure & Release**: A coordinated patch release (`v0.1.x`) and public Common Vulnerabilities and Exposures (CVE) disclosure when the patch is ready for general distribution.

---

## 4. In-Scope Security Areas

The following subsystems are in scope for security evaluations:

- **Stack Virtual Machine (`vm/`)**: Stack overflow defenses, frame bounds, memory corruption, and closure reference safety.
- **Bytecode Compiler (`compiler/`) & Opcode Encoding (`code/`)**: Constant pool index bounds, jump address validation, and bytecode verification.
- **Standard Library Builtins (`vm/builtins.go`)**: File system traversal protections, cryptographic robustness (CSPRNG, SHA-256, HMAC), and socket safety.
- **Standalone Builder (`main.go`)**: Binary embedding integrity and cross-platform compilation security.
