# Argument syntax

An omitted optional argument uses its default. The final slice argument collects all remaining positional values. `--` ends flag parsing, allowing names beginning with `-` and treating `--help` as a literal value. The example prints names without opening files.

Run from the repository root:

```shell
mise example argument-syntax
# file=input.txt more=[]

mise example argument-syntax report.txt first.txt second.txt
# file=report.txt more=[first.txt second.txt]

mise example argument-syntax -- -- -report.txt --help
# file=-report.txt more=[--help]

mise example argument-syntax --help
```

In the third command, the first `--` belongs to mise; the second reaches cli4g.
