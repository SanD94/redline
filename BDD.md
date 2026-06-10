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
  so review artifacts can be inspected in git.

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
    And a later paragraph contains `w:commentRangeStart` for comment id `1`
    And `word/comments.xml` contains comment id `1`
    When the Word XML parser extracts the document
    Then comment id `1` has section id `discussion`

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
    And the comment text is `Nice comment.`
    When the Markdown renderer renders comments
    Then the output contains `## Comment 4`
    And the output contains `- **Author:** Reviewer`
    And the output contains `- **Date:** 2026-06-09T13:01:00Z`
    And the output contains this section line:
      """
      - **Section:** `discussion`
      """
    And the output contains `- **Text:** Nice comment.`

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
    And `comments.md` contains `## Comment 4`
    And `comments.md` contains `## Comment 5`
    And `comments.md` contains `## Comment 6`
    And `comments.md` contains `## Comment 7`
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

## Phase 2: reliable review model

Feature: Normalize DOCX review data into a stable internal model

  Scenario: A document block carries stable source identity
    Given a DOCX paragraph appears in `word/document.xml`
    And the paragraph belongs to section `introduction`
    When Redline builds the normalized review model
    Then the model contains a document block for the paragraph
    And the document block has a stable id
    And the document block has type `paragraph`
    And the document block has a source pointer into `word/document.xml`
    And the document block records surrounding text context

  Scenario: A tracked insertion is normalized as a change entity
    Given a DOCX paragraph contains a tracked insertion
    And the insertion has an author and timestamp
    When Redline builds the normalized review model
    Then the model contains a change with type `insertion`
    And the change has a stable id
    And the change preserves author and timestamp when present
    And the change has surrounding text context
    And the change has a raw DOCX XML pointer

  Scenario: A tracked deletion is normalized as a change entity
    Given a DOCX paragraph contains a tracked deletion
    And the deletion has an author and timestamp
    When Redline builds the normalized review model
    Then the model contains a change with type `deletion`
    And the old version includes the deleted text
    And the new version excludes the deleted text
    And the change has surrounding text context
    And the change has a raw DOCX XML pointer

  Scenario: A comment is normalized with its anchor range
    Given `word/comments.xml` contains a comment body
    And `word/document.xml` contains matching comment range markers
    When Redline builds the normalized review model
    Then the model contains a comment with a stable id
    And the comment preserves author and timestamp when present
    And the comment references an anchor range
    And the anchor range includes nearby selected text
    And the comment has a raw DOCX XML pointer

  Scenario: Model output is deterministic across repeated parses
    Given the same DOCX fixture is parsed twice
    When Redline builds normalized review models for both parses
    Then the model ids are stable across both parses
    And the section ordering is stable across both parses
    And the serialized review data is equal across both parses

## Phase 3: Markdown-first output

Feature: Write all reveal outputs as durable Markdown and JSON artifacts

  Scenario: Reveal writes a complete Markdown-first workspace
    Given a DOCX fixture contains headings, body text, tracked changes, and comments
    When `redline reveal` processes the fixture
    Then the workspace contains `sections/`
    And the workspace contains `document.md`
    And the workspace contains `changes.md`
    And the workspace contains `comments.md`
    And the workspace contains `review.json`
    And the workspace contains `source-map.json`
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

  Scenario: Insertions render with Redline insertion markup
    Given a normalized change has type `insertion`
    And the inserted text is `added text`
    When Redline renders review Markdown
    Then the output contains `[++ added text ++]`

  Scenario: Deletions render with Redline deletion markup
    Given a normalized change has type `deletion`
    And the deleted text is `removed text`
    When Redline renders review Markdown
    Then the output contains `[-- removed text --]`

  Scenario: Comment anchors render with inline markers and report entries
    Given a normalized comment is anchored to text `selected text`
    And the comment id is `c1`
    When Redline renders review Markdown
    Then the inline content contains a marker for comment `c1`
    And `comments.md` contains the body for comment `c1`
    And the marker can be traced back to `review.json`

  Scenario: Figures are extracted as first-class assets
    Given a DOCX fixture contains an embedded figure with a caption
    When `redline reveal` processes the fixture
    Then the figure file is written under `figures/`
    And Markdown content references the figure with a stable relative path
    And the caption remains associated with the figure
    And `review.json` records the figure metadata

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
    And the report includes a stable anchor when possible
    And the report is available in Markdown and JSON forms

  Scenario: Audit distinguishes tracked Word changes from inferred untracked changes
    Given a reveal workspace contains tracked Word changes from `review.json`
    And the current Markdown contains an additional edit not represented by tracked Word changes
    When `redline audit` runs in the workspace
    Then the audit report labels Word-tracked changes as `tracked`
    And the audit report labels Markdown-only differences as `inferred`

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

## Phase 5: better diff semantics

Feature: Normalize noisy Word tracked changes into readable review changes

  Scenario: Adjacent insertions by the same author are merged
    Given Word XML contains adjacent insertion runs
    And the runs share the same author and timestamp
    When Redline normalizes changes
    Then one insertion change is produced
    And the inserted text preserves the original order

  Scenario: Adjacent deletions by the same author are merged
    Given Word XML contains adjacent deletion runs
    And the runs share the same author and timestamp
    When Redline normalizes changes
    Then one deletion change is produced
    And the deleted text preserves the original order

  Scenario: Formatting-only noise is collapsed when possible
    Given Word XML splits unchanged text into multiple runs only because of formatting
    When Redline normalizes text runs
    Then the Markdown output does not expose formatting-only run boundaries
    And no content change is emitted for formatting-only differences unless configured

  Scenario: Replacement is detected from nearby delete and insert changes
    Given a deletion immediately precedes an insertion in the same sentence
    When Redline normalizes changes
    Then the change can be represented as a replacement when confidence is high
    And the original deletion and insertion remain traceable in structured data

  Scenario: Move operations are distinguished from insert/delete pairs when Word data allows
    Given Word XML contains `moveFrom` and `moveTo` elements with matching move identifiers
    When Redline normalizes changes
    Then the change is classified as `move`
    And source and destination anchors are preserved

  Scenario: Sentence-level context is reconstructed around each change
    Given a tracked change occurs inside a sentence
    When Redline emits review data
    Then the change includes sentence-level surrounding context
    And the context is deterministic across repeated runs

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
    Given Markdown sources contain Redline insertion and deletion markers
    When `redline disappear` builds a clean DOCX with default options
    Then the generated DOCX does not contain review markup
    And accepted text appears according to the configured review policy

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
    And the counts match `review.json`

  Scenario: Partially parsed files produce warnings and meaningful exit status
    Given a DOCX contains both supported and unsupported structures
    When `redline reveal` processes the DOCX
    Then supported content is still written
    And warnings list unsupported structures
    And the command exit status follows the documented warning policy

  Scenario: HTML preview renders a reveal workspace
    Given a reveal workspace contains Markdown sections and review metadata
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

  Scenario: Review-preserving Pandoc extraction captures review spans
    Given Pandoc is available
    And a DOCX contains insertions, deletions, and comments
    When Redline runs Pandoc with `--track-changes=all`
    Then the review Markdown contains review-preserving spans or markers
    And Redline can map those spans into its own review markup

  Scenario: Pandoc AST extraction provides structured review hints
    Given Pandoc is available
    And Markdown spans are too fragile for a fixture
    When Redline runs Pandoc with `--to json --track-changes=all`
    Then Redline parses Pandoc AST `Span` or `Div` classes for review data
    And reconciles AST hints with custom DOCX XML extraction

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
    Then supported header/footer content is included in review data
    And unsupported header/footer review data emits an explicit warning

  Scenario: Footnotes and endnotes are parsed or explicitly warned about
    Given a DOCX contains footnotes or endnotes with review data
    When Redline processes the DOCX
    Then supported note content is included in review data
    And unsupported note review data emits an explicit warning

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
    Given a reveal workspace contains comments and source-map data
    When Redline exports GitHub PR comments
    Then each exported comment references the closest Markdown source location
    And comments without reliable locations are included in a summary

  Scenario: Review changes can be tagged by severity
    Given review metadata contains changes and comments
    When a user assigns severity tags
    Then tags are stored in machine-readable metadata
    And Markdown reports include the tags

  Scenario: LLM summary is generated from review activity
    Given a reveal workspace contains changes and comments
    When an optional LLM summary command runs
    Then the summary cites the source changes and comments used
    And the original review data remains unchanged

  Scenario: Managed Pandoc installation is reported or assisted
    Given Pandoc is missing
    When a managed installer helper is available
    Then Redline can report the managed install option
    And dependency checks remain explicit and machine-readable
