## Summary

- Describe the change.

## Scope

- dua is a read-only disk analysis and status tool. Does this change add cleanup, deletion, or modification behavior? If yes, reconsider: the project is intentionally read-only.
- Does this change touch path scanning, symlink handling, or platform (Linux/macOS) branches?
- If yes, describe the new behavior or risk change clearly.

## Tests

- List the automated tests you ran (`go test ./...`).
- List any manual checks (e.g. `dua analyze --json /path`, `dua status --json`).

## Safety-related changes

- None.
