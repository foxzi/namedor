# Package Documentation

This file describes the documentation included in namedot packages.

## Documentation Files Included

When you install namedot via DEB or RPM package, the following documentation files are automatically installed to `/usr/share/doc/namedot/`:

- **README.md** - Main documentation with:
  - Installation instructions
  - Quick start guide
  - GeoIP configuration with auto-download feature
  - REST API usage
  - CLI flags and examples
  
- **REPLICATION.md** - Master-Slave replication setup:
  - Configuration examples
  - Synchronization details
  - Troubleshooting

- **DOCKER.md** - Docker deployment guide:
  - Docker compose examples
  - Container configuration
  - Volume management

- **WEBADMIN.md** - Web admin panel documentation:
  - Setup instructions
  - Security configuration
  - Usage guide

- **tz.md** - Technical zone documentation:
  - Zone file format
  - Record types
  - Examples

- **LICENSE** - MIT License

## Accessing Documentation

After package installation:

```bash
# List all documentation
ls /usr/share/doc/namedot/

# Read main documentation
less /usr/share/doc/namedot/README.md

# Read specific guide
cat /usr/share/doc/namedot/DOCKER.md
```

## Online Documentation

Latest documentation is also available at:
https://github.com/piligrim/namedot
