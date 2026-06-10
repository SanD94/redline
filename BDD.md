# Redline BDD Specifications

This file defines behavior in Gherkin style so unit tests and integration tests can be implemented from shared product expectations.

## Test commands

```sh
make test
make smoke
```

## Phase 1: reveal DOCX review state

Feature: Reveal a reviewed DOCX as a Markdown workspace

  Redline should expose Word review changes and comments as deterministic Markdown files
  so review artifacts can be inspected in the local VCS.

  Background:
    Given a DOCX file contains `word/document.xml`
    And the reveal command writes output to a temporary workspace

  Scenario: A DOCX archive exposes its core WordprocessingML parts
    Given a DOCX archive contains `word/document.xml`
    And the DOCX archive contains `word/comments.xml`
    And the DOCX archive contains `word/commentsExtended.xml`
    And the DOCX archive contains `word/styles.xml`
    When the DOCX reader opens the archive
    Then the document XML is available to the parser
    And comments XML is available to the parser
    And extended comments XML is available to the parser
    And styles XML is available to the parser

  Scenario: A malformed DOCX without document XML is rejected
    Given a DOCX archive does not contain `word/document.xml`
    When the DOCX reader opens the archive
    Then opening fails with a missing-document error

  Scenario: Old text rejects insertions and keeps deletions
    Given a paragraph contains regular text `Before `
    And the paragraph contains a tracked deletion `old`
    And the paragraph contains a tracked insertion `new`
    And the paragraph contains regular text ` after.`
    When the Word XML parser extracts the old version
    Then the paragraph text is `Before old after.`

  Scenario: New text accepts insertions and rejects deletions
    Given a paragraph contains regular text `Before `
    And the paragraph contains a tracked deletion `old`
    And the paragraph contains a tracked insertion `new`
    And the paragraph contains regular text ` after.`
    When the Word XML parser extracts the new version
    Then the paragraph text is `Before new after.`

  Scenario: Heading styles split content into sections
    Given `word/styles.xml` maps style `TitleStyle` to canonical name `heading 1`
    And `word/styles.xml` maps style `SubheadingStyle` to canonical name `heading 2`
    And the document contains a `TitleStyle` paragraph `Results`
    And the document contains a body paragraph `Result body.`
    And the document contains a `SubheadingStyle` paragraph `Sub result`
    When the Word XML parser extracts the document
    Then one section is produced with id `results`
    And the section content contains `Result body.`
    And the section content contains a Markdown level-2 heading `## Sub result`

  Scenario: Comment bodies are extracted with author and date
    Given `word/comments.xml` contains comment id `1`
    And comment id `1` has author `Reviewer`
    And comment id `1` has date `2026-06-09T13:01:00Z`
    And comment id `1` has body text `Nice comment.`
    When the Word XML parser extracts comments
    Then the result contains comment id `1`
    And the comment author is `Reviewer`
    And the comment date is `2026-06-09T13:01:00Z`
    And the comment text is `Nice comment.`

  Scenario: Comment replies are linked to their parent comment
    Given `word/comments.xml` contains parent comment id `1` with paragraph id `parent`
    And `word/comments.xml` contains reply comment id `2` with paragraph id `child`
    And `word/commentsExtended.xml` maps paragraph id `child` to parent paragraph id `parent`
    When the Word XML parser extracts comments
    Then comment id `2` has parent id `1`

  Scenario: Comment anchors are assigned to the nearest section
    Given the document has a level-1 heading `Discussion`
    And a later paragraph contains `w:commentRangeStart` and `w:commentRangeEnd` for comment id `1` around text `Commented text.`
    And `word/comments.xml` contains comment id `1`
    When the Word XML parser extracts the document
    Then comment id `1` has section id `discussion`
    And comment id `1` has anchor kind `normal`
    And comment id `1` has anchor text `Commented text.`

  Scenario: Comment anchors can target normal, added, or deleted text
    Given a paragraph contains a normal-text comment range for comment id `1`
    And the paragraph contains an inserted-text comment range for comment id `2`
    And the paragraph contains a deleted-text comment range for comment id `3`
    And `word/comments.xml` contains comment ids `1`, `2`, and `3`
    When the Word XML parser extracts the new version
    Then comment id `1` has anchor kind `normal`
    And comment id `2` has anchor kind `added`
    And comment id `3` has anchor kind `deleted`
    And all three comments preserve their selected anchor text

  Scenario: A section renders as Markdown
    Given a section has title `Introduction`
    And the section level is `1`
    And the section content is `First paragraph.`
    When the Markdown renderer renders the section
    Then the output is:
      """
      # Introduction

      First paragraph.
      """

  Scenario: Comments render as a Markdown report
    Given a comment has id `4`
    And the comment author is `Reviewer`
    And the comment date is `2026-06-09T13:01:00Z`
    And the comment section id is `discussion`
    And the comment anchor kind is `normal`
    And the comment anchor text is `commented text`
    And the comment text is `Nice comment.`
    When the Markdown renderer renders comments
    Then the output contains `## Comment 4`
    And the output contains `- **Author:** Reviewer`
    And the output contains `- **Date:** 2026-06-09T13:01:00Z`
    And the output contains this section line:
      """
      - **Section:** `discussion`
      """
    And the output contains `- **Anchor kind:** normal`
    And the output contains `- **Anchor text:** commented text`
    And the output contains `- **Text:** Nice comment.`

  Scenario: Comments report exists even when no comments exist
    Given a DOCX contains no comments
    When `redline reveal` writes a temporary workspace
    Then the workspace contains `comments.md`
    And `comments.md` contains `No comments found.`

  Scenario: The sample DOCX produces the current Phase 1 workspace
    Given the fixture `workspace/sample.docx`
    When `redline reveal workspace/sample.docx` writes to a temporary workspace
    Then the workspace contains `sections/`
    And the workspace contains `manifest.json`
    And the workspace contains `comments.md`
    And the manifest lists 11 sections
    And the manifest includes section ids in this order:
      | id                                  |
      | a-very-nice-big-title-for-the-study |
      | abstract                            |
      | introduction                        |
      | results                             |
      | discussion                          |
      | methods                             |
      | data-availability                   |
      | references                          |
      | acknowledgements                    |
      | author-contributions                |
      | additional-information              |
    And `comments.md` contains an anchor kind `normal`
    And `comments.md` contains an anchor kind `added`
    And `comments.md` contains an anchor kind `deleted`
    And `comments.md` contains `- **Text:** Nice comment on deleted text.`
    And `comments.md` contains `- **Text:** Nice comment on added text.`
    And `comments.md` contains `- **Text:** Nice comment.`
    And `comments.md` contains `- **Text:** Comment within a comment`
    And `comments.md` contains `- **Text:** Isn’t it?`
    And `comments.md` contains `- **Text:** Done.`

  Scenario: Reveal output is deterministic for the sample DOCX
    Given the fixture `workspace/sample.docx`
    When `redline reveal workspace/sample.docx` writes to temporary workspace `A`
    And `redline reveal workspace/sample.docx` writes to temporary workspace `B`
    Then `A/manifest.json` equals `B/manifest.json`
    And `A/comments.md` equals `B/comments.md`
    And every file under `A/sections/` equals the matching file under `B/sections/`
    And generated VCS metadata is ignored

## Planned scenarios from the roadmap

These scenarios describe planned coverage and should be implemented only when the corresponding feature exists. They are intentionally broader than the current test suite so future work can be added safely without rediscovering expected behavior.

## Phase 2: reliable comment and source model

Feature: Normalize DOCX comments and source data into a stable internal model

  Scenario: A document block carries stable source identity
    Given a DOCX paragraph appears in `word/document.xml`
    And the paragraph belongs to section `introduction`
    When Redline builds the normalized comment and source model
    Then the model contains a document block for the paragraph
    And the document block has a stable id
    And the document block has type `paragraph`
    And the document block has a source pointer into `word/document.xml`
    And the document block records surrounding text context

  Scenario: Old and new text states are normalized for VCS diffing
    Given a DOCX paragraph contains a tracked insertion
    And the same paragraph contains a tracked deletion
    When Redline builds old and new text states
    Then the old version rejects the insertion and keeps the deletion
    And the new version keeps the insertion and rejects the deletion
    And no separate Redline change entity is required for the VCS diff

  Scenario: A comment is normalized with its anchor range
    Given `word/comments.xml` contains a comment body
    And `word/document.xml` contains matching comment range markers
    When Redline builds the normalized comment and source model
    Then the model contains a comment with a stable id
    And the comment preserves author and timestamp when present
    And the comment references an anchor range
    And the anchor range includes nearby selected text
    And the anchor range records whether the selected text is `normal`, `added`, `deleted`, or `mixed`
    And the comment has a raw DOCX XML pointer

  Scenario: Model output is deterministic across repeated parses
    Given the same DOCX fixture is parsed twice
    When Redline builds normalized comment and source models for both parses
    Then the model ids are stable across both parses
    And the section ordering is stable across both parses
    And the serialized comment and source data is equal across both parses

## Phase 3: Markdown-first output

Feature: Write reveal outputs as durable Markdown artifacts plus VCS state

  Scenario: Reveal writes a complete Markdown-first workspace
    Given a DOCX fixture contains headings, body text, tracked changes, and comments
    When `redline reveal` processes the fixture
    Then the workspace contains `sections/`
    And the workspace contains `document.md`
    And the workspace contains `comments.md`
    And tracked changes are visible through the workspace VCS diff
    And every generated text artifact is deterministic

  Scenario: Section files are written using stable slugs
    Given a DOCX fixture has headings `Introduction` and `Data Availability`
    When `redline reveal` processes the fixture
    Then the workspace contains `sections/introduction.md`
    And the workspace contains `sections/data-availability.md`
    And `manifest.json` records section order

  Scenario: The full document Markdown is assembled from sections in manifest order
    Given a reveal workspace contains ordered section files
    When Redline writes `document.md`
    Then `document.md` contains the sections in `manifest.json` order
    And `document.md` contains no generated VCS metadata

  Scenario: Insertions and deletions are shown by VCS diff
    Given a reveal workspace contains an old-version VCS snapshot
    And the current section files contain the new accepted version
    When the user runs `jj diff` or `git diff`
    Then inserted and deleted text are visible in the VCS diff
    And Redline does not require its own inline insertion/deletion markup

  Scenario: Comment anchors render as report metadata without inline markers
    Given a normalized comment is anchored to text `selected text`
    And the comment id is `c1`
    And the anchor kind is `added`
    When Redline renders `comments.md`
    Then `comments.md` contains the anchor text `selected text`
    And `comments.md` contains the anchor kind `added`
    And `comments.md` contains the body for comment `c1`
    And section Markdown does not need an inline marker for comment `c1`

  Scenario: Figures are extracted as first-class assets
    Given a DOCX fixture contains an embedded figure with a caption
    When `redline reveal` processes the fixture
    Then the figure file is written under `figures/`
    And Markdown content references the figure with a stable relative path
    And the caption remains associated with the figure
  Scenario: References are extracted as first-class content
    Given a DOCX fixture contains a references section or bibliography metadata
    When `redline reveal` processes the fixture
    Then the workspace contains a `references.*` artifact when extractable
    And the references remain ordered deterministically
    And unsupported reference structures produce explicit warnings

## Phase 4: audit safety diff

Feature: Audit revealed Word content against Markdown sources

  Scenario: Audit reports no differences when Markdown matches revealed Word text
    Given a reveal workspace was created from a DOCX file
    And the Markdown section files have not changed since reveal
    When `redline audit` runs in the workspace
    Then the audit report says no content differences were found
    And the command exits successfully

  Scenario: Audit reports section-level differences
    Given a reveal workspace was created from a DOCX file
    And `sections/introduction.md` was edited after reveal
    When `redline audit` runs in the workspace
    Then the audit report lists section `introduction`
    And the report shows the changed text
    And the report points to a reproducible VCS diff command
    And the command exits with a difference status for automation

  Scenario: Audit distinguishes reveal diffs from source divergence
    Given a reveal workspace contains tracked Word changes
    And the current Markdown contains an additional edit not represented by tracked Word changes
    When `redline audit` runs in the workspace
    Then the normal reveal VCS diff remains available as old-to-new Word review
    And the audit VCS diff identifies Markdown-source divergence separately

  Scenario: Audit warns when a collaborator edited without track changes
    Given a newly received DOCX differs from the previous revealed accepted content
    And the new DOCX has no corresponding tracked-change records
    When `redline audit` compares the new reveal against current Markdown sources
    Then the audit report warns that track changes may have been disabled
    And the warning includes affected sections

  Scenario: Audit survives minor heading movement
    Given a section heading was renamed or moved slightly
    And most body content still matches a previous section
    When `redline audit` matches sections
    Then Redline matches the moved section by content similarity where safe
    And the audit report records the section match confidence

## Phase 5: better VCS diff quality

Feature: Normalize noisy Word XML into readable VCS diffs

  Scenario: Adjacent inserted runs produce readable added text
    Given Word XML contains adjacent insertion runs
    When Redline writes old and new section snapshots
    Then the new section contains the inserted text in order
    And the VCS diff shows a readable added line or hunk

  Scenario: Adjacent deleted runs produce readable removed text
    Given Word XML contains adjacent deletion runs
    When Redline writes old and new section snapshots
    Then the old section contains the deleted text in order
    And the VCS diff shows a readable removed line or hunk

  Scenario: Formatting-only noise is collapsed when possible
    Given Word XML splits unchanged text into multiple runs only because of formatting
    When Redline normalizes text runs
    Then the Markdown output does not expose formatting-only run boundaries
    And no VCS content change is emitted for formatting-only differences unless configured

  Scenario: Replacement remains readable in the VCS diff
    Given a deletion immediately precedes an insertion in the same sentence
    When Redline writes old and new section snapshots
    Then the old snapshot contains the deleted text
    And the new snapshot contains the inserted text
    And the old-to-new difference remains visible through `jj diff` or `git diff`

## Phase 6: comment accuracy

Feature: Resolve Word comment anchors accurately

  Scenario: A comment range anchored within one paragraph resolves exactly
    Given `word/document.xml` contains matching `commentRangeStart` and `commentRangeEnd`
    And both markers are in the same paragraph
    When Redline resolves comment anchors
    Then the comment anchor text equals the selected paragraph range
    And the comment section id is set

  Scenario: A comment range spanning multiple runs resolves exactly
    Given a comment starts in one text run
    And the same comment ends in a later text run
    When Redline resolves comment anchors
    Then the anchor text includes all selected run text in order

  Scenario: A comment range spanning multiple paragraphs resolves safely
    Given a comment starts in one paragraph
    And the same comment ends in a later paragraph
    When Redline resolves comment anchors
    Then the anchor range includes all selected paragraphs in order
    And the comment report identifies the starting section

  Scenario: A comment range anchored to inserted text records added kind
    Given a comment range is inside a Word `w:ins` element
    When Redline resolves comment anchors
    Then the comment anchor text equals the inserted selected text
    And the comment anchor kind is `added`

  Scenario: A comment range anchored to deleted text records deleted kind
    Given a comment range is inside a Word `w:del` element
    When Redline resolves comment anchors
    Then the comment anchor text equals the deleted selected text
    And the comment anchor kind is `deleted`

  Scenario: A comment range crossing normal and changed text records mixed kind
    Given a comment range starts in normal text
    And the same comment range continues through inserted or deleted text
    When Redline resolves comment anchors
    Then the comment anchor text includes all selected text in order
    And the comment anchor kind is `mixed`

  Scenario: Broken comment ranges fall back to nearby text
    Given `word/comments.xml` contains a comment
    And `word/document.xml` lacks a complete matching range
    When Redline resolves comment anchors
    Then the comment is still emitted
    And the comment has nearby paragraph context when available
    And the workspace includes a warning about the broken anchor

  Scenario: Threaded replies preserve parent-child order
    Given a parent comment has multiple replies
    When Redline renders the comment report
    Then replies reference the parent comment id
    And replies appear in deterministic timestamp or XML order

## Phase 7: imprint DOCX patching

Feature: Patch a copy of a received DOCX from Markdown sources

  Scenario: Imprint never mutates the source DOCX
    Given a source DOCX file `received.docx`
    And a Markdown workspace with edited sections
    When `redline imprint received.docx --output patched.docx` runs
    Then `received.docx` is byte-for-byte unchanged
    And `patched.docx` is created
    And `patched.docx` opens as a valid DOCX archive

  Scenario: Imprint replaces section content from Markdown
    Given a source DOCX contains a section `Introduction`
    And `sections/introduction.md` contains edited Markdown content
    When `redline imprint` patches a copy of the source DOCX
    Then the patched DOCX contains the edited introduction content
    And unrelated sections are preserved where safe

  Scenario: Imprint preserves useful Word container structure where safe
    Given a source DOCX contains styles, numbering, relationships, and section containers
    When `redline imprint` replaces Markdown-backed section content
    Then reusable Word container parts are preserved
    And unsupported structures are reported explicitly

  Scenario: Imprint reattaches figures and captions when supported
    Given the Markdown workspace contains a figure reference and caption
    And the figure asset exists under `figures/`
    When `redline imprint` patches a copy of the source DOCX
    Then the patched DOCX contains the figure asset
    And the caption appears near the figure

  Scenario: Imprint reattaches references when supported
    Given the Markdown workspace contains reference metadata
    When `redline imprint` patches a copy of the source DOCX
    Then references are included in the patched DOCX when supported
    And unsupported reference formatting loss is reported

  Scenario: Imprint reports unsupported structures without silent data loss
    Given a source DOCX contains an unsupported structure
    When `redline imprint` patches a copy of the source DOCX
    Then the command emits a warning naming the unsupported structure
    And the warning is available in machine-readable output

## Phase 8: disappear clean DOCX generation

Feature: Build a clean Word document from Markdown sources

  Scenario: Disappear merges sections in manifest order
    Given a workspace contains `manifest.json`
    And each manifest entry has a corresponding file under `sections/`
    When `redline disappear --output final.docx` runs
    Then Redline merges sections in manifest order
    And the merged Markdown is used as the Pandoc input

  Scenario: Disappear includes figures and captions
    Given a workspace contains Markdown figure references
    And referenced figure files exist under `figures/`
    When `redline disappear` builds a DOCX
    Then the generated DOCX contains the referenced figures
    And captions appear with their figures

  Scenario: Disappear includes bibliography data
    Given a workspace contains a bibliography file
    And Markdown content contains citations
    When `redline disappear` builds a DOCX with citation processing enabled
    Then the generated DOCX contains rendered citations
    And the generated DOCX contains a references section

  Scenario: Disappear uses a Pandoc reference document when configured
    Given a workspace config points to a Pandoc reference DOCX
    When `redline disappear` builds a DOCX
    Then Pandoc is invoked with the reference document
    And the generated DOCX uses the configured styles where Pandoc supports them

  Scenario: Disappear emits a clean DOCX without review markup by default
    Given Markdown sources come from accepted section files
    When `redline disappear` builds a clean DOCX with default options
    Then the generated DOCX does not contain review markup
    And accepted text appears in the generated document

  Scenario: Generated DOCX checks are structural rather than byte-for-byte
    Given `redline disappear` generated `final.docx`
    When tests validate the generated DOCX
    Then the file is a valid DOCX archive
    And required Word parts are present
    And byte-for-byte equality is not required

## Phase 9: workflow helpers

Feature: Improve everyday Redline workflows

  Scenario: Batch reveal processes a folder of DOCX files
    Given a folder contains multiple `.docx` files
    When a batch reveal command processes the folder
    Then each DOCX gets its own reveal workspace or output subdirectory
    And failures for one DOCX do not hide results for other DOCX files
    And the batch command returns a summary of successes and failures

  Scenario: Markdown front matter preserves document metadata
    Given a DOCX contains document metadata such as title, authors, or date
    When `redline reveal` writes Markdown files
    Then supported metadata appears as front matter
    And unsupported metadata is omitted or warned about deterministically

  Scenario: Reveal writes a review summary with counts
    Given a DOCX contains insertions, deletions, and comments
    When `redline reveal` processes the DOCX
    Then the workspace contains a summary of insertion count
    And the summary contains deletion count
    And the summary contains comment count

  Scenario: Partially parsed files produce warnings and meaningful exit status
    Given a DOCX contains both supported and unsupported structures
    When `redline reveal` processes the DOCX
    Then supported content is still written
    And warnings list unsupported structures
    And the command exit status follows the documented warning policy

  Scenario: HTML preview renders a reveal workspace
    Given a reveal workspace contains Markdown sections and comment metadata
    When Redline generates an HTML preview
    Then the preview contains section content
    And the preview shows tracked changes and comments clearly

## Supporting commands and platform behavior

Feature: Check local Redline dependencies

  Scenario: Check reports Go binary and Pandoc status
    Given Redline is installed
    When `redline check` runs
    Then the output includes the Redline version
    And the output includes whether Pandoc was found
    And the output includes the Pandoc path and version when available
    And missing optional dependencies are reported with actionable guidance

  Scenario: Check JSON output is CI friendly
    Given Redline is installed
    When `redline check --json` runs
    Then the output is valid JSON
    And dependency statuses are machine-readable
    And the command exit status follows the documented dependency policy

  Scenario: Commands that require Pandoc fail clearly when Pandoc is missing
    Given Pandoc is not available on `PATH`
    And `REDLINE_PANDOC` is not set
    When a Pandoc-dependent command runs
    Then the command fails before modifying outputs
    And the error explains how to install or configure Pandoc

  Scenario: REDLINE_PANDOC overrides Pandoc discovery
    Given `REDLINE_PANDOC` points to a Pandoc executable
    When a Pandoc-dependent command runs
    Then Redline uses the executable from `REDLINE_PANDOC`
    And the selected path is recorded in debug metadata when debug output is enabled

Feature: Pandoc pass-through and invocation safety

  Scenario: Pandoc pass-through forwards raw arguments
    Given Pandoc is available
    When `redline pandoc -- --version` runs
    Then Redline invokes Pandoc with `--version`
    And stdout and stderr are forwarded according to command policy

  Scenario: Pandoc defaults files are preferred over long command lines
    Given Redline needs to invoke Pandoc with many conversion options
    When Redline prepares the Pandoc invocation
    Then Redline writes a defaults YAML file
    And invokes Pandoc with the defaults file
    And records the defaults file when debug output is preserved

  Scenario: Pandoc writes to a temporary output before final move
    Given a Pandoc-dependent command has final output `final.docx`
    When Redline invokes Pandoc
    Then Pandoc writes to a temporary DOCX path first
    And Redline validates the temporary DOCX structurally
    And Redline moves it to `final.docx` only after validation succeeds

  Scenario: Debug mode preserves intermediate files
    Given `--debug` or `--keep-temp` is enabled
    When a command creates temporary inputs or Pandoc defaults
    Then Redline preserves the temporary directory
    And the command output reports where the debug files are located

Feature: Project context and configuration

  Scenario: Redline discovers project config by walking upward
    Given a file is inside a project tree
    And an ancestor directory contains `redline.yaml`
    When Redline runs for the file
    Then Redline uses the discovered `redline.yaml`
    And relative paths resolve from the project root

  Scenario: Redline creates a synthetic single-file project when no config exists
    Given no ancestor directory contains `redline.yaml`
    When Redline runs for a DOCX input
    Then Redline creates an in-memory single-file project context
    And the command can complete without a config file

  Scenario: Each CLI invocation uses an isolated session temp directory
    Given two Redline commands run concurrently
    When both commands create temporary files
    Then each command uses its own session temp directory
    And command-specific temp subdirectories do not collide

## Pandoc-assisted reveal experiments

Feature: Use Pandoc as a normalization aid without losing Redline source truth

  Scenario: Accepted-content Pandoc extraction captures visible document text
    Given Pandoc is available
    And a DOCX contains tracked changes
    When Redline runs Pandoc with `--track-changes=accept`
    Then the accepted Markdown contains the collaborator-visible current text
    And media is extracted to the configured figures directory

  Scenario: Rejected-content Pandoc extraction can build the old snapshot
    Given Pandoc is available
    And a DOCX contains tracked changes
    When Redline runs Pandoc with `--track-changes=reject`
    Then the rejected Markdown contains the old pre-review text
    And Redline can use that output as the VCS snapshot baseline

  Scenario: Pandoc AST extraction provides structured content hints
    Given Pandoc is available
    And Markdown output is too fragile for a fixture
    When Redline runs Pandoc with `--to json --track-changes=accept`
    Then Redline parses the Pandoc AST for document structure
    And does not treat the AST as an authoritative review-diff format

  Scenario: Pandoc output differences are recorded by version
    Given Pandoc is used during extraction or generation
    When Redline writes metadata
    Then the Pandoc version is recorded
    And tests can account for documented Pandoc version differences

## Parsing priorities and unsupported structures

Feature: Parse additional DOCX parts as they become relevant

  Scenario: Document relationships resolve media and hyperlinks
    Given `word/_rels/document.xml.rels` contains image and hyperlink relationships
    When Redline parses document relationships
    Then media relationship ids resolve to extracted assets
    And hyperlink relationship ids resolve to stable URLs

  Scenario: Header and footer content is parsed or explicitly warned about
    Given a DOCX contains tracked changes or comments in headers or footers
    When Redline processes the DOCX
    Then supported header/footer content is included in the VCS snapshots or comment report
    And unsupported header/footer review content emits an explicit warning

  Scenario: Footnotes and endnotes are parsed or explicitly warned about
    Given a DOCX contains footnotes or endnotes with tracked changes or comments
    When Redline processes the DOCX
    Then supported note content is included in the VCS snapshots or comment report
    And unsupported note review content emits an explicit warning

  Scenario: Numbering and list styles render deterministically
    Given a DOCX contains ordered and unordered lists
    When Redline renders Markdown
    Then list content is rendered deterministically
    And numbering/style loss is reported when it affects review meaning

## Stretch ideas

Feature: Optional review workflow extensions

  Scenario: Round-trip annotations back into DOCX
    Given a Markdown workspace contains review annotations
    When a round-trip annotation command writes a DOCX
    Then supported annotations appear as Word comments or tracked changes
    And unsupported annotations are reported explicitly

  Scenario: Export review comments for GitHub PRs
    Given a reveal workspace contains comments with anchor metadata
    When Redline exports GitHub PR comments
    Then each exported comment references the closest Markdown source location
    And comments without reliable locations are included in a summary

  Scenario: Review comments can be tagged by severity
    Given comment metadata contains reviewer comments
    When a user assigns severity tags
    Then tags are stored in machine-readable metadata
    And Markdown reports include the tags

  Scenario: LLM summary is generated from review activity
    Given a reveal workspace contains a VCS diff and comments
    When an optional LLM summary command runs
    Then the summary cites the VCS diff and comments used
    And the original comments remain unchanged

  Scenario: Managed Pandoc installation is reported or assisted
    Given Pandoc is missing
    When a managed installer helper is available
    Then Redline can report the managed install option
    And dependency checks remain explicit and machine-readable
