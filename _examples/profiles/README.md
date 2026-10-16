# Profiles

The same `convert` command has different typed options, flags, positional
arguments, defaults, and handlers depending on `ParseOpts.Profiles`.

The example defaults to `text`. Its embedding code reads `CLI4G_PROFILES` as a
comma-separated list and passes it to `ParseOpts`; this is application policy,
not automatic library behavior.

```shell
mise example profiles convert
# hello!

mise example profiles convert --foo world '?'
# world?

mise example profiles convert --help
# Help for the text variant only.
```

Set `CLI4G_PROFILES=number` in your shell, then run:

```shell
mise example profiles convert --foo 7 6
# 42
```

With `CLI4G_PROFILES=text,number`, parsing fails because two `convert` commands
would be active. Unknown profiles also fail, including with `--help`.

See [profile rules and introspection](../../docs/profiles.md).
