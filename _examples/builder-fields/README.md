# Builder-defined fields

The options have no tags. `NewFlag(codec, name, description)` and
`NewArgument(codec, name, description)` define the inputs; `WithField` binds each
value to its options field. Application and command constructors accept the same
optional name/description strings, without a codec argument. Multiple description
parts are joined with spaces.

Run from the repository root:

```shell
mise example builder-fields World
# Hello, World!

mise example builder-fields -g Hey Carl
# Hey, Carl!

mise example builder-fields --help
```
