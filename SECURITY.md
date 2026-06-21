# Security Policy

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability in feedreader, please report it responsibly by emailing [hi@boringcode.dev](mailto:hi@boringcode.dev) instead of using the public issue tracker.

When reporting a security issue, please include:

- A description of the vulnerability
- Steps to reproduce the issue (if applicable)
- The affected version(s)
- Any potential impact or proof of concept

We will acknowledge your report within 48 hours and work with you to understand and resolve the issue promptly.

## Security Considerations for Deployment

Since feedreader is a self-hosted application, the following best practices are recommended:

### Network Security

- Run feedreader behind a reverse proxy (nginx, Caddy, etc.) with TLS/HTTPS enabled
- Use a firewall to restrict access to the application
- Consider running the application in an isolated network segment

### Data Security

- Store the SQLite database file on an encrypted filesystem
- Regularly back up the database
- Ensure proper file permissions on the data directory

### Container Security

- Keep the base Docker image and Go runtime updated
- Run the container with minimal required privileges
- Use secrets management for sensitive configuration values

### Dependency Management

- Monitor dependencies for security updates
- Keep Go and other dependencies current

## Supported Versions

Security updates will be provided for:

- The current release and latest version
- The previous minor version if critical

## Disclosure Timeline

We aim to:

1. Acknowledge receipt of the report within 48 hours
2. Begin investigation and reproduce the issue
3. Develop and test a fix
4. Publish a security release (if needed)
5. Notify the reporter of the resolution

We appreciate your responsible disclosure and will credit you appropriately unless you prefer to remain anonymous.
