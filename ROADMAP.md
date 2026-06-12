# ROADMAP

## Mission

Turn `.docx` review artifacts into clean, git-friendly Markdown outputs, then let those Markdown sources produce safe Word documents again.

Core must-have:

1. Extract tracked diff changes.
2. Extract text comments.
3. Split Word documents into title-wise Markdown, references, figures, comments, and source metadata.
4. Rebuild or patch Word documents from Markdown sources and their assets.

Project goal: let teams keep writing and reviewing in Word while the source-of-truth workflow stays in Markdown plus the local VCS.

Implementation goal: build the CLI in Go, using Pandoc where it is strong at document conversion, and keeping custom Go code for DOCX review extraction, workspace state, VCS orchestration, and safe patch orchestration. Redline should not duplicate the VCS diff system with a separate tracked-change diff format.

## CLI Shape

Redline should become a small command suite with two complementary directions:

1. **Word divide-and-conquer**: take a collaborator's `.docx`, extract structured Markdown plus review information, and verify what changed.
2. **Markdown merge**: combine Markdown sections, figures, references, and metadata back into a `.docx` deliverable.

Proposed action names:

1. `redline reveal <file.docx>` — split a Word file into title-wise content, figures, references, comments, tracked changes, and a small review-intent sidecar for Word actions that VCS cannot disambiguate. The redlines become visible and structured.
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
- VCS diff between old and new revealed document states
- Markdown comment report
- optional merged review report
- patched copy of a received `.docx`
- clean generated `.docx` from Markdown sources

## Guiding Rules

- Preserve review intent, not Word layout noise.
- Treat `jj` or `git` as a first-class hard dependency for review diffs.
- Prefer stable Markdown output over perfect visual reproduction.
- Use Pandoc for Markdown-to-DOCX generation before writing custom DOCX rendering code.
- Keep source mapping so user can trace Markdown item back to DOCX location.
- Make diffs deterministic so same input gives same output.
- Never overwrite a collaborator's original Word file; patch or generate copies only.
- Treat Markdown as the durable source of truth, but keep enough Word metadata to round-trip safely.
- References, figures, captions, and section boundaries are first-class content, not attachments after the fact.
- Keep the Go core independent from Pandoc-specific assumptions where practical, while treating VCS and Pandoc as system tools Redline is allowed to rely on.

## Phase 1: `redline reveal` Smallest Useful Slice

Deliver working CLI that can divide a `.docx` into a redline workspace:

1. Read `.docx` archive.
2. Parse document XML.
3. Detect tracked insertions and deletions.
4. Detect comments and comment anchors.
5. Detect title or heading boundaries.
6. Extract body text into title-wise Markdown files.
7. Emit Markdown outputs and rely on the VCS diff for ordinary tracked insertion/deletion effects.
8. Emit a small review-intent sidecar for explicit Word review actions that outcome diffs cannot attribute safely, starting with `w:moveFrom`/`w:moveTo` pairs.

Success criteria:

- one sample file in, one redline workspace out
- insertions and deletions visible in output
- explicit Word moves preserved as metadata rather than inferred from the VCS diff
- comments attached at least to nearby sections
- sections split by title or heading
- output stable across repeated runs

## Phase 2: Reliable Comment And Source Model

Build a small internal model for things VCS cannot represent by itself: comments, anchors, sections, source pointers, warnings, and output layout. Do not build a parallel tracked-change diff model.

Suggested entities:

- `DocumentBlock`
- `TextRun`
- `Comment`
- `AnchorRange`
- `ReviewAction` / `Move`
- `Warning`

Each extracted item should carry:

- stable id
- type
- author if present
- timestamp if present
- raw DOCX xml pointer or path
- surrounding text context

Reason: comments, explicit moves, and source pointers need deterministic identity, while ordinary insertions and deletions should still be expressed by the VCS diff between old and new workspace snapshots. A reorder such as `X Y` to `Y X` is outcome-equivalent to moving either item, so Redline must preserve Word's explicit move intent when Word provides it instead of asking VCS to infer it.

## Phase 3: Markdown-First Output

Primary export formats:

1. `sections/<slug>.md`
2. `document.md`
3. `comments.md`
4. `figures/`
5. `references.*`
6. optional warning/source metadata for unsupported structures

Suggested render rules:

- Do not render tracked insertions/deletions with a Redline-specific inline syntax by default.
- Render old and new document states as normal Markdown and let `jj diff` or `git diff` show changes.
- Render comment bodies and anchor metadata in `comments.md`.
- Render explicit Word actions that VCS cannot disambiguate in sidecar metadata such as `review-intent.json`, not inline section Markdown.
- Inline comment markers are orthogonal to the VCS-first reveal flow and can be added later for editor/plugin UX.

Need config for compact vs verbose mode.

## Phase 4: Better VCS Diff Quality

Word tracked changes often split text into many tiny XML runs. Redline should improve the old/new Markdown snapshots so the VCS diff is readable, rather than creating its own diff engine.

Tasks:

1. Normalize whitespace and paragraph boundaries before writing snapshots.
2. Collapse formatting-only noise when possible.
3. Keep sentence and paragraph context stable so VCS hunks are readable.
4. Represent moves by stable section files and normal VCS moved-line detection where possible.

This phase matters because raw XML extraction alone will make ugly Markdown.

## Phase 5: Comment Accuracy

Comments often live in separate XML parts with range markers in main document.

Tasks:

1. Parse comment bodies from comments part.
2. Resolve `commentRangeStart` / `commentRangeEnd` anchors.
3. Fall back to nearby text when exact range broken.
4. Preserve author, timestamp, threaded reply data if present.

## Phase 6: `redline imprint` Patch Word Copy

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

## Phase 7: `redline disappear` Build Clean Word From Markdown

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

## Phase 8: Nice Workflow

After core extraction works, add workflow helpers.

High-value add-ons:

1. Batch convert folder of `.docx` files.
2. Front matter metadata in Markdown.
3. Summary section with counts: insertions, deletions, comments.
4. Exit code or warning list for partially parsed files.
5. Optional HTML preview.

## Technical Plan

Language and runtime:

- Go is the implementation language for the CLI, core model, DOCX archive/XML parsing, workspace management, VCS orchestration, and command wiring.
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
7. `internal/render/json` — optional diagnostics/status output, not tracked-change diffs
8. `internal/pandoc` — Pandoc discovery and invocation wrapper
9. `internal/patchdocx` — safe DOCX-copy patching
10. `internal/cli` — command wiring and user-facing errors

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

1. Extract old and new text states plus comments from DOCX XML because Pandoc does not preserve enough comment and anchor metadata for Redline's core purpose.
2. Split revealed Word content into stable section files.
3. Maintain comment anchors, warnings, and deterministic workspace output.
4. Orchestrate VCS snapshots and point users to reproducible VCS diff commands.
5. Decide when to call Pandoc and validate the files it produces.

## Pandoc-Assisted Extraction Plan

Pandoc can help with DOCX extraction, but it should not become a second review-diff system. Use it as a conversion/normalization engine where it improves Markdown quality, then rely on VCS for old/new diffs and custom Go XML parsing for comments and anchors.

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

3. Optionally run Pandoc in rejected-content mode when it can produce better old snapshots than the Go text extractor:

   ```sh
   pandoc input.docx \
     --from docx+styles \
     --to markdown \
     --track-changes=reject \
     --extract-media <workspace>/figures \
     --output <temp>/old.md
   ```

   Redline can snapshot this old state and then write the accepted state. The VCS diff remains the review surface.

4. Optionally run Pandoc to JSON for a more structured content intermediate form when Markdown output is too fragile:

   ```sh
   pandoc input.docx \
     --from docx+styles \
     --to json \
     --track-changes=accept \
     --extract-media <workspace>/figures \
     --output <temp>/review.ast.json
   ```

   The Go CLI can parse Pandoc's AST JSON for document structure, not as the authoritative tracked-change diff.

5. Parse DOCX XML directly in Go to recover data Pandoc may flatten or omit:

   - exact comment bodies from `word/comments.xml`
   - comment range ids and anchors from `word/document.xml`
   - relationship ids for media and hyperlinks
   - raw Word XML pointers for source maps
   - enough tracked-change structure to reconstruct old and accepted text states

6. Reconcile the Pandoc output and Go XML model into stable old/new Markdown snapshots plus comment metadata.
7. Split accepted content into `sections/<slug>.md` using heading/title boundaries.
8. Snapshot old output with VCS, write accepted output, then write `document.md`, `comments.md`, extracted `figures/`, and warnings/diagnostics as needed.

### Why combine Pandoc and custom XML parsing?

- Pandoc is good at turning messy DOCX body content into readable Markdown and extracting media.
- Pandoc's accepted/rejected tracked-change modes can help produce clean old/new snapshots, but Redline should not translate them into its own diff notation.
- Custom Go XML parsing is necessary for deterministic anchors, exact comments, safe snapshot construction, and future DOCX patching.
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

For extraction commands, defaults can include `from: docx+styles`, `to: markdown` or `to: json`, `track-changes: accept` or `track-changes: reject`, and `extract-media: <dir>`.

### Minimum Pandoc experiments before implementation

Create fixtures and record Pandoc outputs for:

1. DOCX with insertion, deletion, replacement, and comments using `--track-changes=accept` to Markdown.
2. Same file using `--track-changes=reject` to Markdown.
3. Same file using `--track-changes=accept --to json` and `--track-changes=reject --to json`.
4. DOCX with custom Word styles using `--from docx+styles`.
5. DOCX with figures and captions using `--extract-media`.
6. Markdown with figures, citations, and a reference doc converted back to `.docx`.

These experiments define which parts Pandoc handles reliably and which parts Redline must own in Go.

## Risks

1. Word XML noisy; naive extraction will fragment text.
2. Comments may anchor to ranges crossing multiple runs.
3. Tables, footnotes, headers may hide comments or content that should appear in VCS snapshots.
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

## Definition Of Done

Project first real release ready when:

1. CLI converts representative `.docx` files to Markdown.
2. tracked changes readable in VCS diff.
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
