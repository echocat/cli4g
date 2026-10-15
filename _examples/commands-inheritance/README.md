# Parent options

The leaf command borrows application options through `parent:"application"`
and the enclosing command's options through `parent:"matchingType"`. The pointers
refer to the same options objects; inherited flags are defined only on their owners.

Run from the repository root:

```shell
mise example commands-inheritance message greet Carl
# Hello, Carl!
# <exitCode 0>

mise example commands-inheritance message greet Carl --exitCode 66
# Hello, Carl!
# <exitCode 66>

mise example commands-inheritance message greet Carl --greeting Hey --punctuation '?'
# Hey, Carl?
# <exitCode 0>

mise example commands-inheritance message greet --help
# <usage with inherited --greeting and --punctuation flags>
```
