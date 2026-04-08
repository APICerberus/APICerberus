# Security Policy

## Reporting Security Vulnerabilities

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via email to:
- **security@apicerberus.local**

We aim to respond to security reports within 48 hours.

### What to Include

When reporting a vulnerability, please include:

1. **Description**: Clear description of the vulnerability
2. **Impact**: What could an attacker achieve?
3. **Reproduction**: Step-by-step instructions to reproduce
4. **Environment**: Version, configuration details
5. **Proof of Concept**: If applicable
6. **Suggested Fix**: If you have one

### Response Process

1. **Acknowledgment**: We will acknowledge receipt within 48 hours
2. **Assessment**: We will assess the severity and impact
3. **Timeline**: We will provide an estimated fix timeline
4. **Updates**: We will keep you informed of progress
5. **Resolution**: We will notify you when fixed
6. **Disclosure**: We will coordinate public disclosure timing

### Security Bug Bounty

We appreciate security researchers helping improve APICerebrus security. While we don't have a formal bounty program, we:

- Credit researchers in security advisories (with permission)
- Add contributors to our Hall of Fame
- Prioritize fixes for reported vulnerabilities

## Supported Versions

Security updates are provided for:

| Version | Supported | Security Fixes Until |
|---------|-----------|---------------------|
| 1.x.x   | ✅ Yes    | Current + 12 months |
| 0.x.x   | ❌ No     | End of life         |

We recommend always running the latest stable version.

## Security Measures

### Implemented Protections

- **Authentication**: API keys, JWT (RS256/HS256), session-based auth
- **Authorization**: Role-based access control, endpoint permissions
- **Cryptography**: bcrypt passwords, SHA-256 key hashing, TLS 1.2+
- **Input Validation**: SQL injection prevention, path traversal protection
- **Transport Security**: HTTPS, HSTS, secure headers
- **Rate Limiting**: Multiple algorithms, per-consumer limits
- **Audit Logging**: Comprehensive request/response logging with masking

### Security Headers

All responses include:
```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Content-Security-Policy: default-src 'self'; frame-ancestors 'none'
Referrer-Policy: strict-origin-when-cross-origin
Strict-Transport-Security: max-age=31536000; includeSubDomains; preload (HTTPS only)
```

## Security Checklist for Deployments

Before deploying to production:

- [ ] Change default admin password
- [ ] Enable HTTPS with valid certificates
- [ ] Configure CORS appropriately
- [ ] Set up rate limiting
- [ ] Enable audit logging
- [ ] Configure request size limits
- [ ] Review security headers
- [ ] Set up monitoring/alerting
- [ ] Configure backup encryption
- [ ] Run security scan: `make security`

## Known Security Considerations

See [SECURITY.md](../SECURITY.md) for detailed security documentation.

## Security Advisories

Security advisories will be published as:
- GitHub Security Advisories
- Release notes
- Email notifications to security-announce list

## Hall of Fame

We thank the following security researchers for responsibly disclosing vulnerabilities:

*No entries yet - be the first!*

## Contact

- **Security Issues**: security@apicerberus.local
- **General Support**: support@apicerberus.local
- **Emergency Contact**: Available to enterprise customers
