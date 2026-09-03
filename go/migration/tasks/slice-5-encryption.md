# Implementation Plan

Slice 5 — encrypted documents.

**Branch: `slice/5-<name>`** — from and back to `migration-base`.
Depends on `slice/1` only. Independent of slices 2, 3, 4, 6, 7 and 8.

## Rules — do not break these

- **NEVER change the Java.** No `.java`, no `pom.xml`, no test resource, for any
  reason. It is the reference; a reference that gets edited stops being one.
- **NEVER fix a bug that is in the Java.** Port it as written, comment it where
  it occurs, and record it in `migration/JAVA-BUGS.md`. **This matters most
  here:** a "fix" to a cryptographic path changes what documents open.
- **NEVER create a branch that is not in the migration plan**, and never add one
  to the plan's list.
- **NEVER change `migration/PLAN.md`.**
- **NEVER commit to `migration-base` directly.**
- **NEVER touch `apache/pdfbox`** — no PR, no pull, fetch, merge or rebase.
- **Do not stop while work in this branch's scope remains.**

`AGENTS.md` marks encryption a sensitive area. Port line for line.

## How each unit of work runs

Five phases, in this order, never overlapping. See
[`TEMPLATE.md`](TEMPLATE.md) for what each phase means.

A — write the test · B — port the implementation · C — run and fix ·
D — adversarial review · E — user feedback

## Scope

| Java package | Files | Java tests |
| --- | ---: | ---: |
| `pdmodel/encryption` | 19 | 0 in that package |

The tests are not where the source is. They are in
`pdfbox/src/test/java/org/apache/pdfbox/encryption/`:
`TestSymmetricKeyEncryption` and `TestPublicKeyEncryption`, with fixtures in
`pdfbox/src/test/resources/org/apache/pdfbox/encryption/`.

Go covers most of the primitives: `crypto/aes`, `crypto/rc4`, `crypto/sha256`,
`crypto/md5`, `crypto/x509`. Port the key derivation, not the ciphers.

---

# Phase A — Write the tests

- [ ] A1. Port `TestSymmetricKeyEncryption` — RC4 and AES, 40/128/256 bit
- [ ] A2. Port `TestPublicKeyEncryption` — certificate-based
- [ ] A3. Write from source for the 19 classes the two tests do not reach
  - Name which ones those are before writing, so the gap is visible

---

# Phase B — Port the implementation

- [ ] B1. The security handler base and the registry
  - `SecurityHandler`, `SecurityHandlerFactory`, `ProtectionPolicy`
- [ ] B2. Standard security — password
  - `StandardSecurityHandler`, `StandardProtectionPolicy`, `StandardDecryptionMaterial`,
    `AccessPermission`
- [ ] B3. Public key security — certificate
  - `PublicKeySecurityHandler`, `PublicKeyProtectionPolicy`,
    `PublicKeyDecryptionMaterial`, `PublicKeyRecipient`
- [ ] B4. The crypt filters and the rest of the package
- [ ] B5. Wire decryption into the parser — an encrypted document must open

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

- [ ] D7. Check every byte-level operation
  - Java `byte` is signed and Go's is not. Every comparison, every shift, every
    array index derived from a byte is a place the port can silently differ.
  - Padding, key length truncation and the 32-byte password pad are the usual
    places this goes wrong.

- [ ] D8. Check what happens on the wrong password
  - Java's behaviour on a failed decrypt is specific. Does the Go do the same,
    or does it return a different error, or worse, garbage?

- [ ] D9. Do not accept a passing test as proof
  - A round trip that encrypts and decrypts with the same port passes even if
    both halves are wrong. Check against the checked-in encrypted fixtures,
    which Java produced.

---

# Phase E — User feedback

See [`TEMPLATE.md`](TEMPLATE.md) for E1–E4.

---

# Blocked

- [ ] `TestSymmetricKeyEncryption` writes encrypted PDFs as well as reading
      them. The writer lands in slice 7. Decide whether this branch ports only
      the reading half, or waits.
