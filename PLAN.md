# PLAN

## What This Project Does

This project turns Word review work into Markdown files safe for git.

People can keep using `.docx`.
Engineering can still review change history in `.md`.

## Why Build It

Two big needs drive whole thing:

1. collaborators use Word, not Markdown
2. you want versioned, diff-friendly workflow in git

So project acts like bridge:

- Word stays input format for collaborators
- Markdown becomes source of truth for review tracking

## Essential Features

### 1. Diff Changes

Extract tracked changes from `.docx`.

Need to show:

- inserted text
- deleted text
- nearby context
- author and time when available

Result: git can show meaningful content changes without forcing team to stop using Word.

### 2. Comments For Text

Extract review comments tied to text ranges.

Need to show:

- comment text
- target text or nearby anchor
- author and time when available
- reply thread if present later

Result: review conversation no longer trapped inside Word file.

## Nice Extra Features

Useful after core works:

1. Markdown summary page with all changes and comments
2. JSON export for automation
3. batch conversion for many files
4. per-section change summary
5. warning report for unsupported DOCX structures
6. optional HTML preview for non-technical reviewers

## Expected Workflow

1. collaborator edits `.docx` with tracked changes and comments
2. tool reads file
3. tool extracts review data
4. tool writes Markdown outputs
5. team stores Markdown in git
6. diffs, history, and automation happen in normal dev workflow

## What User Gets

Main outputs should look like this:

1. `document.md` — readable document with inline change markers
2. `changes.md` — list of tracked edits
3. `comments.md` — list of comments tied to text
4. `review.json` — structured data for scripts or later UI

## First Version Scope

Keep v1 narrow.

Must support:

1. normal body text
2. tracked insertions
3. tracked deletions
4. basic comments on text ranges

Not necessary as the first author handles:

1. tables
2. footnotes
3. headers and footers
4. images
5. complex layout fidelity

## Success Looks Like This

Project succeeds when:

1. non-Markdown collaborators change nothing about their process
2. engineering gets stable Markdown artifacts in git
3. tracked edits become readable in git diff
4. comments become searchable and reviewable outside Word

## Recommended Rollout

1. build small CLI proof of concept
2. test on few real `.docx` files
3. tune Markdown format until git diffs feel clean
4. add metadata and batch processing
5. decide later if web UI or HTML preview worth it

## Bottom Line

Core product idea simple:

Word in.
Markdown out.
Tracked changes and comments survive trip.

That gives mixed teams one workflow instead of two disconnected ones.
