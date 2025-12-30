# StinkyKitty CMS

A multi-tenant CMS platform designed to replace WordPress for Burning Man camps and similar community groups.

## Features

- **Multi-tenant architecture** - Host unlimited camp websites
- **Block-based content editor** - Text, images, headings, quotes, buttons, video, columns
- **Media library** - Centralized image management with tagging, search, and usage tracking
- **User management** - Manage site users, reset passwords, control access
- **Google Analytics** - Built-in analytics tracking
- **Customizable themes** - Colors, fonts, layouts
- **Custom copyright** - Editable footer text per site
- **Fixed header navigation** - Professional site headers
- **Search functionality** - Full-text search across pages
- **Contact forms** - Embedded contact forms
- **Automatic SSL** - Let's Encrypt integration with on-demand certificate provisioning
- **Built-in Backups** - Automatic scheduled backups with full data portability
- **CLI administration** - Command-line site management

## Quick Start

```bash
# Build the CLI
go build -o stinky cmd/stinky/main.go

# Run it
./stinky
```

## SSL/TLS Configuration

StinkyKitty includes automatic SSL/TLS certificate provisioning via Let's Encrypt.

### Enable HTTPS

Configure TLS settings:

```bash
# Required: Enable TLS
./stinky config set server.tls_enabled true

# Required: Email for Let's Encrypt notifications
./stinky config set tls.email "admin@yourdomain.com"

# Required: Your base domain
./stinky config set server.base_domain "yourdomain.com"

# Optional: Use staging for testing (recommended for development)
./stinky config set tls.staging true
```

### Testing with Staging Mode

Before deploying to production, test with Let's Encrypt's staging environment to avoid rate limits:

```bash
./stinky config set tls.staging true
sudo ./stinky server start
```

Staging certificates will show browser warnings but confirm the ACME flow works correctly.

### Production Deployment

Requirements:
- Server with public IP address
- DNS A records pointing to your server
- Ports 80 and 443 open in firewall
- Root/sudo privileges (for binding to privileged ports)

Enable production certificates:

```bash
./stinky config set tls.staging false
sudo ./stinky server start
```

Certificates are automatically provisioned on first HTTPS request to each domain.

### Certificate Management

Check certificate status:

```bash
./stinky tls status
```

Certificates automatically renew 30 days before expiration.

### Troubleshooting

**"Permission denied" on port 80/443**
- Requires root/sudo: `sudo ./stinky server start`

**Certificate provisioning fails**
- Verify DNS points to your server: `dig yourdomain.com`
- Ensure ports 80/443 are open: `netstat -tlnp | grep -E ':(80|443)'`
- Check server logs for ACME challenge errors
- Verify you're not hitting Let's Encrypt rate limits

**Rate limits**
- Use `tls.staging = true` for testing
- Production limit: 50 certificates per domain per week

See [TLS Design Document](docs/plans/2025-12-23-automatic-ssl-design.md) for architecture details.

## Project Status

🚧 **Early Development** - See [design document](docs/plans/2025-12-20-stinkykitty-cms-design.md) for the full vision.

## Technology Stack

- Go 1.24+ (with hybrid PQC support)
- Gin (HTTP routing)
- Cobra (CLI framework)
- Viper (Configuration)
- GORM (Database ORM)
- SQLite & MariaDB support

## Documentation

- [Features Guide](docs/FEATURES.md) - Complete feature documentation
- [Design Document](docs/plans/2025-12-20-stinkykitty-cms-design.md) - Complete system design
- [TLS/SSL Design](docs/plans/2025-12-23-automatic-ssl-design.md) - Automatic HTTPS configuration
- [Media Library Design](docs/plans/2025-12-29-media-library-design.md) - Image management and organization

## License

TBD

## Contributing

TBD
