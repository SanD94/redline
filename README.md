# Redline

Redline converts reviewed `.docx` files into a stable Markdown review workspace. The current implemented slice is `redline reveal`.

## Design stance

Redline treats the local VCS as a first-class dependency, not as an optional export target. The program should work like a UNIX-style tool: use the system tools that already solve the job well, rather than carrying an isolated second diff engine.

- `jj` or `git` is required for reveal workspaces.
- Pandoc is expected to be used where document conversion is the right abstraction.
- Redline should not maintain a redundant review-data diff format for tracked changes. The VCS diff is the review diff.
- Redline-specific output should focus on stable Markdown sections, a manifest, and a human-readable comment report.

## Current command

```sh
redline reveal <file.docx> --output <dir>
```

`reveal` reconstructs two document states from Word tracked changes:

1. the old version, with insertions rejected and deletions kept;
2. the new version, with insertions kept and deletions rejected.

It writes the old version first, snapshots it with `jj` when available or `git` otherwise, then overwrites the workspace with the new version. The resulting VCS diff is the authoritative old-to-new redline.

Inspect it with the command printed by Redline, typically one of:

```sh
jj diff
git diff --no-color
git diff --cached
```

## Current output

- `sections/<section-id>.md` — current/new accepted section text.
- `manifest.json` — deterministic section ordering with id, title, and heading level.
- `comments.md` — comment report. This file is always written, even when the Word document contains no comments.

No separate machine-readable change list is emitted in the current design. Tracked-change information belongs in the VCS diff.

## Comments

Comment bodies are read from Word's `word/comments.xml`; reply relationships are read from `word/commentsExtended.xml`. Comment ids are currently the ids emitted by Word. If Word starts ids at `1`, Redline preserves and reports those ids as-is.

Comment anchoring records the section, selected anchor text, and whether the selection came from normal, added, deleted, or mixed Word text. Redline does not add inline comment markers to section Markdown yet; that editor/plugin UX is orthogonal to the VCS-first diff workflow.

## Development

```sh
make test
make smoke
```

Smoke tests write reveal output to a temporary directory and do not modify `workspace/`.
