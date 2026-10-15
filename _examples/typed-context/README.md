# Typed context access

A helper receiving only `context.Context` reads application options with `ApplicationFromContext`, the selected parent command through an explicit `CommandFromContext` path, and the current command without a path. `FlagFromContext` reads the decoded `verbose` flag. Its explicit `default:"false"` supplies a present false value that still returns `found=true`. Each getter checks whether the requested value exists.

Run from the repository root:

```shell
mise example typed-context message greet Carl
# Hello, Carl! verbose=false

mise example typed-context message greet Carl --greeting Hey --punctuation '?' -v
# Hey, Carl? verbose=true

mise example typed-context message greet --help
```
