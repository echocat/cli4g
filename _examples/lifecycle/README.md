# Options lifecycle

After validation, `Initialize` opens a temporary file owned by the options. The handler reads it; `Close` closes and removes it. The trace shows cleanup after success, failed initialization, and failed execution. Help skips the lifecycle.

Run from the repository root:

```shell
mise example lifecycle
# Initialize
# Execute
# Hello
# Close

mise example lifecycle --fail-initialize
# Initialize
# Close
# <initialization error without usage text>
# program exit code: 1

mise example lifecycle --fail
# Initialize
# Execute
# Close
# <execution error without usage text>
# program exit code: 1

mise example lifecycle --help
# <usage text without Initialize/Execute/Close>
```
