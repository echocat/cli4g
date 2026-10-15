# Simple greeting

A minimal [cli4g](https://github.com/echocat/cli4g) application: a required name, an optional greeting (default: `Hello`), and automatic help.

Run from the repository root:

```shell
mise example simple World
# Hello, World!
# program exit code: 0

mise example simple -g Hey Carl
# Hey, Carl!
# program exit code: 0

mise example simple --help
# <usage text>
# program exit code: 0

mise example simple
# Error: required argument name is missing
# <usage text>
# program exit code: 2
```
