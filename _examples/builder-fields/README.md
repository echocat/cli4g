# Builder-defined fields

The options have no tags. Builder callbacks define flags and arguments;
`WithField` binds each value to its options field.

Run from the repository root:

```shell
mise example builder-fields World
# Hello, World!

mise example builder-fields -g Hey Carl
# Hey, Carl!

mise example builder-fields --help
```
