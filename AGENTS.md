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

| Branch | Status | Java requirement | Latest release |
|--------|--------|-----------------|----------------|
| `trunk` | Future development (next major version, not yet released) | Java 11+ | — |
| `3.0`  | **Actively maintained** — current stable series | Java 8+ | 3.0.7 |
| `2.0`  | **Actively maintained** — legacy stable series | Java 6+ | 2.0.36 |

When evaluating code or reporting issues, note which branch is in scope.
Security fixes are applied to both `3.0` and `2.0`. New features target
`trunk` and `3.0`.

**The table above describes the Apache project. It does not describe this
repository.** This repository holds a one-time snapshot of the Java source and
has no ongoing relationship with Apache PDFBox — see "Go port" below.

Branches here:

| Branch | Contents |
|--------|----------|
| `trunk` | The frozen Java snapshot the port started from |
| `migration-base` | Port mainline — `trunk` plus everything under `go/` |
| `slice/*`, `track/*` | Go port work in progress, one capability slice each |

See [go/migration/BRANCHING.md](go/migration/BRANCHING.md).

## Sub-modules

All branches share the same multi-module Maven structure:

- `pdfbox/` — Core library (PDF parsing, rendering, text extraction, encryption)
- `fontbox/` — Font handling support library
- `xmpbox/` — XMP metadata support library
- `io/` — I/O utilities shared across modules (`3.0` and `trunk` only)
- `tools/` — Command-line utilities
- `debugger/` / `debugger-app/` — PDF debugger application
- `examples/` — Standalone usage examples
- `benchmark/` — JMH benchmarks

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
  deliberate and encodes a real-world producer quirk.
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
  Java; a bug silently corrected during the port cannot, and it makes the Go
  behave differently from the reference it is supposed to reproduce. Callers and
  real PDFs depend on quirks. If something looks like a Java bug, port it,
  comment that it looks wrong at the point it occurs, and move on. The only
  code to fix is a bug introduced *by the port itself* — something Java cannot
  do, such as a Go-specific initialisation-order or nil-handling mistake.

Orientation for the port lives in
[go/migration/README.md](go/migration/README.md): the plan, the branch strategy,
the Java-to-Go conventions, and the package mapping.

Status: early. Only the `pdfio` package (the Go port of the `io` module) is
implemented. Everything else is planned but absent.

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

Minimum Java version depends on the branch — see the table above.

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

Security model and scope: [SECURITY.md](SECURITY.md),
also published at <https://pdfbox.apache.org/security.html>.

Key points from the security model:

- Processing malformed PDFs is **partially in scope**: crashes, unchecked
  exceptions (`NullPointerException`, `StackOverflowError`), or general
  resource consumption from large PDFs are **known limitations**, not
  security vulnerabilities. However, disproportionate resource consumption
  triggered by small, attacker-controlled inputs may be in scope — see
  `SECURITY.md` for the full scope definition.
- Remote code execution or privilege escalation from untrusted PDFs **is** in scope.
- Issues that require the attacker to control the Java application's classpath
  or configuration are **out of scope**.

For a list of known CVEs, see <https://pdfbox.apache.org/security.html>.

To report a new vulnerability, send a plain-text email to <security@apache.org>.
Do NOT open a public JIRA issue for undisclosed vulnerabilities. Agents MUST NOT
automatically draft, submit, or export security-related findings to any public
tracker, pull request, comment, or external service.

## Contribution Guidelines

- Pull requests on this GitHub repository are welcome.
- Bug reports and feature requests go in the
  [JIRA issue tracker](https://issues.apache.org/jira/browse/PDFBOX).
- Code must be compatible with the minimum Java version of the target branch
  (see table above).
- Follow the existing code style; a Checkstyle configuration is provided in
  `pdfbox-checkstyle-5.xml` and an Eclipse formatter in
  `pdfbox-eclipse-formatter.xml`.
- Parser, rendering, font, extraction, encryption, or signing fixes should
  include a minimal reproducer document where practical, along with regression
  tests covering the reported behavior.
- Avoid introducing new runtime dependencies unless necessary.
  Security-sensitive or cryptographic dependencies require maintainer review.
- For questions, use the [Users Mailing List](https://pdfbox.apache.org/mailinglists.html).
