# Custom values

Types with `Set(string) error` and `String() string` are recognized automatically. `IsCumulative` allows repeated flags to update the same map within an invocation. Invalid input is explicitly marked with `cli.NewValidationErrorf`; adapters preserve the returned error's classification.

Run from the repository root:

```shell
mise example custom-values
# level=info headers=[]

mise example custom-values -l debug -H Accept=text/plain -H X-Team=cli4g
# level=debug headers=[Accept=text/plain, X-Team=cli4g]

mise example custom-values --level trace
# <input error and usage text>
# program exit code: 2
```
