# StinkyKitty CMS

A multi-tenant CMS platform designed to replace WordPress for Burning Man camps and similar community groups.

## Features

- **Structured Content Blocks** - Rich pages without security risks (hero, text, gallery, video, buttons)
- **Multi-Tenant Hosting** - Host dozens of camps on a single server
- **Automatic SSL** - Let's Encrypt integration with wildcard support
- **Flexible Scaling** - SQLite for small sites, MariaDB for busy ones
- **Built-in Backups** - Automatic scheduled backups with full data portability
- **CLI-First** - Easy administration via `stinky` command

## Quick Start

```bash
# Build the CLI
go build -o stinky cmd/stinky/main.go

# Run it
./stinky
```

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

- [Design Document](docs/plans/2025-12-20-stinkykitty-cms-design.md) - Complete system design

## License

TBD

## Contributing

TBD
