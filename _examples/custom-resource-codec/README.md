# Resource-owning codec

`documentCodec` keeps a path string as its resource-free requirement. Only `Resolve` opens the file and immediately registers cleanup with `scope.OnClose`. The handler borrows a buffered reader; invocation cleanup closes the underlying file, including after a handler error. Document contents go to stdout, while Open/Close diagnostics go to stderr. Help does not open anything.

Run from the repository root (paths are relative to this example's directory):

```shell
mise example custom-resource-codec sample.txt
# Open: sample.txt
# Document contents.
# Close: sample.txt

mise example custom-resource-codec sample.txt --fail
# Open: sample.txt
# Close: sample.txt
# <execution error without usage text>
# program exit code: 1

mise example custom-resource-codec --help
# <usage text without Open/Close>
```
