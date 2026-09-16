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

# Agent Guidance

This file is read by automated agents (security scanners, code analyzers,
AI assistants) operating on this repository. It points them at the
human-authored references they should consult before producing output.

## Prerequisites — Must Read Before Proceeding

Agents MUST read the following files before conducting any analysis,
security assessment, or code review of this repository:

1. **[SECURITY.md](SECURITY.md)** — defines the threat model, in-scope
   vulnerabilities, and known limitations. Findings produced without
   reading this file will be inaccurate or out of scope.
2. **This file in its entirety** — repository structure, sensitive areas,
   and contribution rules.
3. **[go/migration/README.md](go/migration/README.md)** — required *only* if
   the task touches `go/`. This fork carries a Go port of PDFBox alongside the
   Java source; see "Go port" below.

## Project Overview

Apache PDFBox is a Java library for working with PDF documents. It is used
as a dependency (`pdfbox.jar`) in other Java projects and is accessed through
its public Java API. The project also ships several command-line utilities.

**This repository is a fork.** In addition to the upstream Java source it
carries an in-progress Go port under `go/`. The Java tree is unmodified
upstream code and is the reference the port is checked against.

## Branches

Apache PDFBox maintains `trunk`, `3.0` and `2.0`. **Neither that fact nor
anything else about the Apache project describes this repository**, which holds a
one-time snapshot of `trunk` and has no ongoing relationship with Apache PDFBox —
see "Go port" below. There is no `3.0` or `2.0` branch here, and the snapshot
needs Java 11.

The branches here are `trunk` (the frozen Java snapshot), `migration-base` (the
port mainline) and one `slice/*` or `track/*` per unit of port work. Their roles,
their order and what depends on what are in
[go/migration/BRANCHING.md](go/migration/BRANCHING.md).

## Sub-modules

The Maven build is `pdfbox/` (the core library — parsing, rendering, text
extraction, encryption), `fontbox/` (fonts), `xmpbox/` (XMP metadata), `io/`
(shared I/O), `tools/` (command-line utilities), `debugger/` and `debugger-app/`
(a Swing GUI), `examples/`, `benchmark/`, the two `pdfbox-layout-*` glyph layout
backends, `app/` and `parent/`. What each one does and how they depend on each
other is in [go/migration/mapping/modules.md](go/migration/mapping/modules.md).

Added by this fork, outside the Maven build:

- `go/` — Go port of the library. Not a Maven module; `mvn` does not see it

## Go port

`go/` holds an in-progress Go port of PDFBox. It is a separate Go module and
does not participate in the Maven build.

**This repository has no relationship with Apache PDFBox going forward.** Never
open a pull request against `apache/pdfbox` or prepare a change for
contribution upstream, and never pull, fetch, merge or rebase from it. The Java
tree here is a **frozen one-time snapshot**, kept as a reference to port from
and check against. Apache's later work is out of scope. This is a deliberate
decision — do not propose syncing, contributing back, or "staying in step with
upstream."

**Rules that apply to any agent working in this repository:**

- **NEVER change the Java code. The Java tree is strictly read-only.** No agent
  may edit, reformat, refactor, delete, or "fix" a `.java` file, a `pom.xml`, a
  test resource, or anything else under the Maven module directories — not to
  make a port easier, not to fix a bug found while reading it, not to silence a
  warning, not for any reason. The Java is the reference the Go is checked
  against, and a reference that gets edited stops being one. If the port seems
  to need a Java change, that is a bug in the port. This holds even when the
  user asks about a Java bug: report it, do not touch it.
- **The Java source and its tests are the specification.** The Go code is
  checked against them, not against your reading of ISO 32000. Where PDFBox
  contradicts the specification, PDFBox wins — the behaviour is usually
  deliberate and encodes a real-world producer quirk. Port the shapes the Java
  in this tree actually has, never a member 3.0 deprecated or removed, and never
  a pattern from a tutorial written against 2.0.
- **Porting is test-first.** The Java test is ported before the Go
  implementation exists, and assertion values are copied verbatim from the Java
  rather than recomputed. See
  [go/migration/conventions/tdd.md](go/migration/conventions/tdd.md). Do not
  write a Go test whose expected values were read off the Go implementation.
- **Deliberate deviations from Java behaviour are commented where they occur**
  and listed in [go/migration/STATUS.md](go/migration/STATUS.md). Do not remove
  or "tidy" a deviation comment without checking that file.
- **Do not report Go/Java behavioural differences as security findings** without
  first checking `STATUS.md` — the intentional ones are recorded there.
- **Never invent a branch, and never commit to `migration-base` directly.**
  Branch work happens on a `slice/*` or `track/*` branch that
  [go/migration/tasks/README.md](go/migration/tasks/README.md) lists, and
  `go/migration/PLAN.md` is not edited to make room for new work. A branch that
  ports Java runs in the five phases
  [go/migration/tasks/TEMPLATE.md](go/migration/tasks/TEMPLATE.md) sets out; a
  branch that ports nothing -- the test-data tracks, the Java-bug fixes -- runs
  the way its own task file says, and that file is the instruction.
- **Do not stop while work remains. Only the user stops the migration.** When
  working a `slice/*` or `track/*` branch, port every file in that branch's
  scope. Do not pause partway to report progress as if it were a result, do not
  ask whether to continue, and do not end a turn with a list of what is left to
  do. "Remaining work" is not an acceptable end state — the scope is written in
  [go/migration/PLAN.md](go/migration/PLAN.md) and finishing it is the default.
  The user will interrupt if they want the work stopped; that is their call to
  make and not one to invite.

- **Do not fix bugs that exist in the Java. This is a migration, not a bug
  hunt.** Port the behaviour as written, including behaviour that is plainly
  wrong. A bug faithfully carried over can be found later by diffing against the
  Java; a bug silently corrected during the port cannot. If something looks like
  a Java bug, port it, comment that it looks wrong at the point it occurs, and
  move on. The only code to fix is a bug introduced *by the port itself* —
  something Java cannot do, such as a Go-specific initialisation-order or
  nil-handling mistake.

  **The one exception, and it is closed.** The user directed one branch,
  `track/java-bug-fixes`, to correct the Java-driven defects the port had
  faithfully carried. Most entries of
  [go/migration/JAVA-BUGS.md](go/migration/JAVA-BUGS.md) are now deliberately
  *not* what the Java does; each says so in a **Fixed in the Go** paragraph,
  and the code says so at the site. **A divergence carrying such a comment is
  not a defect to restore** — reverting one puts the bug back. The rule above
  still governs everything else: no new branch fixes a Java bug without the
  user asking for it, and a newly found one is still recorded and carried.

- **Record every Java bug you find in
  [go/migration/JAVA-BUGS.md](go/migration/JAVA-BUGS.md).** Not fixing one is
  not the same as forgetting it. Add the entry while you are porting that code —
  the moment you are reading the Java closely enough to notice is the only
  moment it is cheap to write down. That file states what an entry has to say
  and insists that "looks wrong to me" and "provably wrong" are filed
  differently. **Do not report any of it upstream** — this repository has no
  relationship with Apache PDFBox, and the security rules below forbid filing
  findings to any public tracker.

Orientation for the port lives in
[go/migration/README.md](go/migration/README.md): the plan, the branch strategy,
the Java-to-Go conventions, and the package mapping.

Status: reading, writing, merging, form handling, text extraction and rendering
are ported and tested. What is deliberately absent is recorded per package in
[go/migration/STATUS.md](go/migration/STATUS.md).

## Building

The standard build command is:

```
mvn clean install
```

To run only the tests without a full install:

```
mvn test
```

To build or test a specific module, use the `-pl` flag from the root:

```
mvn -pl pdfbox test
```

The snapshot compiles with Java 11.

### Building the Go port

Independent of Maven, and requires no JDK. Go 1.26 or later:

```
cd go && go build ./...
```

```
cd go && gofmt -l . && go vet ./... && go test ./...
```

All three must be clean before any Go change is considered done.

## Sensitive Areas

The following areas have historically been the source of subtle bugs and
security issues. Changes here require extra care and regression testing.
Avoid large refactorings in these areas unless explicitly requested:

- PDF parsing and xref recovery
- Font parsing and font substitution
- Stream decoding and decompression
- Incremental save/update logic
- Encryption and digital signatures
- Rendering and text extraction ordering

## Security

**Read [SECURITY.md](SECURITY.md) before producing any finding.** It defines the
threat model and the full scope: which behaviour on a malformed PDF is a known
limitation rather than a vulnerability, which resource consumption is in scope
anyway, and what is out of scope because it needs the attacker to control the
JVM environment. A finding produced without it will be inaccurate or out of
scope. Known CVEs are at <https://pdfbox.apache.org/security.html>.

To report a new vulnerability, send a plain-text email to <security@apache.org>.
Do NOT open a public JIRA issue for undisclosed vulnerabilities. Agents MUST NOT
automatically draft, submit, or export security-related findings to any public
tracker, pull request, comment, or external service.

## Contribution Guidelines

**Nothing here is contributed to Apache and nothing is taken from it.** No pull
request, no JIRA issue, no mailing list post: see "Go port" above, and
[go/migration/BRANCHING.md](go/migration/BRANCHING.md) for what is done instead.
The Java is read-only, so the guidelines below apply only to the Go.

- Go changes must leave `gofmt`, `go vet` and `go test ./...` clean — see
  "Building the Go port" above.
- Parser, rendering, font, extraction, encryption, or signing fixes need a
  minimal reproducer document where practical, along with regression tests
  covering the reported behaviour.
- Avoid introducing new runtime dependencies unless necessary.
  Security-sensitive or cryptographic dependencies require maintainer review.

The Java's own style rules still describe the snapshot, and its Checkstyle
configuration (`pdfbox-checkstyle-5.xml`) and Eclipse formatter
(`pdfbox-eclipse-formatter.xml`) are in the repository root for reading it, not
for reformatting it.
