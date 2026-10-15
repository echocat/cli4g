# Dates and timestamps

The fields use `time.Time` and `[]time.Time`. `type:"date"` selects registered date-only codecs; `--at` uses the standard DateTime codec. The `until` argument selects two custom layouts through `type:"custom"`. Repeated reminders preserve order and duplicates. Inputs without a zone use UTC.

Run from the repository root:

```shell
mise example date-time
# day=2026-10-14 at=2026-10-14T12:30:45.123456789Z until=2026-10-15T18:00:00Z reminders=[]

mise example date-time --day "14 Oct 26" --at "14 Oct 26 14:30 +0200" --reminder 15.10.2026 --reminder 20261017 "16.10.2026 18:45"
# day=2026-10-14 at=2026-10-14T14:30:00+02:00 until=2026-10-16T18:45:00Z reminders=[2026-10-15 2026-10-17]

mise example date-time --at 20261016
# day=2026-10-14 at=2026-10-16T00:00:00Z until=2026-10-15T18:00:00Z reminders=[]

mise example date-time "2026-10-16T20:45:00.987654321+02:00"
# day=2026-10-14 at=2026-10-14T12:30:45.123456789Z until=2026-10-16T20:45:00.987654321+02:00 reminders=[]

mise example date-time --day 2026-02-30
# <invalid-date error>
# <usage text>
# program exit code: 2

mise example date-time --day 2026-10-14T00:00:00Z
# <date-only format error>
# <usage text>
# program exit code: 2

mise example date-time "2026-10-16 18:45:00"
# <custom-format error>
# <usage text>
# program exit code: 2

mise example date-time --help
```
