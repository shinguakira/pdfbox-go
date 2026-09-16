# Performance plan

The plan as it was written, before any of it was started. What happened when it
was carried out, and where the plan was wrong, are the last two sections.

[`BENCHMARK.md`](BENCHMARK.md) measured the Go version against PDFBox and found
where the time goes. This file is what could be done about it without leaving
pure Go, in the order it looks worth doing, with what each step needs decided
before it starts.

Each item was checked against the Go source and against the Java it ports. Where
the two already do the same thing, the item says so: a change there is a
deviation, not a repair.

## Constraints

- **Pure Go.** No cgo, no WebAssembly, no external executable.
- **No new dependency without a decision.** One item below would add one; it is
  marked, and it waits.
- **No large hand-written replacement of a library without an instruction.** One
  item below would be that; it is listed so the option is on record, and it is
  not to be started on its own initiative.
- **Every change keeps the oracle where it is.** `corpus -oracle` stood at 1
  disagreement over the 3,646 documents of the day, `PDFBOX-3951`; a change that
  moves it is a change that broke something, however much faster it is.
  [`TESTDATA.md`](TESTDATA.md) is where the comparison stands now.
- **A deviation from the Java is commented where it happens and recorded in
  [`STATUS.md`](STATUS.md)**, even when it is only an allocation size.

## What the profiles say

Measured on the 40 heaviest documents of the benchmark set, one pass. Those 40
are 92% of the Go version's time on all 3,597, so they are where a change
shows.

### CPU, 34.0 s

| where | time | share |
| --- | ---: | ---: |
| inflating compressed streams — `compress/flate` | 11.0 s | 32% |
| allocating memory and collecting it | 8.0 s | 24% |
| operating-system calls, from the font mapper reading installed fonts | 5.0 s | 15% |
| reading one byte at a time — `bufio.(*Reader).ReadByte` | 1.9 s | 6% |
| the Go version's own code | 2.9 s | 9% |
| everything else | 5.2 s | 15% |

The classification groups `pprof` function lines by name, so the edges between
rows are approximate; the ranking is not.

Two of those rows are one cause seen twice. 89% of the byte-at-a-time reads are
made *inside* `compress/flate` — `huffSym` pulling its input through
`io.ByteReader` — so they belong to the inflate row.

### Allocation, about 24 GB in the one pass

| allocated by | amount | share | reached from |
| --- | ---: | ---: | --- |
| `bytes.growSlice` | 4,487 MB | 18.5% | `filter.Flate.Decode` inflating into a `bytes.Buffer` of its own, grown by doubling |
| `compress/flate.(*dictDecoder).init` | 3,342 MB | 13.8% | a new 32 KB window for every stream |
| `pdfio.(*ReadBuffer).expandBuffer` | 2,911 MB | 12.0% | half `ReadWriteBuffer.Write`, half `NewReadBufferFromReader` |
| `ttf.NewRandomAccessReadDataStream` | 1,537 MB | 6.3% | a full copy of the font file on every load |
| `cmap.(*CMap).addCharMapping` | 1,465 MB | 6.0% | character mapping tables |
| `encoding.newEncodingBase` | 989 MB | 4.1% | two maps pre-sized to 250 per encoding |
| `ttf.(*CmapSubtable).processSubtype4` | 866 MB | 3.6% | TrueType cmap subtables |
| `pdfio.NewReadBufferSize` | 774 MB | 3.2% | |
| `cff.(*CFFCharsetType1).AddSID` | 530 MB | 2.2% | |
| `ttf.readBytes` | 499 MB | 2.1% | |

The first two rows are **32% of everything allocated**, and both are the inflate
path — but neither is inflating. One is setting up a decompressor; the other is a
buffer Java does not have. The collector's work follows the allocator's, so they
are also part of the 24% allocation row of the CPU table.

## The work

**This is the plan as it was written, and it was carried out.** Each item below
keeps the reasoning it had then; what each one came to -- including the two that
were measuring the wrong thing -- is in "Where the plan was wrong", at the end.

### 1. Inflate

The largest single cost. **Most of it is the decoding itself, and the standard
library cannot make that faster.** What it can remove is the waste around the
decoding, which is the two largest allocation rows.

What `Flate.Decode` does today, in
[`pdfbox/filter/flate.go`](../pdfbox/filter/flate.go):

1. wraps its input in a `byteCountingReader`, whose count nothing reads;
2. calls `flate.NewReader`, which allocates a new decompressor and a new 32 KB
   window, for every stream;
3. inflates into a `bytes.Buffer` of its own, which grows by doubling;
4. copies that buffer into `w` through `decodePredictor` — a plain `io.Copy`
   when there is no predictor. `w` is a `pdfio.ReadWriteBuffer` in both callers
   in `pdfbox/cos/stream.go`.

Java's `FlateFilter.decode` has no step 3. It transfers the inflating stream
straight into the predictor-wrapped output. **The Go version writes every
inflated byte into memory twice, the first time into a buffer that grows by
doubling; Java writes it once.**

The Go comment gives the reason for the buffer: a decode error must not lose the
bytes that did inflate. Java meets that another way —
`FlateFilterDecoderStream` catches the damage and ends the stream — and the Go
version already has the same in `flateDecoderStream`.

| | change | size | new dependency |
| --- | --- | --- | --- |
| **1a** | Reuse decompressors: a `sync.Pool` of readers, reset per stream through `flate.Resetter`. `compress/flate`'s `Reset` keeps the 32 KB window and the input buffer | small | no |
| **1b** | Drop the buffer: inflate through `flateDecoderStream` into the predictor and on into `w`, the way `FlateFilter.decode` does. `byteCountingReader` goes with it | small–medium | no |
| **1c** | Replace `compress/flate` with `github.com/klauspost/compress/flate` | small | **yes — needs a decision** |
| **1d** | Write an inflater | large | no — **not without an instruction** |

**1b moves the Go version toward the Java**, so it needs no deviation entry, and
it brings one behaviour change with it, also toward the Java: `Decode` logs
*every* inflate error and carries on, a failing source included, where Java
reads the source outside its `try` and catches `DataFormatException` alone. That
needs a test of its own before the change.

**1c** is API-compatible with `compress/flate` (`NewReader`, `Resetter`), has no
cgo and no assembly in its `flate` package, and its module requires nothing
else — checked against v1.19.0. It is a decision about dependencies rather than
about code, and is taken on measurements after 1a and 1b; "Where the plan was
wrong" is where it was taken.

### 2. Allocation outside inflate

Everything in this section is the Go version doing what Java does, or close to
it. So every item here is a deviation, for allocation only, and is commented and
recorded if it is done.

| | change | size | what Java does |
| --- | --- | --- | --- |
| **2a** | Share the font bytes in `ttf.NewRandomAccessReadDataStream` instead of copying the whole file | small–medium | Copies too: `RandomAccessReadDataStream` reads the whole source into a new `byte[]`. Every caller has to be checked for writes to the buffer before it is shared |
| **2b** | Build character-mapping tables on first use rather than on load | medium | Builds them on load, and the Go `CMap.addCharMapping` is a case-for-case port of Java's. Measure first which of its maps the 6.0% is, and whether extracting text reads it |
| **2c** | Drop the pre-size of 250 in `encoding.newEncodingBase` | small | Pre-sizes to 250 as well: `Encoding`'s two `HashMap`s |
| **2d** | Allocate a `pdfio.ReadBuffer` in one piece when its length is known | small | Grows 4 KB at a time, as the Go version does. Those bytes are the data being stored, not waste, so this would cut the number of allocations and not the amount. Last, and possibly not worth doing |

2b is not the GPOS change again. That change skipped work PDFBox never does —
PDFBox has no GPOS reader; see [`BENCHMARK.md`](BENCHMARK.md), "The GPOS table,
in `fontbox/ttf`". The character-mapping tables are work PDFBox does.

`ttf.(*CmapSubtable).processSubtype4`, 3.6%, has not been compared with the
Java yet.

### 3. Installed fonts — cause not yet known

15% of CPU is operating-system calls, reached through
`fontMapperImpl.findFont` → `GetTrueTypeFont`: the Go version looking on disk
for a substitute when a document names a font it does not embed.

The Go version has the same process-wide `FontCache` PDFBox has, and consults it
in `fsFontInfo.Font` before reading a file. So it is one of two things, and they
want different fixes:

- **The cache is missing** — the same substitute is found and read again. Fix
  the cache key or equality.
- **The cache is hitting** — the cost is the first read of each *distinct*
  substitute. Then 2a and 2b are what reduce it.

**The first step is to count hits and misses over the 40 documents.** Nothing
here should be changed until that number exists.

### 4. The 366 MB floor — no step yet

**Answered by change 3**, which took the heap left after a collection down to a
third; see the end of this file.

[`BENCHMARK.md`](BENCHMARK.md) found that the Go version does not go below
about 366 MB, where PDFBox completes in 48 MB. Nothing above is known to move
that. **The first step is an in-use heap profile taken at the floor.**

One candidate, not measured: the Go `FontCache` holds each font outright where
Java holds it through a `SoftReference`, so a font the Go version reads stays
for the life of the process. The deviation is
[`STATUS.md`](STATUS.md)'s, under slice 4.

## Order

1. **1a, 1b.** Standard library only; output unchanged apart from 1b's
   source-failure case, whose test comes first. Measure.
2. **Decide on 1c**, with the numbers from step 1 in hand.
3. **Count the font cache hits (3), and take the heap profile at the floor
   (4).** Then fix what those point at.
4. **2a, 2b, 2c**, each measured on its own so it is known what each bought. 2d
   only if the numbers still say so.

## How each step is checked

After every change, all four:

1. `go run ./cmd/corpus -oracle testdata/oracle/java-corpus.tsv ...` — still
   **1 of 3,646**, and still the same file.
2. `go run ./cmd/bench -list <the 40 documents>` before and after — time and
   peak heap.
3. An allocation profile before and after — the row the change targeted has
   actually fallen.
4. `gofmt -l . && go vet ./... && go test ./...` — clean.

A step that is faster and fails the first check is reverted, not adjusted.

## What this will and will not change

It will change the heavy tail, because that is where every item above lives —
the 40 documents are 92% of the time, and inflate and allocation are 56% of
theirs.

It will not change the startup time, which was already several times faster than
PDFBox when this was written, or the size of the binary beyond what 1c would add
if it is taken.

It will probably not close the whole gap to PDFBox on those documents, because
the decoding itself is the largest cost and Java's `Inflater` is zlib in C.

## What happened when it was carried out

The plan above is left as it was written. It was prototyped on
`track/performance` on 2026-09-15, not committed. This section is what that
found; the next one is where the plan was wrong.

### What was changed, and what each change bought

| # | change | where | what made it slow | what Java does | allocated, 40 heaviest, one pass |
| --- | --- | --- | --- | --- | ---: |
| — | before | | | | 24,373 MB |
| 1 | Decompressors reused, and inflating goes straight through the predictor into the output. The plan's 1a and 1b | `pdfbox/filter/flate.go`, `pdfbox/filter/predictor.go` | a new decompressor and 32 KB window for every stream, and every inflated byte written into memory twice, the first time into a buffer that grew by doubling | inflates straight into the output; a new `Inflater` per stream, in native memory | 13,853 MB, with 2 |
| 2 | `CreateView` answers the decoded buffer instead of copying it, and sizes buffer chunks from `/Length`. Not in the plan | `pdfbox/cos/stream.go` | every stream opened as a view was copied a second time | `createView` answers the buffer `Filter.decode` wrote; chunks are four times `/Length` when that is under 1 KB | measured with 1 |
| 3 | `PDPageTree.All` hands each page the document's resource cache. Not in the plan | `pdfbox/pdmodel/pdpagetree.go` | no page had a cache, so every font lookup built the font again, inflating and parsing an embedded font file each time: `PDFBOX-4423-000746.pdf` built 139 fonts out of 8 font objects | `PageIterator.next()` passes `document.getResourceCache()`, as `get(int)` does | 2,318 MB |
| 4 | The content-stream decompressor goes back to the pool when its data ends | `pdfbox/filter/flate.go` | a new decompressor and window for every page | a new `Inflater` per stream | 1,912 MB |

The tests that pin changes 1, 3 and 4, and the pooled decompressor as a
deviation, are [`STATUS.md`](STATUS.md)'s, under "Deviations — `filter`" and
"`PDPageTree` and the resource cache".

Change 1 does not take 1b's route through `flateDecoderStream`. It keeps
`Decode`'s handling of a failing source exactly as it was — every error ends
the data — so that the output before and after could be required to be
identical. The difference from Java that 1b names, a source's `IOException`
propagating, was closed afterwards by `track/testdata-sources`.

### Before and after

The speed and the peak heap of this prototype are not quoted here: it ran while
other programs were using about four cores, and [`BENCHMARK.md`](BENCHMARK.md)
measured the same changes again on a quiet machine afterwards. What is kept is
what that page does not carry — the heap left after a collection, and the
one-pass profile, the after one with all four changes in:

| | before | after |
| --- | ---: | ---: |
| heap left after a collection, 40 heaviest | 336 MB | **74 MB** |
| heap left after a collection, all 3,597 | 418 MB | **133 MB** |
| CPU, 40 heaviest, one pass | 38.8 s | **6.9 s** |
| — building fonts | 23.4 s, 60% | 0.3 s, 4% |
| — inflating | 14.1 s, 36% | 0.2 s, 3% |
| — the collector marking | 7.5 s, 19% | 0.7 s, 10% |

### How the output was checked

Besides the four checks under "How each step is checked", the throwaway harness
`go/testdata/oracle/perfcheck`, which git and `./...` both ignore, compared the
build before the changes with the build after them: every stream's decoded
bytes through `CreateReader` and through `CreateView`, 42,435 streams in 3,626
documents; every document's extracted text, 3,646 documents; and the pixels of
the first three pages at 36 DPI, 497 pages of 400 documents. No difference
anywhere.

### Why there was this much to take

Mostly change 3, which speeds nothing up: it removes work the Go version was
doing and PDFBox never did. Text extraction walks the page tree, the walk handed
out pages without a resource cache, and so every font lookup built its font
from nothing — for an embedded font, inflating and parsing the font file again.
Building fonts was 60% of the CPU, and 97% of the time spent inflating was
inflating font files for it. A missing cache changes no output, so neither the
tests nor the oracle, which both compare output, could see it.

## Where the plan was wrong

- **Inflate was not the largest cost.** "What the profiles say" put inflating
  first at 32%, and "The work" started there. Inflating was the largest leaf,
  but 97% of it was reached from building fonts, and building fonts again for
  every lookup (change 3) was the cost. Nothing in the plan pointed at it.
- **Item 3, installed fonts, measured something that is not there.** The CPU
  table's 15% of operating-system calls from the font mapper does not appear in
  the profile it came from: `fontMapperImpl.findFont` is 0.23 s of 41.2 s,
  0.56%. The runtime's own operating-system work — returning memory,
  preempting threads — was about 5%, and it belongs to the collector. Counting
  font cache hits was not needed.
- **Item 4, the 366 MB floor, had a step and a cause after all.** The heap left
  after a collection fell to a third over all 3,597 documents, as "Before and
  after" shows, and change 3 is almost all of it.
- **1c, klauspost/compress, is not worth a dependency.** Over all 23,392 flate
  streams in the corpus, 16 of them damaged, it gave identical output, at 1.14×
  the speed of `compress/flate` (1.10× on the 40 heaviest). With inflating at
  3% of the CPU after the changes, that is not worth adding to `go.mod`.
- **2b and 2c were symptoms.** The font-file copy and the character-mapping
  tables were large because fonts were built again and again: after change 3,
  `ttf.NewRandomAccessReadDataStream` allocates 43 MB where it allocated
  1,559 MB, and `cmap.(*CMap).addCharMapping` and `encoding.newEncodingBase`
  are out of the top sixty. Neither is worth a deviation from Java now.
- **"What this will and will not change" aimed too low.** It expected the gap
  to PDFBox to stay mostly where it was, at 6.4× on the total, because inflate
  in pure Go is slower than zlib. What the gap is now is
  [`BENCHMARK.md`](BENCHMARK.md)'s headline.

What is left, on the after profile: parsing content-stream tokens 21% of CPU,
`showText` 20%, loading the document 17%. A second difference from Java found
on the way — `CreateReader` did not collapse a repeated filter the way
`Filter.decode` does — was a behaviour question rather than a speed one, and
`track/testdata-sources` closed it; [`STATUS.md`](STATUS.md) records it with its
tests.
