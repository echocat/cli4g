# Releasing

cli4g is released as a single Go module: `github.com/echocat/cli4g`.
Release tags use canonical versions such as `v0.x.y` or `v1.x.y`.

## Before publishing

1. Run `mise run check:full` on the intended release commit.
2. Confirm that Continuous Integration and CodeQL succeeded for that commit on
   `main`.
3. Prepare a short summary of the published API and features.

## Publish

Create and publish a GitHub release for the module tag, for example `v0.1.0`.
Publishing runs Continuous Integration and CodeQL for the release.
No additional module tags or generated release commits are needed.

If verification fails, correct the problem and publish a new patch version.
Do not move or delete a published tag.
