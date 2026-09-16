<!---
  Licensed to the Apache Software Foundation (ASF) under one or more
  contributor license agreements.  See the NOTICE file distributed with
  this work for additional information regarding copyright ownership.
  The ASF licenses this file to You under the Apache License, Version 2.0
  (the "License"); you may not use this file except in compliance with
  the License.  You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.
--->

pdfbox-go
===================================================

A Go port of [Apache PDFBox](https://pdfbox.apache.org/), living beside the Java
it was ported from.

Every line of Go here is newly written. None of it is generated, translated by a
tool, or bridged to a JVM at runtime. What is *not* rewritten is the design: the
algorithms, the control flow, the field-by-field state and the format quirks are
carried across from the Java, with the original open alongside. A port is close
to 100% new source and close to 0% new design.

**Pure Go, and no cgo.** No wrapped C library, no bundled WebAssembly, no
subprocess. `go build` is the whole build.

Status
------

Reading, writing, merging, form handling, text extraction and rendering are in.
Per-package progress, including what is deliberately absent and why, is in
[`go/migration/STATUS.md`](go/migration/STATUS.md).

The port is checked against the Java rather than against a reading of ISO 32000.
Where PDFBox contradicts the specification, PDFBox wins: the behaviour is usually
deliberate and encodes a real-world producer quirk.

How closely it matches
----------------------

PDFBox itself is the oracle, and it is run rather than consulted.
[`scripts/run-oracle.ps1`](go/migration/scripts/run-oracle.ps1) compiles the Java
in this repository and puts it over the same documents, and `cmd/corpus -oracle`
reports every disagreement.

Over **5,090 documents** — PDFBox's own regression files, veraPDF's ISO clause
tests, qpdf's damaged files, the SafeDocs parser traps, and every PDF pdf.js
tests with:

```
open    both 5037, neither 53, behind 0, ahead 0
pages   0 disagree
text    both 5032, neither 5, behind 0, ahead 0
chars   5030 the same length, 2 not

2 of 5090 files disagree (0.04%)
```

There is no document in that corpus PDFBox reads and this port does not. The two
that disagree are two bugs in the Java that this port fixes on purpose.
That run is of 2026-09-15; [`go/migration/TESTDATA.md`](go/migration/TESTDATA.md)
is where the corpus, the commands that produce this table, the current result
and both of those disagreements live.

Speed and memory
----------------

Measured by running both, on 2026-09-15, and summarised from
[`go/migration/BENCHMARK.md`](go/migration/BENCHMARK.md), which carries the
machine, the method and the rest. There is no single number:

| | this port | PDFBox | |
| --- | ---: | ---: | --- |
| 3,597 documents, one worker, best pass | 8.2 s | 5.6 s | 1.46× slower |
| the same, four workers | **2.7 s** | 3.2 s | **1.2× faster** |
| the median document | 0.333 ms | 0.394 ms | **faster** |
| CPU time, a warmup and one pass | **17.1 s** | 48.8 s | **2.9× less** |
| cores used | 1.11 | 2.93 | |
| cold start, one document | **59 ms** | 615 ms | **10.4× faster** |
| peak heap, PDFBox at `-Xmx4g` | **250 MB** | 590 MB | **2.4× smaller** |
| lowest peak heap that still extracts 40 heavy documents unchanged | 76 MB | 48 MB | 1.6× more |
| minimum to ship | **15.6 MB** | 49.0 MB | **3.1× smaller** |

2,410 of the 3,597 documents are faster here, and none of the ten slowest takes
twice as long as in PDFBox. That is after the performance work of 2026-09-15,
which [`go/migration/PERFORMANCE-PLAN.md`](go/migration/PERFORMANCE-PLAN.md)
records — what it started from, what each change bought and where the plan was
wrong. [`go/migration/BENCHMARK.md`](go/migration/BENCHMARK.md) carries the rest
of the numbers, including four ways of measuring this that produce confident
wrong answers.

Building
--------

Go 1.26 or later. No JDK, no Maven.

```bash
cd go
go build ./...
```

Before any change is considered done:

```bash
cd go && gofmt -l . && go vet ./... && go test ./...
```

The command-line tool is `go/cmd/pdfbox`, a port of PDFBox's own:

```bash
cd go && go run ./cmd/pdfbox export:text -i document.pdf
```

Layout
------

| Path | What |
| --- | --- |
| `go/` | The port. A separate Go module; Maven does not see it |
| `go/pdfbox`, `go/fontbox`, `go/xmpbox`, `go/pdfio`, `go/tools` | The ported modules |
| `go/cmd/pdfbox` | The command-line tool |
| `go/cmd/corpus`, `go/cmd/bench` | Migration tooling: corpus scoring and benchmarking. Not ports |
| `go/migration/` | The plan, the conventions, the status, the findings. No Go source |
| `pdfbox/`, `fontbox/`, `xmpbox/`, `io/`, `tools/`, … | The Java, frozen |

The Java tree
-------------

**It is a one-time snapshot and it is read-only.** It is the reference the port
is checked against, and a reference that gets edited stops being one. Nothing
under the Maven module directories is modified, for any reason — including to
fix a bug found while reading it. A bug faithfully carried can be found later by
diffing against the Java; a bug silently corrected during the port cannot.

Java bugs found while porting are recorded in
[`go/migration/JAVA-BUGS.md`](go/migration/JAVA-BUGS.md), each entry saying what
the Java does, what correct would be, where the Go carries it, and how sure the
author was. That file carries the count. One branch, `track/java-bug-fixes`, then
corrected most of them in the Go on purpose, and every one of those says so both
in its entry and at the site.

**This repository has no relationship with Apache PDFBox going forward.** No pull
requests are opened against `apache/pdfbox`, nothing is pulled or merged from it,
and none of the findings here are reported upstream. That is a deliberate
decision, not an oversight — see
[`go/migration/BRANCHING.md`](go/migration/BRANCHING.md).

Where to start reading
----------------------

| File | For |
| --- | --- |
| [`go/migration/README.md`](go/migration/README.md) | What the port is, and how it is done |
| [`go/migration/STATUS.md`](go/migration/STATUS.md) | What is finished, what is not, and why |
| [`go/migration/conventions/java-to-go.md`](go/migration/conventions/java-to-go.md) | How Java constructs are translated. Read before porting anything |
| [`go/migration/conventions/tdd.md`](go/migration/conventions/tdd.md) | The rule the port runs on: the Java test is ported before the Go exists |
| [`AGENTS.md`](AGENTS.md) | The rules, for automated agents and humans alike |

Licence
-------

Apache License 2.0, as PDFBox is. See [`LICENSE.txt`](LICENSE.txt) and
[`NOTICE.txt`](NOTICE.txt).
