# ROADMAP

## Mission

Turn `.docx` review artifacts into clean, git-friendly Markdown outputs.

Core must-have:

1. Extract tracked diff changes.
2. Extract text comments.

Project goal: let teams keep writing in Word while engineering workflow stays in Markdown + git.

## Product Shape

Input:

- `.docx` file with tracked changes
- `.docx` file with comments

Output:

- normalized Markdown document
- machine-readable change list
- machine-readable comment list
- optional merged review report

## Guiding Rules

- Preserve review intent, not Word layout noise.
- Prefer stable Markdown output over perfect visual reproduction.
- Keep source mapping so user can trace Markdown item back to DOCX location.
- Make diffs deterministic so same input gives same output.

## Phase 1: Smallest Useful Slice

Deliver working CLI:

1. Read `.docx` archive.
2. Parse document XML.
3. Detect tracked insertions and deletions.
4. Detect comments and comment anchors.
5. Emit Markdown report.

Success criteria:

- one sample file in, one Markdown file out
- insertions and deletions visible in output
- comments attached to nearby text
- output stable across repeated runs

## Phase 2: Reliable Review Model

Build internal normalized model before richer output.

Suggested entities:

- `DocumentBlock`
- `TextRun`
- `Change`
- `Comment`
- `AnchorRange`

Each extracted item should carry:

- stable id
- type
- author if present
- timestamp if present
- raw DOCX xml pointer or path
- surrounding text context

Reason: once model stable, Markdown/JSON/HTML exports become easy.

## Phase 3: Markdown-First Output

Primary export formats:

1. `document.md`
2. `changes.md`
3. `comments.md`
4. `review.json`

Suggested render rules:

- insertion: `[++ added text ++]`
- deletion: `[-- removed text --]`
- comment reference: inline marker like `[^c1]selected text[$c1]`
- comment body: footnote or review appendix

Need config for compact vs verbose mode.

## Phase 4: Better Diff Semantics

Word tracked changes often split text into many tiny XML runs. Need normalization layer.

Tasks:

1. Merge adjacent runs from same change.
2. Collapse formatting-only noise when possible.
3. Reconstruct sentence-level context around each change.
4. Distinguish insertion, deletion, move, replacement when data allows.

This phase matters because raw XML extraction alone will make ugly Markdown.

## Phase 5: Comment Accuracy

Comments often live in separate XML parts with range markers in main document.

Tasks:

1. Parse comment bodies from comments part.
2. Resolve `commentRangeStart` / `commentRangeEnd` anchors.
3. Fall back to nearby text when exact range broken.
4. Preserve author, timestamp, threaded reply data if present.

## Phase 6: Nice Workflow

After core extraction works, add workflow helpers.

High-value add-ons:

1. Batch convert folder of `.docx` files.
2. Front matter metadata in Markdown.
3. Summary section with counts: insertions, deletions, comments.
4. Exit code or warning list for partially parsed files.
5. Optional HTML preview.

## Technical Plan

Suggested structure:

1. `parser/zip` — unzip DOCX parts
2. `parser/xml` — parse WordprocessingML
3. `model` — normalized review objects
4. `render/markdown` — Markdown output
5. `render/json` — structured output
6. `cli` — command entrypoint

Keep parser pure where possible. IO only at edges.

## Parsing Priorities

Read these DOCX parts first:

1. `word/document.xml`
2. `word/comments.xml`
3. `word/_rels/document.xml.rels`
4. optional header/footer parts later

WordprocessingML elements likely needed:

- `w:ins`
- `w:del`
- `w:commentRangeStart`
- `w:commentRangeEnd`
- `w:commentReference`
- `w:r`
- `w:t`
- `w:p`

## Risks

1. Word XML noisy; naive extraction will fragment text.
2. Comments may anchor to ranges crossing multiple runs.
3. Tables, footnotes, headers may hide review data.
4. Different Word producers may emit slightly different XML.

Mitigation:

- start with body text only
- keep fixture corpus early
- snapshot expected Markdown outputs

## Testing Plan

Need small but sharp fixture set:

1. insertion only
2. deletion only
3. replacement via delete + insert
4. single comment on word
5. comment on paragraph range
6. nested or adjacent changes
7. document with tables

Best regression test: same `.docx` fixture always renders same Markdown.

## Definition Of Done

Project first real release ready when:

1. CLI converts representative `.docx` files to Markdown.
2. tracked changes readable in git diff.
3. comments preserved with clear anchors.
4. failures reported clearly when document uses unsupported structures.
5. outputs deterministic enough for CI and version control.

## Stretch Ideas

1. round-trip annotations back into DOCX
2. GitHub PR comment export
3. change severity tagging
4. LLM summary of document review activity
5. Neovim plugin to show what's added/edited/deleted
