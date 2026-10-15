# Named codecs

Both fields are `time.Duration`. `WithCodec(..., "positive")` and `WithCodec(..., "nonnegative")` register different policies; each field selects one through its `type` tag. A wrapper delegates parsing and formatting to the standard codec and validates in `Resolve`. Its own `ToRaw` preserves that behavior during registration.

Run from the repository root:

```shell
mise example named-codecs
# interval=1s delay=0s

mise example named-codecs --interval 250ms --delay 2s
# interval=250ms delay=2s

mise example named-codecs --delay 0s
# interval=1s delay=0s

mise example named-codecs --interval 0s
# Error: flag --interval: duration must be at least 1ns; got 0s
# <usage text>
# program exit code: 2

mise example named-codecs --delay=-1s
# Error: flag --delay: duration must be at least 0s; got -1s
# <usage text>
# program exit code: 2

mise example named-codecs --help
```
