# Security policy

## Supported versions

Security fixes are applied to the latest released minor version and the
default branch. Preview builds and older releases may be asked to upgrade
before a fix is backported.

| Version | Supported |
| --- | --- |
| Latest release | Yes |
| Default branch | Best effort |
| Older releases | No |

## Reporting a vulnerability

Do not open a public issue for a vulnerability or include secrets, cookies,
profile data, proxy credentials, local paths, or browsing history in a report.

Use the repository's **Security** tab and choose **Report a vulnerability**
to create a private vulnerability report. If private vulnerability reporting
is not enabled, ask a repository owner to enable it without disclosing the
technical details publicly. Maintainers can continue the discussion in a
private GitHub Security Advisory.

Include the affected version and operating system, impact, minimal reproduction
steps, and any suggested remediation. Redact personal data and use synthetic
profiles. Reports are triaged on a best-effort basis; no response or disclosure
deadline is promised until a maintainer confirms one in the private advisory.

## Security boundaries

ProfileWeave is a local-first profile isolation and configuration coherence
tool. Its HTTP server must remain loopback-only. It is not designed to be
exposed through a reverse proxy, LAN binding, port-forwarding service, or
public tunnel. Proxy authentication secrets are intentionally unsupported.

The project does not claim to make a browser undetectable or to bypass site
risk controls. CAPTCHA solving, credential stuffing, account farming, and
target-specific evasion are outside scope.

Reports about an unmodified third-party browser should be sent to that browser
vendor. Reports about ProfileWeave command construction, path handling, local
API access, profile isolation, persistence, or release integrity are in scope.
