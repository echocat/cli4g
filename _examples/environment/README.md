# Environment and defaults

CLI flags override environment variables; environment variables override defaults.

Run from the repository root using Bash:

```bash
mise example environment
# profile=development port=8080

CLI4G_PROFILE=production CLI4G_PORT=9090 mise example environment
# profile=production port=9090

CLI4G_PROFILE=production mise example environment --profile staging
# profile=staging port=8080

mise example environment --help
```
