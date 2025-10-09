# Configuration Examples

> **⚠️ DEPRECATED:** This directory is kept for backward compatibility only.
>
> **All examples have been moved to [`docs/examples/`](../docs/examples/)**

## New Location

Configuration examples and documentation are now organized in the `docs/` directory:

- 📁 **[docs/examples/](../docs/examples/)** - All configuration examples
- 📖 **[docs/examples/README.md](../docs/examples/README.md)** - Detailed usage guide
- 📚 **[docs/](../docs/)** - Complete documentation

## Available Examples

The following examples are available in `docs/examples/`:

- `config.yaml` - Basic standalone configuration
- `config.master.yaml` - Master server for replication
- `config.slave.yaml` - Slave server for replication
- `config.mysql.yaml` - MySQL backend configuration
- `config.postgres.yaml` - PostgreSQL backend configuration
- `config.docker.yaml` - Docker deployment

## Migration Guide

Update your commands to use the new location:

```bash
# Old (deprecated)
cp examples/config.master.yaml config.yaml

# New
cp docs/examples/config.master.yaml config.yaml
```

The configuration file format is identical - only the location has changed.

## Documentation

For detailed setup instructions, see:
- [Replication Guide](../docs/REPLICATION.md)
- [Web Admin Guide](../docs/WEBADMIN.md)
- [Docker Guide](../docs/DOCKER.md)
- [Main README](../docs/README.md)
