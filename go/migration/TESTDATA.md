# Test data

What the port is checked against, where it comes from, and which of it is worth
the disk.

The Java tree is the specification and its tests are the assertions — that rule
does not change here. This file is about the *input*: the documents those tests
read, the ones they read but this repository does not carry, and the corpora
nobody has written an assertion for yet.

Four tiers, ordered by how targeted they are. A tier-1 file exists because one
numbered bug needed it. A tier-3 file exists because somebody crawled the web.
Both are useful; they answer different questions, and only the first two belong
in a test.

| Tier | What | Files | Where | Committed |
| --- | --- | ---: | --- | --- |
| 0 | In the repository already | 166 PDFs, and 13 more elsewhere in the tree | `*/src/test/resources/` | yes |
| 1 | What the Java build downloads | 74 of 78 | `*/target/{pdfs,fonts,imgs}` | no — `.gitignore` |
| 2 | Targeted third-party suites | 18,889 PDFs | `go/testdata/corpus/` | no — `.gitignore` |
| 3 | Bulk, for scoring not asserting | millions | not fetched | no |

Nothing below tier 0 is ever committed. These are other projects' documents
under other projects' licences; they are fetched, read as test input, and not
redistributed from here.

## Tier 0 — already here

166 PDFs came with the Java snapshot under `src/test/resources`, and 13 more sit
elsewhere in the tree; they are the only ones a fresh clone has. Most are small,
and each one was added alongside a fix.

| Where | Files | What it is for |
| --- | ---: | --- |
| `pdfbox/src/test/resources/input/` | 40 | The text-extraction corpus, with 81 expected-output `.txt` files beside it — `<name>.pdf.txt` unsorted and `<name>.pdf-sorted.txt` sorted by position. This is the only tier-0 set that ships its own ground truth |
| `pdfbox/src/test/resources/input/rendering/` | 6 | PDFs paired with reference PNGs, one per page. The raster ground truth |
| `pdfbox/src/test/resources/org/apache/pdfbox/multipdf/` | 28 | Merge, split and overlay inputs |
| `pdfbox/src/test/resources/input/merge/` | 15 | More of the same, for `PDFMergerUtility` |
| `.../pdmodel/interactive/form/` | 12 | AcroForm fields, appearances, flattening |
| `.../encryption/` | 10 | Every standard security handler revision, plus the keystores |
| `tools/src/test/resources/input/ImageIOUtil/` | 8 | Raster output |
| `pdfbox-layout-{awt,fop}/src/test/resources/pdf/` | 14 | The reference PDFs the two shaping backends render. `go/pdfbox/glyphlayout` is measured against these |
| `.../pdfparser/`, `.../input/compression/`, the rest | 33 | Missing catalogs, object streams, embedded files |

The Go tests reach these by relative path — `../../../pdfbox/src/test/resources/…`
— rather than copying them, so there is one copy and the Java and the Go read
the same bytes. Keep doing that. A fixture only belongs in a Go `testdata/`
directory when the Go generates it, as `multipdf/testdata/genjavabug82.go` does.

## Tier 1 — what the Java build downloads

PDFBOX-3974 added a `download-maven-plugin` block to `pdfbox/pom.xml`,
`fontbox/pom.xml`, `examples/pom.xml` and `benchmark/pom.xml`: **78 files, every
one pinned by a SHA-512 the pom carries, every one the reduced reproducer
attached to a numbered PDFBOX issue.** They are downloaded rather than committed
because they are large, third-party, or both. Nothing here runs Maven, so
`target/` stayed empty until `fetch-testdata.ps1` filled it on 2026-09-11; which
Java tests that unblocked, class by class, is in [`STATUS.md`](STATUS.md), "What
the fetch unblocked".

```bash
pwsh go/migration/scripts/fetch-testdata.ps1          # all of it, ~106 MB
pwsh go/migration/scripts/fetch-testdata.ps1 -List    # what it would fetch
pwsh go/migration/scripts/fetch-testdata.ps1 -Module fontbox -Kind fonts
```

The script reads the manifest out of the poms rather than carrying a copy, so it
cannot drift from the frozen Java tree, and it skips a file already present whose
SHA-512 matches. No JDK needed.

| Lands in | Files | What it unlocks |
| --- | ---: | --- |
| `pdfbox/target/pdfs` | 58, 55 of them PDFs | `TestPDFParser`'s xref-recovery suite, `PDFMergerUtilityTest`'s 30 cases, `PDButtonTest`, `TestRadioButtons`, `TestSymmetricKeyEncryption`, `TestQuality`, `ContentStreamWriterTest` |
| `pdfbox/target/fonts` | 12 | The IPA fonts, `PDFBOX-5484.ttf`, `n019003l.pfb` |
| `pdfbox/target/imgs` | 3 | `JPEGFactoryTest`, `LosslessFactoryTest` — a 16-bit PNG and two JPEGs |
| `fontbox/target/fonts` | 13 | `CFFParserTest`, `TestCMap`, `PfbParserTest`, `TTFSubsetterTest` — all four used to skip on every machine |
| `examples/target/pdfs` | 3 | `TestCreateSignature` |

Four entries no longer fetch, all four in `benchmark`, none read by a test: two
are hosted at `crossasia-books.ub.uni-heidelberg.de`, which no longer resolves,
one is the ECI Altona suite, whose host now serves something other than the
SHA-512 the pom pinned, and one is Adobe's own copy of ISO 32000-1. The script
reports them and exits non-zero; that is the expected result of a clean run.

## Tier 2 — targeted third-party suites

Tier 1 is targeted at the bugs PDFBox has had. That is not the same as targeted
at the format: it says nothing about the parts of ISO 32000 PDFBox has never been
handed a bad file for. These suites are organised the other way round — one file
per clause, per feature, per construct — so a gap shows up as a named file rather
than as a document that happens not to open.

```bash
pwsh go/migration/scripts/fetch-corpus.ps1            # every suite marked small, ~130 MB
pwsh go/migration/scripts/fetch-corpus.ps1 -List
pwsh go/migration/scripts/fetch-corpus.ps1 -Suite pdfjs
pwsh go/migration/scripts/fetch-corpus.ps1 -Suite itext-java,itext-dotnet   # ~1.1 GB, needs git
pwsh go/migration/scripts/fetch-corpus.ps1 -Suite itext-pdfhtml-java,itextpdf,rups   # and iText's other repositories, one name each
```

| Suite | PDFs | Licence | What it is, and what it reaches |
| --- | ---: | --- | --- |
| [`verapdf`](https://github.com/veraPDF/veraPDF-corpus) | 2,908 | CC BY 4.0 | **The best one-file-one-rule corpus that exists.** Each file asserts a single requirement of ISO 19005 (PDF/A 1–4), ISO 14289 (PDF/UA 1–2), ISO 32000-1 or ISO 32000-2; the directory and filename name the clause, and each file documents itself in its own outline. Reaches `cos`, `pdfparser`, `xmpbox`, `documentinterchange`, `graphics/color` |
| [`qpdf`](https://github.com/qpdf/qpdf) | 639 | Apache-2.0 | qpdf's own regression inputs: cross-reference tables and streams in every arrangement, object streams of every shape, linearised and not, damaged xrefs, every encryption revision qpdf can write. Reaches `pdfparser`, `pdfwriter`, `cos`, `pdmodel/encryption` |
| [`cabinet-of-horrors`](https://github.com/openpreserve/format-corpus/tree/master/pdfCabinetOfHorrors) | 24 | see repo | The files digital preservation keeps tripping over: broken embedded fonts, encrypted-without-password, malformed page trees |
| [`safedocs-targeted`](https://github.com/pdf-association/safedocs) | 11 | Apache-2.0 | Hand-coded by the DARPA SafeDocs programme to break parsers on purpose: dual `startxref`, Type 3 inside Type 3, a page with no `/Contents`, a font inside a shading pattern, UTF-16LE strings. Its `ContentStreamCycleType3insideType3.pdf` is [`JAVA-BUGS.md`](JAVA-BUGS.md) 87, the unbounded Type 3 recursion neither side guards; it is why `cmd/corpus` isolates each file |
| [`pdf20examples`](https://github.com/pdf-association/pdf20examples) | 7 | see archive | PDF 2.0 features one per file: UTF-8 strings, page-level output intents, black point compensation, 2.0 reached by incremental save, a non-zero start offset |
| [`text-rendering-tests`](https://github.com/unicode-org/text-rendering-tests) | 0 | OFL fonts | No PDFs — test fonts with expected glyph ids and positions per string. This is the only **shaping ground truth independent of PDFBox**, and `go/pdfbox/glyphlayout` is the one part of the port written from a specification rather than ported, so it is the one part with no Java to check against |
| [`pdfjs`](https://github.com/mozilla/pdf.js/tree/master/test/pdfs) | 1,443, `-Suite pdfjs` | Apache-2.0 | Reduced reproducers from another reader's tracker. Not ground truth — pdf.js's expectations are pdf.js's — but files that broke a real implementation, which is what makes them worth opening. 982 are committed in `test/pdfs`, 459 are `.link` stubs the script downloads and matches against the md5 in `test/test_manifest.json`, and two are committed elsewhere in the repository; `_links.tsv` says where each came from, and the manifest and those two are copied under `_repo/`. Twelve are encrypted and open with the passwords its own tests use, which the script writes to `_passwords.tsv`. `test/pdfs/sig_corpus` is not fetched: pdf.js generates its eight signed PDFs rather than committing or downloading them, and [`tasks/track-testdata-sources.md`](tasks/track-testdata-sources.md) U7 has what having them would take. Crash input, not assertions |
| [`itext-java`](https://github.com/itext/itext-java) | 6,897, `-Suite itext-java` | AGPL-3.0 or commercial | Every PDF committed to iText Core for Java, taken from `develop` as it was on 2026-09-14, each at its path in the repository and each checked against the blob id git records for it. All are test resources: `layout` 2,342, `kernel` 1,779, `svg` 1,446, `forms` 682, `sign` 386, `pdfa` 172, and six smaller modules. They are the inputs iText's tests read and the `cmp_` files the tests compare their output with. 234 name an encryption dictionary. [`scripts/passwords/itext-java.tsv`](scripts/passwords/itext-java.tsv) lists the ways the tests open them, and the 25 certificate and key files it names are fetched with the PDFs. **Not on disk:** the PDFs the tests write, which exist only after iText's Maven build has run them. See "iText against the Java" |
| [`itext-dotnet`](https://github.com/itext/itext-dotnet) | 6,960, `-Suite itext-dotnet` | AGPL-3.0 or commercial | The same for iText Core for .NET, under `itext.tests`. The library is ported from the Java and so are its tests: 6,678 of its 6,780 distinct contents are also in `itext-java`, and the other 102 are not, 80 of them in `itext.sign.tests`. 240 name an encryption dictionary; [`scripts/passwords/itext-dotnet.tsv`](scripts/passwords/itext-dotnet.tsv) |
| iText's other 32 repositories | 20,913, one `-Suite` name each | AGPL-3.0 or commercial | Everything else in [github.com/itext](https://github.com/itext) that holds a PDF: pdfHTML 15,279, the published examples and the books 2,446, iText 5 for Java and for .NET 1,703, its archived sandbox 515, pdfSweep 506, pdfOCR 365, and sixteen smaller repositories 99. Each is a suite of its own, fetched the way iText Core is. They are here because "PDFs iText wrote are all alike" is an argument and not a measurement; see "iText's other repositories" below for what measuring them said |

## Tier 3 — bulk

Not fetched, and not for assertions. These answer "does it crash", "does it hang",
and "has the open rate moved", over inputs nobody curated.

| Corpus | Size | Why it is on this list |
| --- | --- | --- |
| [SafeDocs issue-tracker corpus](https://labs.pdfa.org/stressful-corpus/), `pdfs_202011/batch1.tgz` | 1.8 GB | **Batch 1 is PDFBOX.** Every PDF attached to a public PDFBOX issue, crawled by NASA JPL — the superset of tier 1, without the curation or the checksums. The most on-point bulk corpus for this repository by a distance. Batches 2–6 are Ghostscript, Tika, Mozilla, LibreOffice, pdf.js, poppler, qpdf and 28 others. Note the README's own warning: the collection may contain malicious files |
| [GovDocs1](https://digitalcorpora.org/corpora/files) | ~231k PDFs | Public-domain real-world government documents. Unbiased in a way none of the above are: nothing in it was chosen because it broke something |
| [CC-MAIN-2021-31-PDF-UNTRUNCATED](https://digitalcorpora.org/cc-main-2021-31-pdf-untruncated/) | ~8M PDFs | The web as it is. Only worth it for a producer-distribution question |
| [pdf-association/pdf-corpora](https://github.com/pdf-association/pdf-corpora) | — | The index the three above came from, and the place to look before going hunting. Around fifty corpora with sizes and licences |

## Measured against the Java

Scoring the port on its own answers "where does it fall over", not "does it agree
with PDFBox", and those are different questions. This is the second one.

`migration/scripts/run-oracle.ps1` compiles `io`, `fontbox` and `pdfbox` out of
the tree with `javac` — no Maven, and it fetches the four compile-scope jars the
poms name — and runs PDFBox over the same file list, writing the same table.
`corpus -oracle` joins the two.

**Both sides take the passwords table**, and the run below is the one "Where it
stands" reports. Leaving `-Passwords` and `-passwords` out is a different run: 12
of pdf.js's files are encrypted and their passwords are in that table, so without
it 39 files are skipped as encrypted rather than 27, and those 12 are compared
only on being refused.

```bash
pwsh go/migration/scripts/fetch-corpus.ps1
pwsh go/migration/scripts/run-oracle.ps1 -Passwords go/testdata/corpus/pdfjs/_passwords.tsv
cd go && go run ./cmd/corpus -passwords testdata/corpus/pdfjs/_passwords.tsv \
    -oracle testdata/oracle/java-corpus.tsv \
    ./testdata/corpus ../pdfbox/target/pdfs ../examples/target/pdfs
```

One suite on its own is the same two commands with a list and a table of its
own; `-List` takes a file of paths, one per line, which `corpus` writes with
`-o` or a shell produces:

```bash
pwsh go/migration/scripts/fetch-corpus.ps1 -Suite pdfjs
pwsh go/migration/scripts/run-oracle.ps1 -List pdfjs-paths.txt -Out go/testdata/oracle/java-pdfjs.tsv `
    -Passwords go/testdata/corpus/pdfjs/_passwords.tsv
cd go && go run ./cmd/corpus -passwords testdata/corpus/pdfjs/_passwords.tsv \
    -oracle testdata/oracle/java-pdfjs.tsv ./testdata/corpus/pdfjs
```

What this compares is whether a document opens, how many pages it has, whether
its text extracts, and **how many characters** that text has. It is length, not
content: two documents of the same length are not the same document. What length
catches is a stage that silently produced nothing, a page that was not walked, a
glyph that came out as two characters. A wrong character in the right place
passes, and the ported Java tests are what covers that — this is the layer
underneath them, and its job is to say that nothing is wrong at a scale those
tests cannot reach.

### The separator, which has to be dealt with before any of this means anything

The first run of this comparison said 3,470 of 3,590 documents differed in text
length by more than 2%, which read as a broken extractor. It is not. Java's
`PDFTextStripper` initialises both `lineSeparator` and `pageEnd` from
`System.lineSeparator()`, which on Windows is CRLF; the port hardcodes LF and
says so at `text/pdftextstripper.go:22`. Every line of every document therefore
costs one character more on the Java side. The delta distribution says it
outright — 2,925 files differed by exactly **-1**, and the eight that matched
were the eight with no text at all.

So the oracle forces both to LF, and `run-oracle.ps1 -Crlf` reproduces the
confusion on demand. The deviation is real and deliberate; it is a convention,
not a defect, and comparing content requires taking it out first.

### What the runs found

Three runs, each scored again after its fixes. Every disagreement any of them
found was the port's, and none was a difference of opinion with PDFBox.

**The port on its own over tier 1 and tier 2, 2026-09-12.** 3,646 files in under
ten minutes, 3,596 of them opened. One port defect: `text.handleDirection`
panicked on a word made only of paragraph separators, where
`golang.org/x/text/unicode/bidi` resolves it to zero runs and `java.text.Bidi`
has no such edge. Fixed and pinned by `TestHandleDirectionTakesAWordWithNoRuns`
in `go/pdfbox/text/corpusdefects_test.go`; [`STATUS.md`](STATUS.md) carries it.
Three veraPDF files in clause 6.1.12, *implementation limits*, timed out, which
is close to what those files test for.

**The same 3,646 against PDFBox, 2026-09-12.** Twelve disagreements, and six
defects behind them, all closed:

| Defect | Files | What pins it |
| --- | ---: | --- |
| `processPages` walked `pages.Get(i)` where Java's is `for (PDPage page : pages)`. `PDPageTree` guards a cyclic page tree in two places on purpose — the indexed accessor throws, the iterator logs and skips the kid — and text extraction reaches the iterator's | 5 | `TestProcessPagesWalksTheTreeTheWayJavaDoes`, `go/pdfbox/text/corpusdefects_test.go` |
| A computed cross-reference repair was dropped. `XrefTrailerResolver.getXrefTable` hands out the live map and `validateXrefOffsets` edits it in place; the port's `XrefTable` answers a copy, so the swap went out of scope | 1 | `TestSwappedXrefEntriesAreRepaired`, `go/pdfbox/pdfparser/corpusdefects_test.go` |
| `Close` on a damaged Flate stream raised the damage `Read` had already absorbed. `compress/flate` remembers the error and hands it back from `Close`; Java's close is `inflater.end()` and cannot | 2 | `TestFlateDecoderReaderCloseSwallowsDamage`, `go/pdfbox/filter/flate_test.go` |
| The duplicate suppression scanned every key where Java asks a `TreeMap` for `subMap` and then `subSet` — the same answer, quadratic in the glyphs on a page, 22 seconds on two clause 6.1.12 files | 2 | commented at `go/pdfbox/text/pdftextstripper.go` |
| `COSParser.getObjectKey` copied the whole cross-reference table to ask how big it is, once per object read. On the ten-thousand-page Isartor file that was 71% of 238 seconds; answering without copying made it 0.51 | 1 | `XRefTableSize` and `EachXRefKey`, `go/pdfbox/cos/document.go` |
| Type 0 glyphs advanced by the font program instead of `/W`, which cost `PDFBOX-3951` one character in 126,330 | 1 | below |

Running the Java also settled the question qpdf's damaged files raise, which is
whether PDFBox's cross-reference reconstruction recovers them where the port does
not. It does not: seventeen of the eighteen fail in PDFBox with the identical
message, and the eighteenth was `issue-202.pdf`, the repair above.

**pdf.js against the Java, 2026-09-15.** Its 1,443 files gave fourteen
disagreements, all of them the length of the text; nothing PDFBox opened failed
to open, and no page count disagreed. Twelve were one defect, and fixing it
closed `PDFBOX-3951` above with them: `pdFont.Displacement` called its own
`Width`, where Java's `getWidth` is a virtual call reaching
`PDType0Font.getWidth` and the descendant's `/W`, so every glyph of every Type 0
font was advanced by its font program, embedded or substituted, in extraction
and in rendering alike.
`TestType0DisplacementIsTheDescendantWidth` pins it, and
[`tasks/track-testdata-sources.md`](tasks/track-testdata-sources.md) has it and
the four further differences that reading around it turned up. The other two are
[`JAVA-BUGS.md`](JAVA-BUGS.md) 15 and 30, where `track/java-bug-fixes` made the
Go differ from the Java on purpose.

### Where it stands

Everything on disk outside the repository — the 3,646 of the first run, pdf.js's
1,443, `PDFBOX-4131-0.pdf`, which the 2026-09-12 PDFBox table was missing,
iText Core's 13,857 and iText's other 20,913 — with `run-oracle.ps1`'s default
list and all eleven passwords tables on both sides. Measured 2026-09-16:

```
39860 files, opened 40136 ways, 27 of them encrypted and skipped
  open    both 40008, neither 125, behind 0, ahead 3
  pages   0 disagree
  text    both 39999, neither 9, behind 0, ahead 0
  chars   39997 the same length, 2 not
  digest  39997 of the same length the same text, 0 not

  5 of 39860 files disagree (0.01%)
```

"Opened 40,136 ways" is 39,860 files, 244 of which a passwords table opens more
than one way — with a password, with a certificate or with a keystore — for 276
openings beyond the first; "iText against the Java" and "iText's other
repositories", below, say which.

**There is no document in the corpus that PDFBox reads and the port does not.**
Not one, at either stage. No page count disagrees anywhere, and of the five files
in that last line, three are files PDFBox did not finish inside its 20-second
limit and the port did: `manyAppendModeUpdates.pdf`,
`background-size-near-zero-svg.pdf` and pdf.js's `issue1721.pdf`. Given five
minutes PDFBox reads all three, and its text is the port's to the digest — 4,216
characters, 1 and 2,036,568. The other two are the deliberate Java-bug fixes,
JAVA-BUGS 15 and 30: the only two texts in 39,997 that differ.

The run before it, of the same day and over the 18,947 files that were on disk
before iText's other repositories, was the first to compare the text by a digest
of its characters as well as by its length:

```
18947 files, opened 19201 ways, 27 of them encrypted and skipped
  open    both 19094, neither 107, behind 0, ahead 0
  pages   0 disagree
  text    both 19085, neither 9, behind 0, ahead 0
  chars   19083 the same length, 2 not
  digest  19083 of the same length the same text, 0 not

  2 of 18947 files disagree (0.01%)
```

The 27 encrypted files are 24 of qpdf's, one of cabinet-of-horrors' and two
PDFBox downloads, whose passwords are in qpdf's test scripts and PDFBox's Java
tests and not yet in a table. Until they are, both sides are compared only on
refusing them; how far finding them has got is
[`tasks/track-testdata-sources.md`](tasks/track-testdata-sources.md) U8.

## iText against the Java, 2026-09-16

Every PDF iText Core keeps in its two repositories, fetched and put through both
sides:

```bash
pwsh go/migration/scripts/fetch-corpus.ps1 -Suite itext-java,itext-dotnet
pwsh go/migration/scripts/run-oracle.ps1 -List <the 13,857 paths> -Out go/testdata/oracle/java-itext.tsv `
    -Passwords go/testdata/corpus/itext-java/_passwords.tsv,go/testdata/corpus/itext-dotnet/_passwords.tsv
cd go && go run ./cmd/corpus -passwords testdata/corpus/itext-java/_passwords.tsv \
    -passwords testdata/corpus/itext-dotnet/_passwords.tsv \
    -oracle testdata/oracle/java-itext.tsv ./testdata/corpus/itext-java ./testdata/corpus/itext-dotnet
```

**What is there.** 6,897 PDFs from `itext/itext-java` and 6,960 from
`itext/itext-dotnet`, from `develop` at the commits each suite's `_revision.txt`
records, both of 2026-09-14. That is every PDF either repository commits, and
every one is a test resource: the inputs iText's tests read, and the `cmp_` files
they compare what they write with. Each is at its path in the repository, and
each was checked against the blob id git holds for it, so a file changed on the
way — a line ending converted, a transfer cut short — fails the fetch. They are
542 MB of a 1.4 GB tree in each repository, so the script takes them with a
partial clone and a sparse checkout rather than an archive.

The .NET library is ported from the Java one, and so are its tests. Counted by
content, 6,678 distinct files are in both repositories, 39 only in the Java one
and 102 only in the .NET one, 80 of those in `itext.sign.tests`. Both are scored whole, so
most documents are compared twice, and the findings below come in pairs.

**Not there, and why.** The PDFs iText's tests write, which exist only after its
Maven build has run them; their `cmp_` counterparts are what is here. iText's
other 45 public repositories — about 21,000 PDFs of pdfHTML, pdfSweep, pdfOCR,
iText 5 and the published examples — which
[`TESTDATA-CANDIDATES.md`](TESTDATA-CANDIDATES.md) counts and nothing here
fetches. And nothing is behind a link: iText's tests read no PDF from the
network, and the one PDF address in their sources is a signature policy's, which
a test writes into a signature and never downloads.

**Passwords and keys.** 474 of the files name an encryption dictionary. Opened
with no password, PDFBox refuses 364 for the password, 80 as encrypted for a
certificate, 12 as revision 7 and 4 for a security handler it does not have, and
opens 14.

- **364 need a password.** The candidates are every string literal iText's test
  classes use as a password, and for `UnicodeBasedPasswordEncryptionTest` both
  strings of each entry in its table, the one as typed and the SASLprep result the
  test encrypts with: 47 in all, counting the empty one. PDFBox was asked to open
  each encrypted file with each, and every one of the 364 opened with at least
  one. [`scripts/passwords/itext-java.tsv`](scripts/passwords/itext-java.tsv) and
  [`itext-dotnet.tsv`](scripts/passwords/itext-dotnet.tsv) give each file every
  candidate PDFBox opened it with — the user and the owner password both where
  there are both, and where a prepared password differs from the typed one, as
  for an Arabic ligature, Katakana and Hangul that normalise, a soft hyphen or
  non-ASCII spaces, the owner password both ways. When the tables were written,
  each password was checked against the string literals of the test class named
  above its group.
- **80 are encrypted for a certificate.** Each is given the certificate and
  private key its test decrypts with, and the key's passphrase, `testpassphrase`
  where the key is encrypted and empty where it is not. Each certificate was
  checked against the issuer and serial number the file's `/Recipients` names.
  The tables name 25 certificate and key files in each repository, and those are
  fetched with the PDFs.
- **12 are revision 7**, AES-GCM from ISO/TS 32003, which PDFBox refuses before it
  looks at a password. They are given the passwords their tests use, so that both
  sides are asked the same thing.
- **14 open with no password.** 8 of them also open with an owner password their
  tests use, and are opened both ways; the other 6 are given nothing.
- **4 name `/Standart` or `/iText` as their security handler**, for iText's tests
  of custom handlers. Neither side has one, and they are given nothing.

That is 464 files opened 718 ways. The tables are committed, since they hold file
names, passwords and key file names and none of iText's content, and
`fetch-corpus.ps1` writes each into its suite as `_passwords.tsv`.

**Wrong passwords as well.** A table of passwords that open says nothing about a
password one side accepts and the other refuses. So both sides were also run
once with every one of the 47 candidates on each of the 378 files a password
opens or that open with none: 17,766 opens, from a table kept out of the
repository. Both sides accepted the same 622 and refused the same 17,144, and the
622 texts have the same digest on both.

### Three disagreements, two of them defects

The first run, with no passwords:

```
13857 files
  open    both 13375, neither 482, behind 0, ahead 0
  pages   0 disagree
  text    both 13371, neither 4, behind 0, ahead 0
  chars   13371 the same length, 0 not
```

Nothing disagreed, which is as much as a comparison of length can say, and is
why both tables now carry a digest of the text. With it, the same run showed two
documents with the same number of characters and not the same text. The
passwords then brought four files PDFBox opens and the Go version refused, and
four that crash on both sides:

| Files, each in both repositories | Go | PDFBox | Cause |
| --- | --- | --- | --- |
| `kernel/parser/BidiTextExtractionTest/in02.pdf` | 179 characters | 179 characters, not the same | runs of mixed-direction text written in logical order |
| `kernel/crypto/PdfEncryptionManuallyPortedTest/encryptedWithCertificateAes128.pdf`, `kernel/crypto/pdfencryption/PdfEncryptionTest/encryptedWithCertificateAes128.pdf` | `key encryption algorithm 1.2.840.113549.1.1.7 is not supported` | opens, "Hello world!" | RSAES-OAEP recipients not read |
| `kernel/crypto/securityhandler/PubSecHandlerUsingAesGcmTest/externalFile.pdf`, `invalidCryptFilter.pdf` | panic: slice bounds out of range [:32] with capacity 20 | `ArrayIndexOutOfBoundsException` | JAVA-BUGS 89, the same crash |

#### Mixed-direction text written in logical order

`in02.pdf` is two lines of Hebrew, a list of inventions with their inventors'
names and years in parentheses. PDFBox extracts each line in reading order. The
Go version extracted the same 179 characters with each line's runs in the order
they were stored: the line started from its middle, and each year stood next to
the wrong names.

`PDFTextStripper.handleDirection` asks `java.text.Bidi` for a word's runs and
their levels, puts the runs into visual order with `Bidi.reorderVisually`, and
reverses the odd-level ones. The port asked `golang.org/x/text/unicode/bidi`,
which answers runs in logical order and no levels, and wrote them as they came.
A word of one direction has one run and comes out the same either way; a word
with runs of both, like a Hebrew line with digits in it, does not, and its runs
came out in the wrong order. Reordering changes no character count, so the
earlier comparisons, of length alone, could not see it. `track/pdfbox-layout` had found the same
limitation of `x/text` and written `go/javatext/bidi`, the port of
`java.text.Bidi` with levels and `reorderVisually`, for
`AbstractGlyphLayoutProcessor`; `handleDirection` never moved to it.
`go/pdfbox/text/direction.go` now uses it, line for line with the Java, keeping
JAVA-BUGS 15's fix. `TestHandleDirectionPutsRunsInVisualOrder` holds fourteen
words and PDFBox's answer for each, printed by calling its `handleDirection`,
the two lines of `in02.pdf` among them; eight of them failed before the change.
With it, `in02.pdf`'s text is PDFBox's byte for byte.

#### RSAES-OAEP recipients not read

A certificate-encrypted document carries its key wrapped for each recipient, in a
CMS envelope. PDFBox unwraps it with BouncyCastle's
`JceKeyTransEnvelopedRecipient`, which takes the algorithm the recipient names:
RSA with PKCS#1 v1.5 padding, which is what PDFBox writes, or RSAES-OAEP. iText
writes OAEP, with an empty parameter sequence, every field at its default. The
port's `cms.go`, which stands in for BouncyCastle, unwrapped PKCS#1 v1.5 alone
and refused the other. It now reads the OAEP parameters — hash, mask generation
hash, label, and their defaults — into Go's own `rsa.OAEPOptions`.
`TestOAEPRecipientUnwraps` builds envelopes with iText's parameters, with none,
with SHA-256 for both hashes, with SHA-256 for the hash alone and with a label,
and one with a mask generation function other than MGF1, which is refused.

Two things are true of these files' neighbours and are not disagreements. The
other 70 certificate-encrypted files that open at all — keys wrapped with PKCS#1
v1.5, in versions PDFBox implements — opened on both sides from the first. And `kernel/crypto/PdfDecryptingTest/adobe/withCertificate/aes256EcdsaP256.pdf`,
which Acrobat encrypted for an elliptic curve certificate with a key agreement
recipient, is refused by both: PDFBox hands the recipient BouncyCastle's key
transport unwrapper and fails with a `ClassCastException`, and the port skips key
agreement recipients and answers that none matches. The comment in `cms.go`
says so.

#### Version 6 certificate encryption, JAVA-BUGS 89

ISO/TS 32003 adds `/V 6` and the `AESV4` crypt filter method. PDFBox does not
implement them, and for a certificate-encrypted document it does not refuse them
either: it takes a SHA-1 digest for any version but 4 or 5, then copies a key of
the crypt filter's 256 bits out of its 20 bytes. The port does the same and
panics where PDFBox throws. That is a bug in the Java, carried; [`JAVA-BUGS.md`](JAVA-BUGS.md)
89 has it, and `TestVersion6CertificateEncryptionReadsPastTheDigest` pins it on
PDFBox's own `AESkeylength256.pdf` rewritten to version 6.

It showed as a disagreement only because `cmd/corpus` wrote a panic during
opening to the text column and left the open column `ok`. It now writes a panic
to the stage that was running, and opening runs to the page count, where
`JavaCorpus`'s own fence ends.

#### After the fixes

Over iText, with both tables:

```
13857 files, opened 14111 ways
  open    both 14057, neither 54, behind 0, ahead 0
  pages   0 disagree
  text    both 14053, neither 4, behind 0, ahead 0
  chars   14053 the same length, 0 not
  digest  14053 of the same length the same text, 0 not

  0 of 13857 files disagree
```

What neither side opens: the 12 revision 7 files, 22 ways; 20 files broken on
purpose or empty, 8 refused with `Missing root object specification in trailer`,
8 with `Page tree root must be a dictionary` and 4 that end before their first
line; the 4 custom handlers; the 4 version 6 certificate files; the 2 elliptic curve recipients;
and the 2 copies of `kernel/pdf/PdfReaderTest/exponentialXObjectLoop.pdf`, which
outlasts both timeouts. The four whose text neither side extracts are the two
copies each of `kernel/pdf/PdfFontTest/cmp_halfWidthFont.pdf` and
`cmp_uniJIS2004UTF16Font.pdf`, whose fonts name the predefined CMaps
`UniJIS-UTF32-H` and `UniJIS2004-UTF16-H`, which neither side carries.

#### And everything together, with the digest

Every file on disk, iText Core's 13,857 and the 5,090 that were there before, is
"Where it stands" above: of the 18,947 they then made, 2 disagree, and they are
JAVA-BUGS 15 and 30 as before. Before the last fix below, the digest also found
a third:

**A position's text reversed by script, not by direction.** pdf.js's
`TaroUTR50SortedList112.pdf` is a table of code points for vertical text, 64,255
characters on both sides, and every one of its 57,547 text positions is the same
on both — Unicode, codes, position, width, height and font, compared as bits.
One line differed: in the row for U+05BF HEBREW POINT RAFE, a space stood on the
other side of the point. `TextPosition.getVisuallyOrderedUnicode` reverses a
position's text when a code point in it is right to left by
`Character.getDirectionality`. The port asked instead whether the code point
belongs to the Hebrew, Arabic, Syriac, Thaana, NKo, Samaritan or Mandaic script.
That counts those scripts' combining marks and digits, which are not right to
left — the position there holds RAFE merged with the space beside it, and the
port reversed the two — and it misses the right-to-left characters of every other
script, and the right-to-left mark.
Measured against JDK 17 over every code point, the script test called 361 code
points right to left that are not and missed 1,473 that are. `isRightToLeft` in
`go/pdfbox/text/textposition.go` now asks the bidi class `x/text` holds, R or AL,
for an assigned code point. That answers JDK 17's 2,904 code points and 58 more,
all added in Unicode 14 and unknown to the JDK's Unicode 13, and it is Unicode
15's data, as the rest of the package's is. `TestVisuallyOrderedUnicodeAsksTheDirectionality`
holds PDFBox's answers for twenty-one strings, that position's among them; seven
failed before the change. The file's text is now PDFBox's byte for byte.

## iText's other repositories, 2026-09-16

`itext-java` and `itext-dotnet` are iText Core. The other 45 repositories of
[github.com/itext](https://github.com/itext) are its add-ons, its samples and
its tools, and 32 of them hold PDFs: **20,913 files, 889 MB, all of them on disk
and all of them scored.**

What was nearly recorded instead is why they are here. The first answer was that
they need not be fetched: 15,279 of them are pdfHTML's `cmp_` files, most of the
rest is expected output, so they are PDFs iText wrote — the same writer as the
13,857 that had just agreed on every file. That is an argument about the
producer. It is not a measurement of the files, and a document is only ever
settled by opening it.

```bash
# each repository is a suite of its own, fetched the way iText Core is
pwsh go/migration/scripts/fetch-corpus.ps1 -Suite itext-pdfhtml-java,itextpdf,rups
pwsh go/migration/scripts/run-oracle.ps1 -List <every path> -Out java-full.tsv `
    -Passwords (Get-ChildItem go/testdata/corpus/*/_passwords.tsv).FullName
cd go && go run ./cmd/corpus -passwords testdata/corpus/itextpdf/_passwords.tsv ... ./testdata/corpus
```

**What is there.** Counted from each default branch's tree on 2026-09-16 and
fetched the same day, each file at its path in its repository and each checked
against the blob id git holds for it:

| Group | Repositories | PDFs |
| --- | ---: | ---: |
| pdfHTML, for Java and for .NET | 2 | 15,279 |
| the published examples and the three books | 7 | 2,446 |
| iText 5, for Java and for .NET | 2 | 1,703 |
| the archived iText 5 sandbox | 1 | 515 |
| pdfSweep | 2 | 506 |
| pdfOCR | 2 | 365 |
| demos, tutorials, RUPS and the rest | 16 | 99 |
| repositories with no PDF in them | 13 | 0 |

Most of it is iText's own output, and the one part that is not is iText 5:
`itextpdf`, `itextsharp` and `i5js-sandbox` are the previous generation of the
library, a different writer from iText Core.

**Encrypted files.** 35 of the 20,913 name an encryption dictionary. Eight
repositories publish the passwords and the keys for them, in their samples and
tests, and eight tables under [`scripts/passwords/`](scripts/passwords/) carry
them: 28 of the 35 files, 50 ways of being opened. Twelve of those 28 open with
no password as well as with their owner's, and keep a line for each. Of the
seven the tables leave alone, five open with no password and nothing else is
known about them, and two are below. Two of the tables name a PKCS#12 keystore
the sample decrypts with, `test.p12` with the passphrase its own comment gives,
which the fetch brings down with the PDFs; that is the third shape a passwords
line now has.

Two files are opened by neither side, and the reason is in the material rather
than in either reader.
`itext-publications-book-java`'s `Listing_12_11_EncryptWithCertificate` writes
for two recipients. The key of the one whose certificate the repository carries
is in a JKS keystore whose store password and key password differ — `f00b4r` and
`f1lmf3st`, both in the sample — and PDFBox takes **one** password for both
(`COSParser.prepareDecryption`), so it answers "the private key is not
recoverable" with the first and "keystore was tampered with" with the second.
Measured, both ways. The other recipient's key is in a store the repository does
not carry.

Run once more over every candidate password these projects hold — 13 of them
against each of the 14 files a password touches, 182 openings — both sides
accepted exactly the same 20 and refused exactly the same 162, and the 20 they
accepted gave the same text on both sides.

**What it found.** Both sides over all 20,913, with the tables on each:

```
20913 files, opened 20935 ways
  open    both 20917, neither 17, behind 0, ahead 1
  pages   0 disagree
  text    both 20917, neither 0, behind 0, ahead 0
  chars   20917 the same length, 0 not
  digest  20917 of the same length the same text, 0 not
```

**Nothing disagrees.** Every file PDFBox opens, the port opens; every page count
is the same; and all 20,917 texts are the same length *and* the same characters
— 57,496 pages and 44,088,979 characters of them. The "ahead" is
`itext-pdfhtml-dotnet`'s `background-size-near-zero-svg.pdf`, which PDFBox did
not finish inside the 20-second limit and the port read in under one; given
five minutes PDFBox reads it too, and agrees.

The 17 openings neither side gets past are the 4 revision 7 files, the 2
certificate files above, 7 files of zero length, and 4 whose trailer names no
root object — `itextpdf` and `itextsharp`'s `endArrayClosingBracketInsteadOfEndDic.pdf`
and `endDicClosingBracketInsideTheDic.pdf`, which iText 5's own
`CompressionTest` expects to fail. Both sides refuse each of them with the same
reason.

## Scoring a corpus

Three thousand files cannot become assertions by hand, and they should not: the
question they answer is not "is this byte right" but "which of these can the port
open, read and draw at all, and did that change".

`go/cmd/corpus` is that. It is migration tooling, not a port of anything — PDFBox
has no such command.

```bash
cd go
go run ./cmd/corpus -o corpus.tsv ./testdata/corpus            # one row per file
go run ./cmd/corpus -render ./testdata/corpus/safedocs-targeted
go run ./cmd/corpus -baseline corpus.tsv ./testdata/corpus     # only what changed
```

One row per file, one column per stage — open, pages, text, chars, render — each
cell `ok` or the first thing that went wrong. Saved and passed back through
`-baseline`, the table is a regression check: a file that used to open and no
longer does is a defect the unit tests did not catch, and it exits non-zero so CI
can hold the line. A file that used to fail and now opens is a slice landing.

`-isolate` is on by default and scores each file in a child process. That is not
belt-and-braces: `recover` cannot catch a Go stack overflow, and a goroutine that
will not return cannot be stopped from outside, so without it one file ends the
run. `-isolate=false` is faster on a corpus already known to be survivable.

`-passwords` takes a table of the ways a project's tests open its encrypted
files, in UTF-8, and can be given more than once. Each line is one way of opening
one file: `<path ending>`, tab, `<password>`; or `<path ending>`, `<password>`,
`<certificate>`, `<private key>`, tab separated, for a file encrypted for the
holder of a certificate, with the two files named relative to the table and the
password decrypting the key where it is encrypted; or `<path ending>`,
`<passphrase>`, `<keystore>`, for a project that keeps that certificate and key
in a PKCS#12 keystore of its own, which is the shape both loaders read and is
handed over as it is. `fetch-corpus.ps1` writes one as `_passwords.tsv` for a
suite whose project publishes them — pdf.js, both iText Core suites, and eight
of iText's other repositories — and `run-oracle.ps1 -Passwords` gives PDFBox the
same tables. A
file is opened once for every line naming it and gets a row for each; where there
is more than one, each row's file is followed by the rest of its line in brackets.
A file no line names that refuses a password is reported `encrypted` and left out
of the rates; a file opened as a line says keeps its error if it still will not
open.

The last column of both tables is a digest of the text: the first eight bytes of
the SHA-256 of its UTF-8. Where two texts have the same length, `-oracle` compares
the digests as well and reports a pair that differs as `DIGEST`, which is the
wrong character, or the right characters in the wrong order, that length alone
passes. A PDFBox table written before the column existed is compared on length
alone.

`-oracle` walks PDFBox's table as well as the run's own rows. A row the table has
under one of the directories the run was given, and the run has no row for, is
reported as `MISSING` and counted as behind, so a file the port never answered
for cannot pass as a clean comparison. Rows outside those directories are left
alone: the table is usually the whole corpus's, and a run is often one suite.

**Narrowing one of those to a page.** The digest says a document's text differs;
it does not say where. Both drivers write a page table on request — one row per
page per way of opening a file, with that page's text digested the same way —
and `corpus -comparepages` joins them. Both tables are walked: a file one side
opened and the other did not is one `OPEN` line, a page only one table has is a
`PAGE` line, and a file only one table holds is a `MISSING` line, each counted as
a disagreement. Run it over the files the comparison named, not over a corpus:
each page is extracted in a pass of its own.

```bash
printf 'go/testdata/corpus/pdfjs/bug1175962.pdf\n' > narrow.txt
pwsh go/migration/scripts/run-oracle.ps1 -List narrow.txt -Out java-pages.tsv -Pages
cd go && go run ./cmd/corpus -pages go-pages.tsv testdata/corpus/pdfjs/bug1175962.pdf
go run ./cmd/corpus -comparepages ../java-pages.tsv go-pages.tsv
```

```
  CHARS  go/testdata/corpus/pdfjs/bug1175962.pdf page 1: go 117, java 126
  CHARS  go/testdata/corpus/pdfjs/poppler-90-0-fuzzed.pdf page 10: go 4, java 209
```

Those are the two files that disagree, each narrowed to its page in seconds:
JAVA-BUGS 15 on the first page of one, and on the tenth page of the other the
content stream that ends early under JAVA-BUGS 30's fix.

Note what this does **not** do: it does not compare against the Java. Running
PDFBox over three thousand files is not cheap either, and the comparison that
matters is per-case rather than per-corpus. For that, keep to what
[`README.md`](README.md) already says — run the Java on the one file, and convert
the answer into a pinned Go assertion rather than a commit message.

## Where a new document should go

1. **Does a tier-0 file already show it?** Use that one. There are more of them
   than the Go tests currently read.
2. **Does the Java download one for exactly this?** Then the pom has it, the
   SHA-512 is already written down, and `fetch-testdata.ps1` already fetches it.
3. **Is the smallest file that shows the behaviour one you can write?** Write it,
   as a Go generator under the package's `testdata/`, the way
   `multipdf/testdata/genjavabug82.go` does. A generated fixture can be read in a
   text editor and re-made when the question changes; a downloaded one cannot.
4. **Only then reach for a corpus.** And when a corpus file settles a question,
   the answer belongs in a Go test with the file's name in the comment — not in
   a directory nobody re-runs.
