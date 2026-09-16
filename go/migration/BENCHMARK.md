# Speed and memory, against PDFBox

Measured 2026-09-15 by running both, on the same documents, on one machine: an
Intel Core i7-8700, 6 cores and 12 threads, 48 GB, Windows 10; Go 1.26.1;
OpenJDK 17.0.19 with its default G1 collector and, unless a row says otherwise,
`-Xmx4g -Xss8m`. Every run also recorded how much CPU the rest of the machine
used while it ran: between 0.3 and 1.3 cores of the twelve, with the exceptions
named where they occur.

This page was first written on 2026-09-13, before
[`PERFORMANCE-PLAN.md`](PERFORMANCE-PLAN.md) was carried out, and everything
measured on it was measured again on 2026-09-15, after. Where a section records
what was found on the first date, it says so and keeps that date's numbers.

**Not yet measured here: the page purge.** On 2026-09-16 `track/testdata-sources`
ported `PDPage.removePageResourceFromCache` and the call to it that ends
`PDFTextStripper.processPage`. On this page's 3,597 documents it makes a pass
allocate 4.4% more with the same characters out, and what it does to time could
not be told apart from the noise that day, so the tables below are the code
before it. The runs are in
[`tasks/track-testdata-sources.md`](tasks/track-testdata-sources.md), "Found in
review".

`go/cmd/bench` and `migration/oracle/JavaBench.java` print the same numbers in
the same shape; [`scripts/run-oracle.ps1`](scripts/run-oracle.ps1) builds the
Java side. The unit of work is one document's worth of real use — open the file,
count the pages, extract the text — because the question is what a caller pays
for a document, not how fast a function is.

**The set is 3,597 documents, 226.8 MB**: every file of the 2026-09-13 corpus
that *both* implementations fully process, the same list both times, so the two
dates compare. Timing a file one of them gives up on would score "fast" for
"stopped early". It does not include pdf.js's files, which arrived later. Both
sides extract 1,834,959 characters from it; on 2026-09-13 the port extracted one
more, the `PDFBOX-3951` defect [`TESTDATA.md`](TESTDATA.md) records.

## Headline

| | port | PDFBox | |
| --- | ---: | ---: | --- |
| total, 3,597 documents, best of three passes | 8,234 ms | 5,625 ms | 1.46× |
| documents per second | 436.8 | 639.5 | |
| per document, median | 0.333 ms | 0.394 ms | **port faster** |
| per document, p90 | 2.00 ms | 1.50 ms | 1.33× |
| per document, p99 | 12.0 ms | 12.0 ms | the same |
| slowest document | 847 ms | 534 ms | 1.6×, and it is the same document |
| peak live heap | **249.7 MB** | 590.4 MB | **2.4× the other way** |
| heap the OS was asked for | 263.8 MB | — | |

**2,410 of the 3,597 documents are faster in the port**, and the median is
faster. The ten slowest documents are 44.5% of the port's time and 39.5% of
PDFBox's, and none of them takes the port twice what it takes PDFBox: the most is
1.83×, `qpdf/inline-images-ii-some.pdf`, 140.0 ms against 76.5.

It is also only true of one worker on a machine with cores to spare. Everything
else measured points the other way:

| | port | PDFBox | |
| --- | ---: | ---: | --- |
| CPU time, a warmup and one pass | **17.1 s** | 48.8 s | **2.9× the other way** |
| cores used | 1.11 | 2.93 | |
| four workers | **2,669 ms** | 3,202 ms | **1.2× the other way** |
| cold start, one document | **59 ms** | 615 ms | **10.4× the other way** |
| peak resident set | **288 MB** | 724 MB | **2.5× the other way** |
| minimum shipped | **15.56 MB** | 49.0 MB | **3.1× the other way** |

### Since 2026-09-13

| | port, 09-13 | port, 09-15 | PDFBox, 09-13 | PDFBox, 09-15 |
| --- | ---: | ---: | ---: | ---: |
| total | 36,805 ms | **8,234 ms** | 5,779 ms | 5,625 ms |
| per document, p90 | 2.38 ms | **2.00 ms** | 1.53 ms | 1.50 ms |
| per document, p99 | 30.6 ms | **12.0 ms** | 12.0 ms | 12.0 ms |
| ten slowest, share of the total | 79% | **44.5%** | 40% | 39.5% |
| CPU time | 90.3 s | **17.1 s** | 56.5 s | 48.8 s |
| peak live heap | 794.9 MB | **249.7 MB** | 593.6 MB | 590.4 MB |
| peak resident set | 854 MB | **288 MB** | 740 MB | 724 MB |

The port got 4.5× faster, used 5.3× less CPU and held 3.2× less heap. PDFBox
moved within the noise, which is what it should do: its code did not change,
only the machine's state around it. What changed the port is in
[`PERFORMANCE-PLAN.md`](PERFORMANCE-PLAN.md), "What happened when it was carried
out".

## CPU, which is where the headline changes shape

Wall clock is not what a container is billed for. One warmup pass and one timed
pass, both sides, as process CPU time; the median of three rounds, run in turn:

| | port | PDFBox |
| --- | ---: | ---: |
| wall, the whole process | 15.4 s | 16.7 s |
| user CPU | 15.3 s | 44.7 s |
| kernel CPU | 1.9 s | 4.2 s |
| **total CPU** | **17.1 s** | **48.8 s** |
| **cores used (CPU ÷ wall)** | **1.11** | **2.93** |

**PDFBox's best pass is 1.46× faster, and it spends 2.9× the CPU to get there.**
Over the whole process — the warmup pass the JIT is still compiling through, and
the timed one — the port finishes first. The rest of PDFBox's advantage is
bought with cores, by the JIT compiler threads.

Which number matters depends entirely on the deployment. On an idle machine with
cores to spare and a process that lives long enough to warm up, the best pass is
the answer and PDFBox is faster. Where CPU is what is counted — a one-core
quota, a bill per CPU-second — the CPU column is the answer, and the port needs
2.9× less. Neither was measured under an actual quota; both are read off the
table above. For a process that handles a few documents and exits, see "Starting
up".

### Where those cores go, and whether the port is missing something

The obvious reading of 2.93 against 1.11 is that PDFBox parallelises something
the port does not. It does not.

**PDFBox processes documents on one thread.** Its whole source has a single
`Thread`, in `IOUtils`, and it is a shutdown hook that deletes temporary
directories. No `ExecutorService`, no `parallelStream`, no `ForkJoinPool`. The
port has no goroutines in its ported packages either. On this they are equal.

The cores are the JIT, and constraining the JVM says so. The median of three
rounds each:

| | wall | CPU | cores |
| --- | ---: | ---: | ---: |
| default | 16.7 s | 48.8 s | 2.93 |
| `-XX:+UseSerialGC` | 16.1 s | 45.1 s | 2.81 |
| serial GC, one compiler thread | 21.9 s | 39.1 s | 1.78 |

"One compiler thread" is `-XX:+UseSerialGC -XX:-TieredCompilation
-XX:CICompilerCount=1`. Serialising the collector changes little, so it is not
GC. Restricting the compiler to one thread drops PDFBox to 1.78 cores — and costs
31% of the wall clock. The extra cores are not overhead being wasted: PDFBox is
buying speed with them, by compiling on other cores while its one application
thread runs.

**And the port is not leaving a win on the table that PDFBox has taken.** Both
were given a worker pool over the same list — `-workers` here, `ExecutorService`
there:

| workers | port | PDFBox | port ÷ PDFBox |
| ---: | ---: | ---: | ---: |
| 1 | 8,308 ms | 6,100 ms | 1.36× |
| 4 | **2,669 ms** | 3,202 ms | **0.83×** |
| 12 | **2,368 ms** | 2,654 ms | **0.89×** |
| **speedup** | **3.51×** | **2.30×** | |

**With four workers or more, the port is the faster of the two.** The output was
identical at every worker count on both sides — 1,834,959 characters — which is
also the answer to whether the port has shared mutable state getting in the way:
it does not. On 2026-09-13 the port scaled 2.19× and PDFBox 2.24×, and both hit
the same ceiling, because ten documents were 79% of the port's time and no number
of workers gets below the slowest one. Now the tail is gone, and the cores say
the rest: PDFBox's one worker already uses 2.93, four use 6.57 and twelve 8.61,
where the port's go from 1.09 to 3.97 and 6.27.

Workers cost the port less memory than they cost PDFBox, too: the port's peak
live heap went 246 MB → 294 MB → 415 MB across those runs, PDFBox's 575 MB →
593 MB → 761 MB. On 2026-09-13 it was the other way round, 729 MB → 1,605 MB
against 611 MB → 760 MB.

## Starting up

Process launch to exit, as the operating system timed it, for one small document
— `safedocs-targeted/Dual-startxref.pdf`, 17 characters — extracted twice, a
warmup and a pass. Ten runs, best and mean:

| | port | PDFBox |
| --- | ---: | ---: |
| best | **59 ms** | 615 ms |
| mean | 62.7 ms | 648.6 ms |

**10.4× the other way.** There is no JVM to start, no classes to load and nothing
to JIT. For a command-line tool or a per-request process this reverses the
throughput result completely: PDFBox takes about 590 ms longer than the port to
start, extract this document twice and exit, which is roughly 1,800 median
documents' worth of the port's time.

The warmup is visible in the passes, too. PDFBox's three timed passes ran 6,309 →
5,733 → 5,625 ms as the JIT settled; the port's were 8,309 → 8,352 → 8,234 ms,
flat from the first.

## What has to be shipped

| | port | PDFBox |
| --- | ---: | ---: |
| the library | — | 1.75 MB of classes |
| its resources | — | 4.45 MB (glyph lists, AFMs, CMaps) |
| log4j-api | — | 0.34 MB |
| Bouncy Castle, if signing or public-key encryption | — | 11.74 MB |
| a runtime | included | 42.5 MB jlink minimum, 302 MB for this JDK |
| **binary, stripped** | **15.56 MB** | — |
| binary, unstripped | 20.50 MB | — |
| **total, minimum** | **15.56 MB** | **49.0 MB** |

The binary is `go/cmd/pdfbox`, built with and without `-ldflags "-s -w"`; the
Java column was measured on 2026-09-13 and nothing in it has changed since. One
file against a tree, and 3.1× smaller at the minimum — more against a full JDK.
The port's number is also the whole story: no runtime to install, no classpath,
no version to match.

## Memory, as the operating system sees it

Peak resident set for the CPU runs above, the median of three, which is the
number a container limit is compared against:

| | peak RSS | 2026-09-13 |
| --- | ---: | ---: |
| port | **288 MB** | 854 MB |
| PDFBox, `-Xmx4g` | 724 MB | 740 MB |
| PDFBox, `-Xmx256m` | 399 MB | 399 MB |

RSS is above the heap figures because it includes what neither heap accounts for
— the JVM's metaspace, code cache and thread stacks, the Go runtime's own arenas.
The port's resident set is now below PDFBox's even with PDFBox's heap capped at
256 MB.

## Four ways to measure this that do not work

Each was tried first, on 2026-09-13, and produced a confident wrong answer. The
numbers in this section are that date's.

**Timing a small document once.** `time.Now()` on this machine resolves about
7µs, and 2,516 of the 3,597 files are veraPDF clause tests a few hundred bytes
long. The first run reported a median of 0.000 ms, which is a statement about
the clock. Both sides now repeat a document until 2 ms has accumulated and
divide, which is what `testing.B` does — and using the same method on both
matters more than using each language's best clock.

**Go's `HeapSys`.** The first memory figure was 2,727 MB, which is the arena the
runtime has taken from the OS and does not give back. It is not comparable to a
JVM heap. Both sides now report the peak heap in use, sampled every millisecond:
`HeapAlloc` on one side, `MemoryMXBean.getHeapMemoryUsage()` on the other.

**Adding up the JVM's pool peaks.** The Java side's first memory figure was the
sum of `MemoryPoolMXBean.getPeakUsage()` over the heap pools. Each pool reaches
its peak at its own moment — the young generation just before a collection, the
old one somewhere else — so the sum describes a heap that never existed, and it
is never smaller than the real peak. Sampling the whole heap instead took the
default run from 638.4 MB to 593.6 MB.

**Comparing two default configurations.** Both runtimes use what they are given.
PDFBox peaked at 513 MB under `-Xmx4g` and at 94 MB under `-Xmx96m` on the 40
documents of "Memory" below, extracting identical text either way. A
default-vs-default number compares two GC settings, not two implementations. See
"Memory" for the number that means something.

And one presentational error: comparing the two `max` columns. Those were
*different documents* — on 2026-09-13 the port's slowest was not PDFBox's
slowest — and dividing them gave "20×" where the same document was **112×**.

## Where the time actually goes

The same document, both sides, slowest in the port first:

| document | port | PDFBox | |
| --- | ---: | ---: | ---: |
| `PDFBOX-3949-MKFYUG…pdf` | 847.1 ms | 533.9 ms | 1.6× |
| `PDFBOX-3947-670064.pdf` | 503.1 ms | 367.5 ms | 1.4× |
| Isartor PDFA-1b 6.1.12 `t01-fail-a` | 441.7 ms | 260.7 ms | 1.7× |
| `PDFBOX-3785-202097.pdf` | 319.8 ms | 252.3 ms | 1.3× |
| `PDFBOX-3951-FIHUZ…pdf` | 251.3 ms | 149.7 ms | 1.7× |

**The port's slowest documents are now PDFBox's slowest documents**, the top four
in the same order, and the port takes between 1.3× and 1.7× as long on each. The
three that led this table on 2026-09-13 fell out of it:

| document | port, 09-13 | port, 09-15 | PDFBox, 09-15 |
| --- | ---: | ---: | ---: |
| `PDFBOX-4423-000746.pdf` | 10,485 ms | 103.8 ms | 97.7 ms |
| `PDFBOX-4418-000671.pdf` | 6,760 ms | 209.9 ms | 184.6 ms |
| `PDFBOX-3964-c687766d…pdf` | 3,137 ms | 128.6 ms | 124.6 ms |

By ratio rather than by time, over the 3,597 documents, every one of which takes
more than 0.05 ms on both sides:

```
                      09-15   09-13
port faster (<1×)      2410    2300
1–2×                   1115    1114
2–5×                     68     153
5–20×                     3      20
over 20×                  1      10
```

The four documents still above 5× are small veraPDF files, where a few
milliseconds make a large ratio: `PDF_A-1b 6.1.3 File trailer t01-fail-a` 7.58 ms
against 0.34 ms, `t02-fail-a` 3.50 against 0.31, `TWG A004-pdfa1-pass-a` 4.00
against 0.41, and `PDF_A-2b 6.6.2.3.1 t15-fail-q` 2.00 against 0.34. Why is not
yet known.

## What was fixed

Before the performance plan, on 2026-09-13: two defects, both found by this
benchmark. The numbers in this section are that date's. What the plan fixed
after them is in [`PERFORMANCE-PLAN.md`](PERFORMANCE-PLAN.md).

### The predictor's row, in `filter`

`qpdf/issue-1688a.pdf` is **531 bytes** and took **435 ms**, against PDFBox's
0.3 ms — 1,449×. 83% of it was the runtime zeroing memory. The file declares:

```
/DecodeParms << /Predictor 2 /Colors 536870913 /Columns 1 /BitsPerComponent 8 >>
```

The port worked the row out as 536,870,913 bytes, and `decodePredictor`
allocated two of them — **a gigabyte of zeroed memory from a 531-byte file**.
PDFBox's row for this file is 32 bytes: `Predictor.wrapPredictor` reads
`/Colors` through `Math.min(..., 32)` before the row length is worked out. What
is there now is that clamp, where Java has it, and Java's 32-bit arithmetic all
through the predictor — `/BitsPerComponent` and `/Columns` are not clamped and
can still wrap. Matching PDFBox's output case by case turned up the rest: the
last row completed with zeros, a row's algorithm byte read as a signed Java
byte, and a predictor of 0 or below passing the data through whatever the other
parameters say. [`STATUS.md`](STATUS.md), "Slice 1 — `pdfbox/filter`", records
it with the tests that hold PDFBox's own output for each case; JAVA-BUGS 88 is
the one case where PDFBox has no answer.

PDFBox's own `SECURITY.md` puts disproportionate resource consumption from small
attacker-controlled inputs in scope, so this was not only a speed defect.

### The GPOS table, in `fontbox/ttf`

The port's table switch is identical to `TTFParser.readTable` **except for one
extra case**: `GPOS`. PDFBox has no glyph positioning reader at all — GSUB is
there, GPOS is not — and an unknown tag becomes a plain `TTFTable` that reads
nothing. So `parseTables` costs Java nothing for GPOS, and cost the port a full
parse on every font load: **work PDFBox does not do**, on a path that never uses
it. Extracting text from `PDFBOX-5927.pdf`, a one-megabyte document, the port
held 464 MB against PDFBox's 59.4 MB, and 52 MB of that was `readPairSet` —
kerning pairs, parsed while extracting text. The only caller of `GPOS()` in the
tree is the shaper, and `table()` already reads on demand.

Left until asked for, which `TestGPOSIsNotReadUntilAsked` pins. **464 MB →
156 MB on that document**, and across the whole corpus:

| | before | after |
| --- | ---: | ---: |
| peak live heap | 1,651.0 MB | **794.9 MB** |
| heap from the OS | 2,731.8 MB | **803.8 MB** |
| total allocated | 118,373 MB | 104,883 MB |
| total time | 38,691 ms | 36,806 ms |

Memory is where it paid; the throughput barely moved, because those documents
were under a second of thirty-eight.

## What was not fixed on 2026-09-13, and what became of it

The page said two things were not worth fixing, and neither diagnosis held.
**`compress/flate`** was named as most of the remaining gap on the heavy
documents — the 112× document, `PDFBOX-4423-000746.pdf`, spent 26% of its time
in `huffSym`, and Go's inflate is pure Go where PDFBox's is native zlib. **`encoding.newEncodingBase`,
58.75 MB on `PDFBOX-4418-000671.pdf`,** was put down as a faithful 250-entry
pre-size with 15,422 encodings live at the peak. Both were symptoms of fonts
being built again on every lookup: [`PERFORMANCE-PLAN.md`](PERFORMANCE-PLAN.md),
"Where the plan was wrong", has the measurements, and what those two documents
take now is in "Where the time actually goes" above.

## Memory

Defaults compare settings. The number that compares implementations is how far
each can be squeezed while still producing identical output. Measured on the
same 40 documents as on 2026-09-13, which were then the heaviest: they are now
61.5% of the port's per-document time and 55.5% of PDFBox's. PDFBox's heap is
capped with `-Xmx`; Go has no hard cap, so the port's is `GOMEMLIMIT`, a soft
limit.

| cap | PDFBox | | port | |
| --- | ---: | --- | ---: | --- |
| 4 GB / none | 3,349 ms | peak 500 MB | 5,150 ms | peak 143 MB |
| 512 MB | 3,515 ms | peak 377 MB | 5,624 ms | peak 133 MB |
| 256 MB | 3,714 ms | peak 229 MB | 5,319 ms | peak 138 MB |
| 128 MB | 3,904 ms | peak 120 MB | 4,958 ms | peak 109 MB |
| 96 MB | 4,278 ms | peak 94 MB | 5,004 ms | peak 79 MB |
| 64 MB | 5,134 ms | peak 63 MB | 6,828 ms | peak 79 MB |
| 48 MB | 8,425 ms | peak 48 MB | 16,695 ms | peak 76 MB |
| 32 MB | **fails** — output truncated | | — | |

All 40 documents come out at 1,610,612 characters on both sides at every level
but PDFBox's 32 MB, where it exits with 567,779. The port's 96, 64 and 48 MB rows
are from a second run: during the first, the rest of the machine took 2.5 cores
and the port's times were 3–19% longer.

**PDFBox still completes in 48 MB.** Its 500 MB under a 4 GB heap is appetite,
not need.

**The port's floor is now about 76 MB**, where on 2026-09-13 it did not go below
366 MB. Down to 96 MB its time does not move; at 64 MB the collector works on
2.6 cores and the pass takes 1.3× as long; at 48 MB it takes 3.2× and 5 cores,
still correct.

So the honest memory statement is: **the port holds 2.4× less at defaults, and
needs about 1.6× more at the floor** — 76 MB against 48. On 2026-09-13 those were
1.34× more and about 7× more.

## Reproducing

```bash
pwsh go/migration/scripts/run-oracle.ps1          # builds the Java side
cd go && go run ./cmd/bench -list files.txt -passes 3 -o go-timings.tsv
java -Xss8m -Xmx4g -cp <classpath> JavaBench files.txt 3 java-timings.tsv
```

`files.txt` should hold documents both implementations handle; `corpus -oracle`
is what establishes that. The other tables are the same two drivers with
`-passes 1 -throughput-only` and `1 throughput-only`, `-workers N` and a fourth
argument `N`, `GOMEMLIMIT` and `-Xmx`, each run as its own process so that its
CPU time and peak resident set are its own.

## Leads

Moved into [`PERFORMANCE-PLAN.md`](PERFORMANCE-PLAN.md), which profiles the 40
heaviest documents rather than one, and corrected there: what this list said
about `bufio.ReadByte` being the port's own code, and about inflate needing the
pure-Go rule revisited, is answered by "What the profiles say" and item 1 of
"The work".

What is left to look at, from the 2026-09-15 numbers: the four small veraPDF
files above 5×, and the floor, 76 MB against PDFBox's 48.
