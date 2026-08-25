# Security Policy

## Supported versions

Security fixes primarily target the latest published release and maintained
development branches where appropriate. Older releases may not receive fixes.
Users should reproduce a report against the latest available release when it is
safe to do so and should include the affected version in the report.

## Reporting a vulnerability

**Do not open a public GitHub Issue for an undisclosed security vulnerability.**
Do not include exploit details, private data, credentials, or affected
infrastructure in a public issue, discussion, or pull request.

If the repository's **Security** tab offers **Report a vulnerability**, use
GitHub Private Vulnerability Reporting. If that option is unavailable, contact
a repository maintainer through a private contact method they publish on their
GitHub profile and ask to establish a secure reporting channel before sending
technical details.

The repository does not currently publish a dedicated security email address or
bug-bounty program. If no private contact method is available, you may open a
minimal public issue asking maintainers to provide a private channel, but it
must not identify the vulnerability, affected versions, systems, or people.

Include as much of the following as is safe and relevant:

- affected component and version or commit;
- environment and prerequisites;
- reproduction steps;
- security impact and realistic attack scenario;
- a minimal proof of concept, when it can be shared safely; and
- suggested mitigation, if known.

Do not test against production, access other people's data, degrade service, or
retain data beyond what is necessary to demonstrate the issue.

## Relevant security areas

Security reports may include, but are not limited to:

- authentication, session, JWT, and credential handling;
- authorization, ownership, and role enforcement;
- official testcase confidentiality;
- sandbox escape or code execution outside the intended executor boundary;
- cross-service and gateway trust boundaries;
- injection, request forgery, or unsafe deserialization;
- secret, private-data, or source-code exposure; and
- build, release, and deployment security.

A report in one of these areas is not automatically a confirmed vulnerability.
Maintainers will evaluate the behavior, impact, supported threat model, and
available evidence.

## Coordinated disclosure

Please keep a report and any proof of concept confidential while maintainers
investigate and prepare a fix. Coordinate publication with the maintainers so
users have a reasonable opportunity to update. This small open-source project
does not promise a response SLA, remediation deadline, reward, or bug bounty,
but maintainers will aim to acknowledge and assess good-faith reports as their
availability permits.
