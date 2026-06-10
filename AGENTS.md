# Agent guidance for redline

## Scope

This file applies to the whole repository.

## Project shape

- This is a Go CLI project for converting reviewed `.docx` files into stable Markdown review workspaces.
- The current implemented phase is Phase 1: `redline reveal`.
- Keep changes small and aligned with `ROADMAP.md`; do not implement later phases unless explicitly requested.
- `ROADMAP.md` describes implementation direction and status; `BDD.md` describes behavior and test scenarios.

## Test workflow

- Run all tests with:

  ```sh
  make test
  ```

- Run the Phase 1 sample-document smoke test with:

  ```sh
  make smoke
  ```

- `workspace/sample.docx` is the representative Phase 1 integration fixture.
- Use `BDD.md` as the source of truth for behavior scenarios when adding or updating tests.
- Tests and smoke runs must write reveal output to a temporary directory, never back into `workspace/`.
- Do not compare generated VCS metadata such as `.git/` or `.jj/` in tests; compare Redline outputs only.

## Fixture expectations

- Keep DOCX fixtures small where possible for unit tests.
- Use XML snippets for parser unit tests when a full DOCX is unnecessary.
- Use `workspace/sample.docx` for end-to-end Phase 1 coverage only.
- Prefer deterministic output checks over broad visual/document-fidelity assertions.

## Coding notes

- Keep parser code pure where possible; perform filesystem and process work at CLI/workspace boundaries.
- Do not overwrite collaborator/source Word files.
- Current Phase 1 output consists of `sections/`, `manifest.json`, and `comments.md`.
