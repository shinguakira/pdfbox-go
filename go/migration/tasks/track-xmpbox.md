# Implementation Plan

Track — `xmpbox`. XMP metadata parsing.

**Branch: `track/xmpbox`** — from and back to `migration-base`.
**Depends on nothing.** It can be worked in parallel with any slice, including
`slice/1`. `pdfbox` hands back a raw metadata stream and `xmpbox` parses it
separately.

## Rules — do not break these

- **NEVER change the Java.** No `.java`, no `pom.xml`, no test resource, for any
  reason. It is the reference; a reference that gets edited stops being one.
- **NEVER fix a bug that is in the Java.** Port it as written, comment it where
  it occurs, and record it in `migration/JAVA-BUGS.md`.
- **NEVER create a branch that is not in the migration plan**, and never add one
  to the plan's list.
- **NEVER change `migration/PLAN.md`.**
- **NEVER commit to `migration-base` directly.**
- **NEVER touch `apache/pdfbox`** — no PR, no pull, fetch, merge or rebase.
- **Do not stop while work in this branch's scope remains.**

## How each unit of work runs

Five phases, in this order, never overlapping. See
[`TEMPLATE.md`](TEMPLATE.md) for what each phase means.

A — write the test · B — port the implementation · C — run and fix ·
D — adversarial review · E — user feedback, strict test first

## Scope

| Java module | Files | Java tests |
| --- | ---: | ---: |
| `xmpbox` | 74 | 28 |

`migration/mapping/packages.tsv` renames one package: `org.apache.xmpbox.type`
becomes `xmpbox/xmptype`, because `type` is a Go keyword.

Java parses XMP with DOM. Go has `encoding/xml`. Whichever way that goes, it is
a deviation and belongs in `STATUS.md`.

---

# Phase A — Write the tests

- [ ] A1. `xmpbox/type` — port its Java tests
- [ ] A2. `xmpbox/schema` — port its Java tests
- [ ] A3. `xmpbox/xml` — port its Java tests, including the parser round trips
- [ ] A4. The rest of the 28

---

# Phase B — Port the implementation

- [ ] B1. `xmpbox/xmptype` — the type system
- [ ] B2. `xmpbox/schema` — the schemas
- [ ] B3. `xmpbox/xml` — the parser and the serialiser
- [ ] B4. `XMPMetadata` and the rest of the root package

---

# Phase C — Run and fix

- [ ] C1. `gofmt -l .` clean
- [ ] C2. `go vet ./...` clean
- [ ] C3. `go test ./...` green
- [ ] C4. Record every Java bug found in `migration/JAVA-BUGS.md`
- [ ] C5. Update `migration/STATUS.md`

---

# Phase D — Adversarial review

敵対的レビュー. See [`TEMPLATE.md`](TEMPLATE.md) for D1–D6. Additionally:

- [ ] D7. Check the XML handling difference honestly
  - Java's DOM and Go's `encoding/xml` differ on namespaces, on attribute
    order, on whitespace, and on what they accept from a malformed document.
  - Every one of those differences is a deviation to record, not a detail.

- [ ] D8. Check the round trip against Java's output
  - Serialising with the port and parsing with the port proves nothing.

---

# Phase E — User feedback

See [`TEMPLATE.md`](TEMPLATE.md) for E1–E4. E3 is the strict-test rule.

---

# Blocked

- [ ] Nothing. This is the one part of the project with no ordering constraint
      at all.
