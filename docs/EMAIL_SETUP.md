# Email Setup Guide for StinkyKitty CMS

This guide covers the complete email configuration for StinkyKitty, including SMTP setup, DNS records, and troubleshooting.

## Overview

StinkyKitty sends transactional emails for:
- Password reset links (when creating new camp users)
- Welcome emails for new users
- Contact form submissions to camp owners

## SMTP Configuration

### Required Environment Variables

Set these environment variables where you run the StinkyKitty server:

```bash
export SMTP="smtp.example.com"        # Your SMTP server hostname
export SMTP_PORT="587"                # Port 587 for STARTTLS, 465 for implicit TLS
export EMAIL="noreply@campasaur.us"   # Your sending email address
export SMTP_SECRET="your-password"    # SMTP authentication password
```

### Supported SMTP Methods

StinkyKitty supports both common SMTP connection methods:

- **Port 587**: STARTTLS (connect plain, then upgrade to TLS) - **Recommended**
- **Port 465**: Implicit TLS (TLS from connection start)

The code automatically detects which method to use based on the port number.

### Common SMTP Providers

#### IONOS (1&1)
```bash
SMTP="smtp.ionos.com"
SMTP_PORT="587"
EMAIL="your-email@yourdomain.com"
SMTP_SECRET="your-app-password"
```

#### Gmail
```bash
SMTP="smtp.gmail.com"
SMTP_PORT="587"
EMAIL="your-email@gmail.com"
SMTP_SECRET="your-app-password"  # Use App Password, not account password
```

#### SendGrid
```bash
SMTP="smtp.sendgrid.net"
SMTP_PORT="587"
EMAIL="apikey"                    # Literally the word "apikey"
SMTP_SECRET="your-sendgrid-api-key"
```

#### Mailgun
```bash
SMTP="smtp.mailgun.org"
SMTP_PORT="587"
EMAIL="postmaster@your-domain.mailgun.org"
SMTP_SECRET="your-smtp-password"
```

## DNS Configuration

Proper DNS records are **critical** for email deliverability. Without these, most mail servers will reject your emails as spam.

### 1. SPF Record (Required)

SPF (Sender Policy Framework) authorizes your server to send email for your domain.

**Add this TXT record to your domain:**

```
Type: TXT
Name: @ (or campasaur.us)
Value: v=spf1 ip4:YOUR.SERVER.IP.ADDRESS ~all
```

**Example for campasaur.us:**
```
v=spf1 ip4:216.250.115.223 ~all
```

**Multiple IPs or services:**
```
v=spf1 ip4:216.250.115.223 include:_spf.google.com ~all
```

### 2. Reverse DNS / PTR Record (Required)

The PTR record maps your IP address back to a hostname. **You must contact your hosting provider to set this up** - it cannot be set in your domain's DNS.

**Request your hosting provider set:**
```
IP: YOUR.SERVER.IP.ADDRESS
PTR: mail.campasaur.us
```

**How to verify:**
```bash
host YOUR.SERVER.IP.ADDRESS
# Should return: YOUR.IP.in-addr.arpa domain name pointer mail.campasaur.us.
```

### 3. MX Record (Optional but Recommended)

If you want to receive email at your domain:

```
Type: MX
Name: @ (or campasaur.us)
Priority: 10
Value: mail.campasaur.us
```

### 4. DKIM (Optional but Highly Recommended)

DKIM provides cryptographic authentication. Most SMTP providers (like IONOS, SendGrid, Mailgun) handle this automatically. If you're running your own mail server, you'll need to:

1. Generate DKIM keys
2. Add the public key as a TXT record
3. Configure your mail server to sign outgoing emails

**For most users using a managed SMTP service, this is handled automatically.**

## Testing Email Configuration

### 1. Check DNS Records

```bash
# Check SPF
dig TXT campasaur.us | grep spf

# Check PTR
host YOUR.SERVER.IP.ADDRESS

# Check MX
dig MX campasaur.us
```

### 2. Test SMTP Connection

```bash
# Test STARTTLS connection (port 587)
openssl s_client -starttls smtp -connect smtp.ionos.com:587

# Test implicit TLS connection (port 465)
openssl s_client -connect smtp.ionos.com:465
```

### 3. Create Test User in StinkyKitty

Create a new camp with a test email address and check the server logs:

```bash
# Watch for email-related log messages
tail -f /path/to/localhost.log | grep -E "INFO|ERROR|SUCCESS"
```

**Look for:**
- `INFO: Attempting to send password reset email to user@example.com`
- `INFO: Reset URL: https://...`
- `SUCCESS: Password reset email sent to user@example.com`

**Or errors like:**
- `ERROR: Failed to send password reset email: ...`

### 4. Check Email Deliverability

Use online tools to test your email configuration:
- [MXToolbox](https://mxtoolbox.com/SuperTool.aspx) - Check SPF, DKIM, MX records
- [Mail-tester](https://www.mail-tester.com/) - Test email spam score

## Common Issues and Solutions

### Error: "tls: first record does not look like a TLS handshake"

**Cause:** Port/TLS method mismatch

**Solution:**
- If using port 587, the code uses STARTTLS (correct)
- If using port 465, the code uses implicit TLS (correct)
- Verify SMTP_PORT matches your provider's requirements

### Error: "554 Transaction failed - invalid DNS PTR resource record"

**Cause:** Missing or incorrect PTR record

**Solution:**
1. Contact your hosting provider to set up reverse DNS
2. PTR should point to a hostname like `mail.campasaur.us`
3. Ensure the hostname matches your sending domain

### Error: "550 5.7.1 Relaying denied"

**Cause:** SMTP server doesn't recognize you as authorized sender

**Solution:**
- Verify EMAIL environment variable matches an authorized address
- Check SMTP authentication credentials
- Ensure SPF record includes your server IP

### Error: "535 Authentication failed"

**Cause:** Invalid SMTP credentials

**Solution:**
- Verify SMTP_SECRET is correct
- For Gmail, use an App Password, not your account password
- For SendGrid, EMAIL should be literally "apikey"

### Emails going to spam

**Solutions:**
1. Verify all DNS records (SPF, PTR, optionally DKIM)
2. Use a professional SMTP service (IONOS, SendGrid, Mailgun)
3. Ensure From address matches your domain
4. Don't send too many emails in a short period
5. Include proper unsubscribe options for bulk emails

## Email Templates

StinkyKitty includes these email templates:

### Password Reset Email
Sent when creating new camp users. Includes a 24-hour password reset link.

**Location:** `internal/email/email.go` - `SendPasswordReset()`

### Welcome Email
Can be used for new user onboarding.

**Location:** `internal/email/email.go` - `SendNewUserWelcome()`

### Contact Form Notification
Sent to camp owners when visitors submit the contact form.

**Location:** `internal/handlers/public.go` - `ContactFormHandler()`

## Security Best Practices

1. **Use App Passwords**: For Gmail, always use App Passwords, never your main password
2. **Rotate Credentials**: Regularly rotate SMTP passwords
3. **Limit Rate**: Implement rate limiting for contact forms (already done in StinkyKitty)
4. **Monitor Logs**: Watch for authentication failures or unusual sending patterns
5. **Use TLS**: Always use port 587 (STARTTLS) or 465 (implicit TLS), never port 25 unencrypted

## Environment Setup Example

### Systemd Service (if running as service)

```ini
[Unit]
Description=StinkyKitty CMS
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/var/lib/stinkykitty
Environment="SMTP=smtp.ionos.com"
Environment="SMTP_PORT=587"
Environment="EMAIL=noreply@campasaur.us"
Environment="SMTP_SECRET=your-password-here"
ExecStart=/usr/bin/stinky server start
Restart=always

[Install]
WantedBy=multi-user.target
```

### Manual Shell Script

```bash
#!/bin/bash
export SMTP="smtp.ionos.com"
export SMTP_PORT="587"
export EMAIL="noreply@campasaur.us"
export SMTP_SECRET="your-password-here"

cd /var/lib/stinkykitty
/usr/bin/stinky server start 2>&1 | tee localhost.log
```

## Troubleshooting Checklist

When emails aren't sending:

- [ ] SMTP environment variables are set correctly
- [ ] SMTP credentials are valid (test with `openssl s_client`)
- [ ] Port matches TLS method (587=STARTTLS, 465=implicit TLS)
- [ ] SPF record exists and includes server IP
- [ ] PTR record points to valid hostname
- [ ] Server IP is not blacklisted (check MXToolbox)
- [ ] From address domain matches or is authorized
- [ ] Check server logs for specific error messages
- [ ] Firewall allows outbound connections on SMTP port

## Additional Resources

- [SPF Record Syntax](https://www.dmarcanalyzer.com/spf/spf-record-check/)
- [DKIM Setup Guide](https://www.dkim.org/)
- [Email Deliverability Best Practices](https://sendgrid.com/blog/email-deliverability-best-practices/)
- [IONOS Email Help](https://www.ionos.com/help/email/)

## Code References

- SMTP client implementation: `internal/email/email.go`
- Password reset flow: `internal/handlers/admin_create_camp.go` (lines 864-910)
- Contact form handler: `internal/handlers/public.go` - `ContactFormHandler()`
