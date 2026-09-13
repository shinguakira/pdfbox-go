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
in this repository and puts it over the same documents, and
`go run ./cmd/corpus -oracle` reports every disagreement.

Over **3,646 documents** — PDFBox's own regression files, veraPDF's ISO clause
tests, qpdf's damaged files, the SafeDocs parser traps:

```
open    both 3600, neither 46, behind 0, ahead 0
pages   0 disagree
text    both 3597, neither 3, behind 0, ahead 0
chars   3596 the same length, 1 not

1 of 3646 files disagree (0.03%)
```

There is no document in that corpus PDFBox reads and this port does not.
[`go/migration/TESTDATA.md`](go/migration/TESTDATA.md) has the corpus, how to
fetch it, and the one remaining disagreement.

Speed and memory
----------------

Measured by running both. There is no single number:

| | this port | PDFBox | |
| --- | ---: | ---: | --- |
| 3,597 documents, wall clock | 36.8 s | 5.8 s | 6.4× slower |
| the median document | 0.330 ms | 0.389 ms | **faster** |
| CPU time for the same work | 90.3 s | 56.5 s | 1.6× more |
| cores used | 1.10 | 2.86 | |
| cold start, one document | **71.6 ms** | 649.0 ms | **9.1× faster** |
| peak heap, default settings | 795 MB | 638 MB | 1.24× |
| minimum to ship | **15.6 MB** | 49.0 MB | **3.2× smaller** |

2,300 of the 3,597 documents are faster here; ten documents are 79% of the total
time, and most of that is `compress/flate` against PDFBox's native zlib — the
price of the pure-Go rule. The full analysis, including three ways of measuring
this that produce confident wrong answers, is in
[`go/migration/BENCHMARK.md`](go/migration/BENCHMARK.md).

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
[`go/migration/JAVA-BUGS.md`](go/migration/JAVA-BUGS.md) — 87 entries so far,
each saying what the Java does, what correct would be, where the Go carries it,
and how sure the author was. One branch, `track/java-bug-fixes`, then corrected
61 of them in the Go on purpose; every one of those says so at the site.

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
