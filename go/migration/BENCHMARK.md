# Speed and memory, against PDFBox

Measured 2026-09-13 by running both, on the same documents, on one machine.

`go/cmd/bench` and `migration/oracle/JavaBench.java` print the same numbers in
the same shape; [`scripts/run-oracle.ps1`](scripts/run-oracle.ps1) builds the
Java side. The unit of work is one document's worth of real use — open the file,
count the pages, extract the text — because the question is what a caller pays
for a document, not how fast a function is.

**The set is 3,597 documents, 226.8 MB**: every file in the corpus that *both*
implementations fully process. Timing a file one of them gives up on would score
"fast" for "stopped early".

## Headline

| | port | PDFBox | |
| --- | ---: | ---: | --- |
| total, 3,597 documents | 36,805 ms | 5,779 ms | 6.4× |
| documents per second | 97.7 | 622.4 | |
| per document, median | 0.330 ms | 0.389 ms | **port faster** |
| per document, p90 | 2.38 ms | 1.53 ms | 1.6× |
| per document, p99 | 30.6 ms | 12.0 ms | 2.6× |
| peak live heap, default settings | 794.9 MB | 593.6 MB | 1.34× |
| heap the OS was asked for | 803.8 MB | — | |

**"6.4× slower" is true of the total and false of almost every document.**
2,300 of the 3,597 are faster in the port, and the median is faster. The total
is what it is because **the ten slowest documents are 79% of the port's time**
(PDFBox's ten slowest are 40% of its own).

It is also only true of wall clock on a machine with cores to spare. Three other
measurements point the other way, and each has its own section below:

| | port | PDFBox | |
| --- | ---: | ---: | --- |
| CPU time for the same work | 90.3 s | 56.5 s | 1.6×, not 6.4× |
| cores used | 1.10 | 2.86 | |
| cold start, one document | 71.6 ms | 649.0 ms | **9.1× the other way** |
| minimum shipped | 15.55 MB | 49.0 MB | **3.2× the other way** |

There is no single number for "is the port slower than PDFBox". It is slower per
document on a fast machine, cheaper per document in CPU, an order of magnitude
quicker to start, and a third of the size.

## CPU, which is where the headline changes shape

Wall clock is not what a container is billed for. The same run, measured as
process CPU time:

| | port | PDFBox |
| --- | ---: | ---: |
| wall | 82.3 s | 19.7 s |
| user CPU | 85.9 s | 51.2 s |
| kernel CPU | 4.4 s | 5.3 s |
| **total CPU** | **90.3 s** | **56.5 s** |
| **cores used (CPU ÷ wall)** | **1.10** | **2.86** |

(One warmup pass plus one timed pass, both sides, so the wall figures are about
twice the per-pass numbers above.)

**PDFBox is 4.2× faster in wall clock and uses 1.6× less CPU.** The rest of its
advantage is bought with cores: the JIT compiler threads and the parallel
collector put it at 2.86 cores where the port sits at 1.10 — essentially one.

Which number matters depends entirely on the deployment. On an idle machine with
cores to spare, wall clock is the answer and PDFBox wins by 4.2×. Under a
one-core quota, the CPU column is the answer and the gap is 1.6×. Neither is the
"real" one.

### Where those cores go, and whether the port is missing something

The obvious reading of 2.86 against 1.10 is that PDFBox parallelises something
the port does not. It does not.

**PDFBox processes documents on one thread.** Its whole source has a single
`Thread`, in `IOUtils`, and it is a shutdown hook that deletes temporary
directories. No `ExecutorService`, no `parallelStream`, no `ForkJoinPool`. The
port has no goroutines in its ported packages either. On this they are equal.

The cores are the JIT, and constraining the JVM says so:

| | wall | CPU | cores |
| --- | ---: | ---: | ---: |
| default | 18.9 s | 50.9 s | 2.69 |
| `-XX:+UseSerialGC` | 18.5 s | 49.8 s | 2.68 |
| serial GC, one compiler thread | 25.6 s | 43.7 s | 1.71 |

Serialising the collector changes nothing, so it is not GC. Restricting the
compiler to one thread drops it to 1.71 cores — and costs 35% of the wall clock.
So the extra cores are not overhead being wasted: PDFBox is buying speed with
them, by compiling on other cores while its one application thread runs.

**And no, the port is not leaving a win on the table that PDFBox has taken.**
Both were given a worker pool over the same corpus — `-workers` here,
`ExecutorService` there — and they scale the same:

| workers | port | PDFBox | port ÷ PDFBox |
| ---: | ---: | ---: | ---: |
| 1 | 38,299 ms | 6,625 ms | 5.8× |
| 4 | 18,806 ms | 3,533 ms | 5.3× |
| 12 | 17,504 ms | 2,958 ms | 5.9× |
| **speedup** | **2.19×** | **2.24×** | |

The output was identical at every worker count on both sides, which is also the
answer to whether the port has shared mutable state getting in the way: it does
not.

Parallelism helps both and closes nothing, because both hit the same ceiling.
Ten documents are 79% of the port's time and the slowest is 10.5 s on its own —
no number of workers gets below one document. The tail is the problem, and the
tail is `compress/flate`.

The memory cost is not symmetric, though: the port's peak went 729 MB → 1,605 MB
across those runs, PDFBox's 611 MB → 760 MB. Workers are cheaper for PDFBox than
for the port.

## Starting up

Process launch to one small document extracted, ten runs, best and mean:

| | port | PDFBox |
| --- | ---: | ---: |
| best | **71.6 ms** | 649.0 ms |
| mean | 82.7 ms | 681.1 ms |

**9.1× the other way.** There is no JVM to start, no classes to load and nothing
to JIT. For a command-line tool or a per-request process this reverses the
throughput result completely: PDFBox needs about 640 ms of startup before it is
faster at anything, which is roughly two thousand median documents' worth.

The warmup is visible in the passes, too. PDFBox's three timed passes ran 6,677
→ 5,919 → 5,778 ms as the JIT settled; the port's were 38,571 → 38,418 → 38,431,
flat from the first.

## What has to be shipped

| | port | PDFBox |
| --- | ---: | ---: |
| the library | — | 1.75 MB of classes |
| its resources | — | 4.45 MB (glyph lists, AFMs, CMaps) |
| log4j-api | — | 0.34 MB |
| Bouncy Castle, if signing or public-key encryption | — | 11.74 MB |
| a runtime | included | 42.5 MB jlink minimum, 302 MB for this JDK |
| **binary, stripped** | **15.55 MB** | — |
| binary, unstripped | 20.48 MB | — |
| **total, minimum** | **15.55 MB** | **49.0 MB** |

One file against a tree, and 3.2× smaller at the minimum — more against a full
JDK. The port's number is also the whole story: no runtime to install, no
classpath, no version to match.

## Memory, as the operating system sees it

Peak resident set for the same run, which is the number a container limit is
compared against:

| | peak RSS |
| --- | ---: |
| port | 854 MB |
| PDFBox, `-Xmx4g` | 740 MB |
| PDFBox, `-Xmx256m` | 399 MB |

RSS is above the heap figures below because it includes what neither heap
accounts for — the JVM's metaspace, code cache and thread stacks, the Go
runtime's own arenas.

## Four ways to measure this that do not work

Each of these was tried first and produced a confident wrong answer.

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
is never smaller than the real peak. Review caught it. Every PDFBox heap figure
on this page was measured again on 2026-09-15 by sampling the whole heap, the
way the Go side samples its own: the default run went from 638.4 MB to
593.6 MB, and the capped runs under "Memory" fell by between 7% and 29%. The
times beside those figures are from the first runs.

**Comparing two default configurations.** Both runtimes use what they are given.
PDFBox peaks at 513 MB under `-Xmx4g` and at 94 MB under `-Xmx96m`, extracting
identical text either way. A default-vs-default number compares two GC
settings, not two implementations. See "Memory" below for the number that means
something.

And one presentational error worth naming, because it flattered the port:
comparing the two `max` columns. Those are *different documents* — the port's
slowest is not PDFBox's slowest — and dividing them gave "20×" where the same
document is **112×**.

## Where the time actually goes

The same document, both sides, slowest first:

| document | port | PDFBox | |
| --- | ---: | ---: | ---: |
| `PDFBOX-4423-000746.pdf` | 10,485 ms | 95.3 ms | **112×** |
| `PDFBOX-4418-000671.pdf` | 6,760 ms | 180.4 ms | 37× |
| `PDFBOX-3964-c687766d…pdf` | 3,137 ms | 131.5 ms | 24× |
| `PDFBOX-3947-670064.pdf` | 1,484 ms | 395.3 ms | 3.8× |
| `PDFBOX-3949-MKFYUG…pdf` | 1,337 ms | 541.8 ms | 2.5× |

The last row is PDFBox's own worst document. Where PDFBox struggles, the port
struggles similarly; the gap only opens on documents that are hard for the port
specifically.

By ratio rather than by time, over the files above 0.05 ms:

```
port faster (<1×)   2300
1–2×                1114
2–5×                 153
5–20×                 20
over 20×              10
```

## What was fixed

Two defects, both found by this benchmark, both verified against the oracle
afterwards (`0 of N files disagree`).

### The predictor's row, in `filter`

`qpdf/issue-1688a.pdf` is **531 bytes** and took **435 ms**, against PDFBox's
0.3 ms — 1,449×. 83% of it was the runtime zeroing memory. The file declares:

```
/DecodeParms << /Predictor 2 /Colors 536870913 /Columns 1 /BitsPerComponent 8 >>
```

The port worked the row out as 536,870,913 bytes, and `decodePredictor`
allocated two of them — **a gigabyte of zeroed memory from a 531-byte file**.

**The first fix had the right symptom and the wrong cause.** It saw that
`536870913 * 8` wraps to 8 in Java's 32-bit `int`, made the port's arithmetic
wrap the same way, and wrote down that PDFBox's row is one byte. It is not.
`Predictor.wrapPredictor` reads `/Colors` through `Math.min(..., 32)` before the
row length is worked out, so PDFBox's row for this file is 32 bytes. The wrap
took the time to 0.71 ms and still gave a different answer from the reference:
over 40 bytes of data, one-byte rows and 40 bytes out, where PDFBox has 32-byte
rows and 64 bytes out. Review caught it.

What is there now is the clamp, where Java has it, and Java's 32-bit arithmetic
all through the predictor — `/BitsPerComponent` and `/Columns` are not clamped
and can still wrap. Matching PDFBox's output case by case turned up the rest: the
last row completed with zeros, a row's algorithm byte read as a signed Java
byte, and a predictor of 0 or below passing the data through whatever the other
parameters say. `TestPredictorAnswersWhatPDFBoxAnswers` holds PDFBox's own
output for each case; JAVA-BUGS 88 is the one case where PDFBox has no answer.

`conventions/java-to-go.md` already said which width to use: *"Java int is
32-bit: use int32 where the width is load-bearing (format fields,
overflow-sensitive arithmetic)"*. The width was load-bearing, and it was not the
whole of it.

PDFBox's own `SECURITY.md` puts disproportionate resource consumption from small
attacker-controlled inputs in scope, so this was not only a speed defect.

### The GPOS table, in `fontbox/ttf`

The port's table switch is identical to `TTFParser.readTable` **except for one
extra case**: `GPOS`. PDFBox has no glyph positioning reader at all — GSUB is
there, GPOS is not — and an unknown tag becomes a plain `TTFTable` that reads
nothing. So `parseTables` costs Java nothing for GPOS, and cost the port a full
parse on every font load.

Not a slower version of the same work: **work PDFBox does not do**, on a path
that never uses it. Extracting text from `PDFBOX-5927.pdf`, a one-megabyte
document, the port held 464 MB against PDFBox's 59.4 MB, and 52 MB of that was
`readPairSet` — kerning pairs, parsed while extracting text. The only caller of
`GPOS()` in the tree is the shaper, and `table()` already reads on demand.

Left until asked for. **464 MB → 156 MB on that document**, and across the whole
corpus:

| | before | after |
| --- | ---: | ---: |
| peak live heap | 1,651.0 MB | **794.9 MB** |
| heap from the OS | 2,731.8 MB | **803.8 MB** |
| total allocated | 118,373 MB | 104,883 MB |
| total time | 38,691 ms | 36,806 ms |

Memory is where it paid; the throughput barely moved, because those documents
were under a second of thirty-eight.

## What was not fixed, and why

**`compress/flate`.** The 112× document is inflate-bound: 26% of its time in
`huffSym`, another 10% in `huffmanBlock`, 8% in `huffmanDecoder.init`. PDFBox's
`Inflater` is native zlib through the JVM; Go's `compress/flate` is pure Go.
This is the price of the pure-Go rule, not a defect, and it is most of the
remaining gap on the heavy documents. Nothing in the corpus suggests the port
inflates *wrongly* — only that it inflates in Go.

**`encoding.newEncodingBase`, 58.75 MB on `PDFBOX-4418-000671.pdf`.** The port
pre-sizes its two maps to 250, and so does Java — `new HashMap<>(250)` — so the
port is faithful here. 15,422 of them were live at the peak, which is worth
understanding, but `PDResources.GetFont`'s cache is the same shape as Java's and
PDFBox needs 283 MB for the same document. A 1.7× gap, the smallest of the
three, and proving anything needs instrumentation rather than a profile.

## Memory

Defaults compare settings. The number that compares implementations is how far
each can be squeezed while still producing identical output. Measured on the 40
heaviest documents, which are 92% of the time and 86% of the peak:

| cap | PDFBox | | port | |
| --- | ---: | --- | ---: | --- |
| 4 GB / none | 3,977 ms | peak 513 MB | 33,461 ms | peak 473 MB |
| 512 MB | 3,565 ms | peak 379 MB | 33,461 ms | peak 473 MB |
| 256 MB | 3,918 ms | peak 228 MB | 64,833 ms | peak 366 MB |
| 128 MB | 4,056 ms | peak 127 MB | 104,682 ms | peak 368 MB |
| 96 MB | 4,481 ms | peak 94 MB | — | |
| 64 MB | 5,323 ms | peak 63 MB | — | |
| 48 MB | 8,468 ms | peak 48 MB | — | |
| 32 MB | **fails** — output truncated | | — | |

**PDFBox completes in 48 MB.** Its 513 MB under a 4 GB heap is appetite, not
need.

**The port does not go below about 366 MB.** `GOMEMLIMIT` is a soft limit, so it
does not fail — it collects harder and harder, and 128 MB costs 3× the time
while still peaking at 368 MB. Something holds that much live and the collector
cannot help. Output stayed correct at every level on both sides.

So the honest memory statement is: **1.34× on defaults, about 7× on the floor.**
The floor is the one that would matter in a container.

## Reproducing

```bash
pwsh go/migration/scripts/run-oracle.ps1          # builds the Java side
cd go && go run ./cmd/bench -list files.txt -passes 3 -o go-timings.tsv
java -Xss8m -Xmx4g -cp <classpath> JavaBench files.txt 3 java-timings.tsv
```

`files.txt` should hold documents both implementations handle; `corpus -oracle`
is what establishes that.

## Leads, in the order they look worth taking

1. **What holds 366 MB.** It sets the floor, and the floor is the worst number
   on this page.
2. **`bufio.ReadByte` at 7.8%** of the 112× document. Reading the predictor a
   byte at a time through an interface call is not free, and unlike inflate it
   is the port's own code.
3. **The 15,422 encodings**, if the floor turns out to be there.
4. **Inflate**, only if someone is willing to revisit the pure-Go rule. Nothing
   else on this page is worth that.
