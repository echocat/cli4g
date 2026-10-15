# Ctrl+C cancellation

`signal.NotifyContext` turns Ctrl+C (`os.Interrupt`) into context cancellation. The handler runs indefinitely, waiting on a local ticker and `ctx.Done()`. `--sleep` uses the standard `time.Duration` codec; `Validate` requires a positive duration and marks failures with `cli.NewValidationErrorf`. A deferred `ticker.Stop()` releases the ticker when the handler returns `ctx.Err()`. The standard error handler reports cancellation and exits with 130.

Run from the repository root:

```shell
mise example cancellation
# Working. Press Ctrl+C to stop.
# Step 1
# ...
# <press Ctrl+C>
# Error: context canceled
# program exit code: 130

mise example cancellation --sleep 250ms
# Working. Press Ctrl+C to stop.
# Step 1
# Step 2
# ...
# <press Ctrl+C>
# Error: context canceled
# program exit code: 130

mise example cancellation --sleep 0s
# Error: sleep must be greater than zero
# <usage text>
# program exit code: 2

mise example cancellation --help
# <usage text without starting work or a ticker>
```
