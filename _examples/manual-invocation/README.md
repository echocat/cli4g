# Manual invocations

`Create` compiles one application. Each `Parse` creates independent options;
`Execute` runs their handler and an explicit deferred `Close` releases them.
Application cleanup runs after invocation cleanup, with cleanup errors preserved.

The first invocation uses the supplied CLI arguments. A second uses fixed inputs;
both counters start at zero. A help request or error stops before the second invocation.

Run from the repository root:

```shell
mise example manual-invocation
# Hello, World! (calls=1)
# Hey, Carl! (calls=1)

mise example manual-invocation --greeting Hi Alice
# Hi, Alice! (calls=1)
# Hey, Carl! (calls=1)

mise example manual-invocation --help
# <usage text only>

mise example manual-invocation --unknown
# Error: unknown flag --unknown
# program exit code: 2
```
