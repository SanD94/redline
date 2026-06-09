# ROADMAP

## Mission

Turn `.docx` review artifacts into clean, git-friendly Markdown outputs, then let those Markdown sources produce safe Word documents again.

Core must-have:

1. Extract tracked diff changes.
2. Extract text comments.
3. Split Word documents into title-wise Markdown, references, figures, and review metadata.
4. Rebuild or patch Word documents from Markdown sources and their assets.

Project goal: let teams keep writing and reviewing in Word while the source-of-truth workflow stays in Markdown + git.

Implementation goal: build the CLI in Go, using Pandoc where it is strong at document conversion and keeping custom Go code for DOCX review extraction, workspace state, diffing, and safe patch orchestration.

## CLI Shape

Redline should become a small command suite with two complementary directions:

1. **Word divide-and-conquer**: take a collaborator's `.docx`, extract structured Markdown plus review information, and verify what changed.
2. **Markdown merge**: combine Markdown sections, figures, references, and metadata back into a `.docx` deliverable.

Proposed action names:

1. `redline reveal <file.docx>` — split a Word file into title-wise content, figures, references, comments, and tracked changes. The redlines become visible and structured.
2. `redline audit` — compare the latest revealed Word content against the current Markdown sources, even when Word track changes were not enabled.
3. `redline imprint <file.docx>` — patch a copy of the received Word file using the Markdown sources, preserving as much useful Word structure as possible.
4. `redline disappear` — merge Markdown sources, figures, references, and metadata into a clean Word file. The redline workflow has done its job and vanishes into the final deliverable.

Supporting commands:

- `redline check` — report Go binary version, Pandoc path/version, config status, and whether the current project can run Pandoc-dependent commands.
- `redline pandoc -- <args>` — optional raw Pandoc pass-through for debugging and reproducibility.

Naming rule: commands should be short verbs that describe the redline lifecycle rather than implementation mechanics. `reveal` exposes Word review state, `audit` checks hidden or untracked differences, `imprint` writes Markdown decisions onto a Word copy, and `disappear` emits the clean `.docx` after review is complete.

## Product Shape

Inputs:

- `.docx` file with tracked changes
- `.docx` file with comments
- Markdown section files
- figure files and captions
- reference metadata or bibliography files
- optional previously revealed redline workspace

Outputs:

- normalized Markdown document
- title-wise Markdown section files
- extracted figures and references
- machine-readable change list
- machine-readable comment list
- optional merged review report
- patched copy of a received `.docx`
- clean generated `.docx` from Markdown sources

## Guiding Rules

- Preserve review intent, not Word layout noise.
- Prefer stable Markdown output over perfect visual reproduction.
- Use Pandoc for Markdown-to-DOCX generation before writing custom DOCX rendering code.
- Keep source mapping so user can trace Markdown item back to DOCX location.
- Make diffs deterministic so same input gives same output.
- Never overwrite a collaborator's original Word file; patch or generate copies only.
- Treat Markdown as the durable source of truth, but keep enough Word metadata to round-trip safely.
- References, figures, captions, and section boundaries are first-class content, not attachments after the fact.
- Keep the Go core independent from Pandoc-specific assumptions so extraction and audit still work when Pandoc is unavailable.

## Phase 1: `redline reveal` Smallest Useful Slice

Deliver working CLI that can divide a `.docx` into a redline workspace:

1. Read `.docx` archive.
2. Parse document XML.
3. Detect tracked insertions and deletions.
4. Detect comments and comment anchors.
5. Detect title or heading boundaries.
6. Extract body text into title-wise Markdown files.
7. Emit Markdown report plus machine-readable review data.

Success criteria:

- one sample file in, one redline workspace out
- insertions and deletions visible in output
- comments attached to nearby text
- sections split by title or heading
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

1. `sections/<slug>.md`
2. `document.md`
3. `changes.md`
4. `comments.md`
5. `review.json`
6. `figures/`
7. `references.*`

Suggested render rules:

- insertion: `[++ added text ++]`
- deletion: `[-- removed text --]`
- comment reference: inline marker like `[^c1]selected text[$c1]`
- comment body: footnote or review appendix

Need config for compact vs verbose mode.

## Phase 4: `redline audit` Safety Diff

Track changes may be missing or incomplete, so Redline needs a fallback comparison step.

Tasks:

1. Use the output of `redline reveal <file.docx>` as the Word-side snapshot.
2. Compare revealed current Word content against current Markdown sources.
3. Report differences by section with stable anchors where possible.
4. Distinguish tracked Word changes from inferred untracked changes.
5. Warn when a collaborator appears to have edited without enabling track changes.

Success criteria:

- user can detect untracked Word edits before accepting or overwriting anything
- audit output is readable in Markdown and machine-readable in JSON
- section-level matching survives minor heading or paragraph movement

## Phase 5: Better Diff Semantics

Word tracked changes often split text into many tiny XML runs. Need normalization layer.

Tasks:

1. Merge adjacent runs from same change.
2. Collapse formatting-only noise when possible.
3. Reconstruct sentence-level context around each change.
4. Distinguish insertion, deletion, move, replacement when data allows.

This phase matters because raw XML extraction alone will make ugly Markdown.

## Phase 6: Comment Accuracy

Comments often live in separate XML parts with range markers in main document.

Tasks:

1. Parse comment bodies from comments part.
2. Resolve `commentRangeStart` / `commentRangeEnd` anchors.
3. Fall back to nearby text when exact range broken.
4. Preserve author, timestamp, threaded reply data if present.

## Phase 7: `redline imprint` Patch Word Copy

After Markdown edits are reviewed, Redline should patch a copy of a received `.docx`.

Tasks:

1. Take a source `.docx` and current Markdown sources.
2. Create a new output `.docx`; never mutate the input file.
3. Replace section content from Markdown while preserving useful Word container structure.
4. Reattach figures, captions, references, and comments where supported.
5. Emit warnings for structures that cannot be preserved safely.

Success criteria:

- patched `.docx` opens in Word/LibreOffice
- original `.docx` remains untouched
- changed sections reflect Markdown content
- unsupported formatting loss is explicit, not silent

## Phase 8: `redline disappear` Build Clean Word From Markdown

Redline should also generate a clean Word document from Markdown sources, with Pandoc as the primary conversion engine.

Tasks:

1. Merge title-wise Markdown files in configured order.
2. Attach figures with captions and stable paths.
3. Include references from bibliography or extracted reference metadata.
4. Call Pandoc with a controlled set of arguments for `.docx` output.
5. Support a Pandoc reference document for Word styles when configured.
6. Produce a clean `.docx` without review markup by default.

Success criteria:

- Markdown project can produce a standalone `.docx`
- figures and references are included
- section order is deterministic
- output can be regenerated in CI
- missing Pandoc is reported as an actionable dependency error

## Phase 9: Nice Workflow

After core extraction works, add workflow helpers.

High-value add-ons:

1. Batch convert folder of `.docx` files.
2. Front matter metadata in Markdown.
3. Summary section with counts: insertions, deletions, comments.
4. Exit code or warning list for partially parsed files.
5. Optional HTML preview.

## Technical Plan

Language and runtime:

- Go is the implementation language for the CLI, core model, DOCX archive/XML parsing, workspace management, diffing, and command orchestration.
- Pandoc is an external dependency for high-quality Markdown-to-DOCX conversion, especially for `redline disappear` and possibly parts of `redline imprint`.
- The CLI should detect Pandoc availability with `pandoc --version` and produce clear errors when commands need it.
- Quarto CLI is useful as an architecture reference, but Redline should stay much smaller: no execution engines, no notebooks, no website/project publishing layer, and no large extension system.

Suggested structure:

1. `cmd/redline` — Go command entrypoint
2. `internal/docx` — unzip DOCX parts and expose WordprocessingML files
3. `internal/wordxml` — parse WordprocessingML
4. `internal/model` — normalized review objects
5. `internal/workspace` — redline workspace layout and source mapping
6. `internal/render/markdown` — Markdown output
7. `internal/render/json` — structured output
8. `internal/pandoc` — Pandoc discovery and invocation wrapper
9. `internal/patchdocx` — safe DOCX-copy patching
10. `internal/diff` — tracked and inferred difference model
11. `internal/cli` — command wiring and user-facing errors

Keep parser pure where possible. IO only at edges.

Quarto-inspired patterns to copy:

1. Walk up from the input path to discover `redline.yaml`; otherwise create a synthetic single-file project context.
2. Use one session temp directory per CLI invocation, then command-specific temp subdirectories.
3. Keep a `--debug` or `--keep-temp` flag that preserves intermediate Pandoc input, defaults, metadata, and output files.
4. Put all child process execution behind one Go wrapper that handles stdout/stderr, cancellation, timeout, cleanup, and debug logging.
5. Prefer Pandoc defaults YAML files over very long command lines.
6. Allow `REDLINE_PANDOC` to override the Pandoc binary path, then fall back to bundled or `$PATH` Pandoc.
7. Use an output recipe: Pandoc may write to a temp `.docx`, then Redline validates and moves it to the final destination.
8. Add `redline check --json` for CI-friendly dependency diagnostics.

Pandoc responsibilities:

1. Convert merged Markdown to clean `.docx` for `redline disappear`.
2. Apply reference-doc styles when requested.
3. Handle common Markdown constructs, figures, tables, citations, and bibliography rendering where possible.

Go responsibilities:

1. Extract tracked changes and comments from DOCX XML because Pandoc does not preserve enough review metadata for Redline's core purpose.
2. Split revealed Word content into stable section files.
3. Maintain source maps, review JSON, warnings, and deterministic workspace output.
4. Compare Word snapshots against Markdown sources.
5. Decide when to call Pandoc and validate the files it produces.

## Pandoc-Assisted Extraction Plan

Pandoc can help with DOCX extraction, but it should not be the only source of truth for review metadata. Use it as a normalization engine, then augment or verify with custom Go XML parsing.

### `redline reveal` extraction pipeline

1. Create a temp context and extract the DOCX archive enough to inspect `word/document.xml`, relationships, media, styles, and comments.
2. Run Pandoc in accepted-content mode:

   ```sh
   pandoc input.docx \
     --from docx+styles \
     --to markdown \
     --track-changes=accept \
     --extract-media <workspace>/figures \
     --output <temp>/accepted.md
   ```

   This gives Redline the collaborator's current visible document text, extracted media, and style hints.

3. Run Pandoc in review-preserving mode:

   ```sh
   pandoc input.docx \
     --from docx+styles \
     --to markdown \
     --track-changes=all \
     --extract-media <workspace>/figures \
     --output <temp>/review.md
   ```

   Pandoc emits insertions, deletions, comments, and paragraph insert/delete markers as spans/classes. Redline can translate these into its own Markdown review syntax and `review.json`.

4. Optionally run Pandoc to JSON for a more structured intermediate form when Markdown spans are too fragile:

   ```sh
   pandoc input.docx \
     --from docx+styles \
     --to json \
     --track-changes=all \
     --extract-media <workspace>/figures \
     --output <temp>/review.ast.json
   ```

   The Go CLI can parse Pandoc's AST JSON to find `Span` and `Div` classes such as `insertion`, `deletion`, `comment-start`, `comment-end`, `paragraph-insertion`, and `paragraph-deletion`.

5. Parse DOCX XML directly in Go to recover data Pandoc may flatten or omit:

   - exact comment bodies from `word/comments.xml`
   - comment range ids and anchors from `word/document.xml`
   - relationship ids for media and hyperlinks
   - raw Word XML pointers for source maps
   - author and timestamp fields on change elements

6. Reconcile the Pandoc output and Go XML model into Redline's normalized model.
7. Split accepted content into `sections/<slug>.md` using heading/title boundaries.
8. Write `document.md`, `changes.md`, `comments.md`, `review.json`, `source-map.json`, extracted `figures/`, and warnings.

### Why combine Pandoc and custom XML parsing?

- Pandoc is good at turning messy DOCX body content into readable Markdown and extracting media.
- Pandoc's `--track-changes=all` is useful for scripting review markup, but its output is not a complete source map back to Word XML.
- Custom Go XML parsing is necessary for deterministic anchors, exact comments, safer audit diffs, and future DOCX patching.
- The accepted-content Pandoc output is ideal for `redline audit`, because it represents what a collaborator would see after accepting tracked changes.

### Pandoc defaults files

Redline should generate Pandoc defaults files instead of building long command strings. Example for `redline disappear`:

```yaml
from: markdown+pipe_tables+footnotes+citations
to: docx
standalone: true
output-file: /path/to/output.docx
extract-media: figures
reference-doc: /path/to/reference.docx
citeproc: true
bibliography: references.bib
metadata-file: /tmp/redline-metadata.yml
```

For extraction commands, defaults can include `from: docx+styles`, `to: markdown` or `to: json`, `track-changes: all`, and `extract-media: <dir>`.

### Minimum Pandoc experiments before implementation

Create fixtures and record Pandoc outputs for:

1. DOCX with insertion, deletion, replacement, and comments using `--track-changes=all` to Markdown.
2. Same file using `--track-changes=accept` to Markdown.
3. Same file using `--track-changes=all --to json`.
4. DOCX with custom Word styles using `--from docx+styles`.
5. DOCX with figures and captions using `--extract-media`.
6. Markdown with figures, citations, and a reference doc converted back to `.docx`.

These experiments define which parts Pandoc handles reliably and which parts Redline must own in Go.

## Parsing Priorities

Read these DOCX parts first:

1. `word/document.xml`
2. `word/comments.xml`
3. `word/_rels/document.xml.rels`
4. optional header/footer parts later
5. media relationships for figures
6. numbering/styles later for better Markdown and DOCX output

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
5. Untracked collaborator edits may look like intentional Markdown divergence.
6. Full DOCX round-trip fidelity is hard; patching must be conservative.
7. Reference and figure handling can become brittle without stable IDs.
8. Pandoc output may differ across versions.
9. Users may not have Pandoc installed locally or in CI.

Mitigation:

- start with body text only
- keep fixture corpus early
- snapshot expected Markdown outputs
- record Pandoc version in generated metadata when Pandoc is used
- provide a documented install check for Pandoc-dependent commands

## Testing Plan

Need small but sharp fixture set:

1. insertion only
2. deletion only
3. replacement via delete + insert
4. single comment on word
5. comment on paragraph range
6. nested or adjacent changes
7. document with tables
8. document with headings split into multiple Markdown files
9. document with embedded figures and captions
10. document with references or bibliography section
11. document edited without track changes
12. Markdown project that builds a clean `.docx`
13. Markdown project built with a Pandoc reference document
14. Pandoc-missing command failure path

Best regression test: same `.docx` fixture always renders same Markdown.

For generated `.docx` outputs, prefer structural checks over byte-for-byte comparisons because Pandoc-created archives may contain timestamps, relationship ids, or version-dependent details.

## Definition Of Done

Project first real release ready when:

1. CLI converts representative `.docx` files to Markdown.
2. tracked changes readable in git diff.
3. comments preserved with clear anchors.
4. failures reported clearly when document uses unsupported structures.
5. outputs deterministic enough for CI and version control.
6. CLI can audit Word-vs-Markdown differences when track changes are missing.
7. CLI can create a patched copy of a received Word file from Markdown sources.
8. CLI can generate a clean Word file from Markdown sources, figures, and references.
9. Go CLI reports Pandoc dependency status clearly for commands that need it.

## Stretch Ideas

1. round-trip annotations back into DOCX
2. GitHub PR comment export
3. change severity tagging
4. LLM summary of document review activity
5. citation processor integration for BibTeX/CSL workflows beyond Pandoc defaults
6. embedded Pandoc distribution or managed installer helper
