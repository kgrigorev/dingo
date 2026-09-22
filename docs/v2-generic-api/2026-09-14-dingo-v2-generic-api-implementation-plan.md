# dingo v2 Generic API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship `flamingo.me/dingo/v2`, a type-safe generic typed API (`injector.Bind[T]().To[U]()`) over the existing v0 reflection engine, together with the root changes it needs and the `compat` adapters that lets v0 and v2 modules run on one injector.

**Architecture:** Two pull requests against `github.com/i-love-flamingo/dingo`, the repository behind the `flamingo.me/dingo` vanity path. Neither is gated on a release (see "How the v2 module resolves the engine"). The root PR — released as `v0.5.0` — adds two private packages (`internal/hooks`, `internal/typename`), an attached slot on the engine's `Injector` filled eagerly through hooks.Attach, adapter-aware module identity, one exported sentinel, one wrapped error cause and one pointer guard. The v2 PR adds the `v2/` module: a thin `Injector`/`Binding[T]` typed API whose every method validates at bind time and then forwards to the engine through `reflect.New` type carriers, plus `compat`, the black-box suite driven by the 182-ID catalogue, the README, the example and the CI changes.

**Tech Stack:** Go 1.27 generic methods (v2 module), Go 1.25.8 (root module), `reflect`, testify, gonum (engine, unchanged), golangci-lint v2.13.2, GitHub Actions, semanticore.

**Spec:** `docs/v2-generic-api/2026-09-08-dingo-v2-generic-api-design.md` and its appendix `docs/v2-generic-api/2026-09-08-dingo-v2-generic-api-test-catalogue.md`. The plan argues from the spec; executors read both. Where the plan and the spec disagree, the spec wins and the plan gets fixed — except at the three points listed under "Verified deviations from the spec", where the spec is factually wrong about the current engine and this plan records the evidence.

## Global Constraints

Copied from the spec. Every task's requirements implicitly include this section.

- Root module `flamingo.me/dingo` keeps `go 1.25.8`. Do not raise it. The `errors.AsType` TODO in `module.go:131` stays.
- v2 module path is `flamingo.me/dingo/v2`, directory `v2/`, `go 1.27`, `require flamingo.me/dingo v0.4.1`, **no `replace` directive**. The engine is resolved by the committed `go.work`, not by that version; see "How the v2 module resolves the engine".
- `go.work` at the repository root: `go 1.27`, `use (. ./v2)`, committed; `go.work.sum` committed when non-empty.
- Runtime behavior identical to v0.4.1 except the three changes under "Verified deviations".
- Every existing v0 call takes today's code path. Root `go test ./...` passes before and after the root PR with **no test edited** except the `TestDingoCircula` → `TestDingoCircular` rename; new tests are added, never changed.
- Every v2 bind-time check panics with an `error` wrapping `ErrInvalidBinding`, message shape `dingo: <call>: <reason>`, types printed by `typename.Qualified` (full import path). The prefix, the call and both type names are required; exact wording may differ.
- v2 public API: `Bind[T]`, `BindMulti[T]`, `BindMap[T](key)`, `Override[T](annotatedWith)`, `BindInterceptor[T, I]`, `GetInstance[T]`, `GetAnnotatedInstance[T](a)`; `Binding[T]` with `To[U]`, `ToType(reflect.Type)`, `ToInstance(T)`, `ToProvider(any)`, `AnnotatedWith`, `In`, `AsEagerSingleton`; plus the names carried over from v0.4.1 listed in the spec's "Package flamingo.me/dingo/v2". The zero `Injector` panics on every method except `Child`, which returns an error.
- `compat` API: `Injector(*v0.Injector) *dingo.Injector`, `Root(*dingo.Injector) *v0.Injector`, `FromRoot(v0.Module) dingo.Module`, `ToRoot(dingo.Module) v0.Module`. Both adapters implement `Depender` unconditionally and return `nil` (not an empty slice) when the inner module is no `Depender`.
- Typed-API attachment is eager, inside the root's `NewInjector`, and **idempotent**. See "Verified deviations", item 3.
- Hooks unwrap method is `DingoUnwrap() any`; `Unwrap` stops on nil, on a fixed point and at `MaxUnwrapDepth = 8`.
- Tests: black box only (`package dingo_test`, `package compat_test`), every test and subtest `t.Parallel()` except C-04 and the tracing test, `t.Helper()` first in helpers, names `Test<Surface>_<Rule>`, doc comment with `// Covers <IDs>.` and `// Catches: ...`. No `dingo.NewInjector` outside C-04, S-10 and the tests asserting what `NewInjector` binds. Bare `assert.Error` and bare `assert.Panics` are banned. Counters are `atomic.Int64`; never `time.Sleep`.
- Lint: the whole v2 tree must be clean under the repository's `.golangci.yml` with golangci-lint **v2.13.2** (a binary built with Go older than 1.27 refuses generic-method files). `//nolint` needs a specific linter and an explanation. Tests are excluded from `varnamelen`, `err113`, `forcetypeassert`, `goconst`, `wrapcheck`, `containedctx`; production code is not. Production code needs named constants for every numeric literal (`mnd` checks arguments, conditions, returns and assignments).
- CI after the v2 PR: `tests` (`go test -shuffle=on -race ./... ./v2/...`, Go `1.27` and `1.*`), `tests-v0` (`GOWORK=off`, Go `1.25` and `1.*`), `tests-v2-published` (`GOWORK=off`, dir `v2`, `continue-on-error: true` until `v0.5.0` is on the proxy, then required), `coverage` (one profile per module, v2 gated only once `v2/coverage.min` exists), `static-checks` (vet over both modules, gofmt, goimports, go generate, miniexample smoke test), golangci-lint matrix over `.` and `v2`, semanticore skipped when the commit message contains `[skip release]`.
- Conventional commits. Root PR commits are `feat:`/`fix:`/`test:`/`refactor:`; the v2 PR's squash commit carries `[skip release]` and a `BREAKING CHANGE:` footer. `Changelog.md` is never hand-edited.
- Before editing any `.go` file, invoke the `use-modern-go` skill for that module's Go version (`--go-version 1.25` for root files, `--go-version 1.27` for `v2/`). Guidance already folded into the code below: `any` not `interface{}`, `reflect.TypeFor[T]()`, `for range n`, `atomic.Int64`, `t.Context()`, `errors.Is`, generic methods where the operation belongs to the type.

---

## Verified deviations from the spec

Each was reproduced against the engine at `master` tip as of 2026-09-14 (the SHA recorded in
early drafts was not a commit on this repository's history) with a throwaway module using
`replace flamingo.me/dingo => .`. The spec's **decisions** stand in all three cases; only its
**rationale** is corrected, and the PR body must say so rather than repeat a claim that is false.

1. **`GetInstance[**T]` panics in v0, it does not silently resolve `*T`.** The spec (review decision 20) says the `reflect.Type` path "would strip one level and resolve `*T`". What actually happens: `getInstanceOfTypeWithAnnotation` strips one level to `*Service`, `createInstanceOfAnnotatedType` does `reflect.New(*Service)` and `requestInjection` walks `**Service` → `*Service` (nil) → `Elem()` of a nil pointer → a zero `reflect.Value` → `panic: reflect: call of reflect.Value.Type on zero Value` at `dingo.go:724`. G-05 therefore turns a **panic** into a clean error, which is a stronger reason for the check, and its `Catches:` line must say so.

2. **The pointer guard fixes a bug reachable through the public v0 API today.** The spec (review decision 12) says the guarded path "is only reachable once modules are wrapped". It is not: `dingo.TryModule(ValueModule{})`, where `ValueModule` is a value-typed module with an unresolvable `inject` field, already returns `dingo.TryModule panic: "reflect: Elem of invalid type main.ValueModule"` because `dingo.go:125` calls `.Elem()` unconditionally. Task A6 therefore pins the unwrapped case as well as the wrapped one, and the root PR body describes the guard as a small bug fix, not as behavior-neutral.

3. **A second typed injector attached in `Child` would break every Flamingo child area.** `Child()` already calls `NewInjector()` internally (`dingo.go:95`), so a child engine receives its typed injector there. Attaching again in `Child` would append a second, *unequal* binding for the v2 `Injector` key; a later `child.InitModules(...)` — exactly what `framework/config/area.go` does per area — then fails with `already known binding for "…Injector"` (reproduced: two unequal bindings for one key on a child do fail `InitModules`). Attachment is therefore written once, is idempotent, and Task A5 asserts `Child` attaches exactly one typed injector.

Two further engine facts the plan relies on, verified the same way and *not* in conflict with the spec: a provider bound as `ToProvider(func() Service {…})` makes the engine return a `Service` **value** even for `GetInstance(reflect.TypeFor[*Service]())`, so the address-of-a-copy half of `adapt[T]` is load-bearing rather than theoretical; and `reflect.Value.Comparable()` reports `false` for a struct holding a func while `reflect.Type.Comparable()` reports `true`, so `hooks.Unwrap`'s fixed-point check (`same`) must use the **Value** form or it panics on a wrapped `ModuleFunc`.

---

## How to read this plan

**Two parts, sequential in content but not gated on a release.** Part A is the root PR (branch `feat/v2-naming-hooks-root`). Part B is the v2 PR (branch `feat/v2-generic-api`). Part B builds on Part A's `internal/hooks` and `internal/typename`, so Part A's code must exist on the branch Part B starts from — but Part B does **not** wait for a tagged release. See the next section.

## How the v2 module resolves the engine

**This work is destined for `github.com/i-love-flamingo/dingo`.** That is the repository behind the `flamingo.me/dingo` vanity import path, and it is where Part A lands and is released as `v0.5.0`. Any fork this plan is executed in is a staging area: `flamingo.me/dingo` resolves through `flamingo.me`, never through a fork's Git remote, so a release cut in a fork does not appear at that path and cannot satisfy a `require` line. Neither can a hash-based pseudo-version — it would have to name a commit in the upstream repository.

The end state is therefore exactly what the spec describes: `v2/go.mod` requires `flamingo.me/dingo v0.5.0`, with no `replace`, and the v2 module builds as any consumer would build it.

**The interim, which is where Part B is actually written.** Between Part A being written and `v0.5.0` being published upstream, the engine is resolved by the committed `go.work`: in workspace mode Go resolves `flamingo.me/dingo` to the local `.` module and never fetches, so the `require` line's version is inert. That is not a workaround — it is what Task B22's `tests` job wants regardless, because it builds the typed injector against the engine *at the same commit*.

During the interim:

- `v2/go.mod` requires `flamingo.me/dingo v0.4.1` — the newest version that exists at the vanity path, chosen only so the file resolves outside the workspace. Still **no `replace` directive**.
- `tests-v2-published`, the one job that leaves the workspace, cannot pass, because the published engine has no `internal/hooks`. It stays `continue-on-error` and serves as the signal for when the interim ends.
- **Part B is not gated on any of this.** It starts as soon as Part A's code is on the branch it builds from.

**Ending the interim** is Task B24 step 1a: once Part A is merged upstream and `v0.5.0` is on the proxy, bump the `require` line to `v0.5.0` and drop `continue-on-error` in the same commit. Nothing else changes.

**Landing the root changes upstream.** They touch only Go source; a fork's `CLAUDE.md`, `.claude/` tooling and `skills-lock.json` are fork-local and do not travel. Of the documents, the design doc and the test catalogue travel — they are the reasoning and the behaviour contract a reviewer needs — and **this plan does not**: it is an execution script for whoever does the work, not a design artifact. Rebase the code commits onto `upstream/master`, put the two documents in one commit ahead of them, and drop any content-neutral merge commits a fork's branch history accumulated.

**What is given verbatim and what is not.** Every **production** file in this plan is given as complete, final code: that is where correctness is hard and where the engine's behavior had to be verified. **Test** files are given as complete code for the structurally novel ones (helpers, fixtures, tables, drivers, gates) and, for the large tables, as their full row set — a table row *is* the test. Where a task says "one row per catalogue line", the catalogue's Case column is the assertion to write and `catalogue_test.go` (Task B20) fails the build if a row is missing, so completeness is machine-checked rather than left to memory. No step says "add tests" without saying which behavior, which fixture and which assertion.

**Working directory.** All paths are relative to the repository root (`/home/user/dingo` in the authoring session). The shell resets its working directory between commands; use absolute paths or `git -C <repo>` where that bites.

**Commands.**

| What | Root (Part A) | Workspace (Part B) |
|---|---|---|
| tests | `CGO_ENABLED=1 go test -race ./...` | `CGO_ENABLED=1 go test -shuffle=on -race ./... ./v2/...` |
| one test | `go test -run 'TestName' -v .` | `go test -run 'TestName' -v ./v2/` (or `./v2/compat/`) |
| vet | `go vet ./...` | `go vet ./... ./v2/...` |
| format | `gofmt -l .` prints nothing; `go run golang.org/x/tools/cmd/goimports@latest -w .` produces no diff | same |
| lint | `golangci-lint run ./...` | `cd v2 && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./...` |
| toolchain | local `go` is 1.25.8 | `GOTOOLCHAIN=auto` (the default) picks 1.27.x once `go.work` says `go 1.27`; 1.27.1 is in this environment's module cache |

**Each task ends in a commit.** Commit messages in the steps are complete; append the attribution lines the session requires. Never push to `master`; each part is a draft PR.

---

## File structure

### Part A: root PR

| Path | Responsibility | Change |
|---|---|---|
| `internal/typename/typename.go` | `Qualified(reflect.Type) string`, the full-import-path type printer shared by root injector and typed API | create |
| `internal/typename/typename_test.go` | the printer's table, black box against `typename.Qualified` | create |
| `internal/hooks/hooks.go` | `Unwrapper`, `MaxUnwrapDepth`, `Unwrap`, the four hook variables | create |
| `internal/hooks/hooks_test.go` | `TestUnwrap_StopsOnNilFixedPointAndDepth` | create |
| `dingo.go` | `ErrPointerToInterface`; the `attached` slot; `attach`; `InitModules` injecting `hooks.Unwrap(module)` with the pointer-guarded name; the `// coverage:` comment on the add-failure branch; `init` installing `hooks.Attached` | modify |
| `module.go` | `moduleKeyOf` through `Unwrap`; `qualifiedTypeName` delegating to `typename.Qualified`; the wrapped `ErrModuleSort` cause with its `// coverage:` comment | modify |
| `dingo_test.go` | four added tests (sentinel, typed-API attachment, pointer guard, unwrapped-module injection) | modify (add only) |
| `module_test.go` | `TestModuleKeyOf_KeysWrappedModulesByTheUnwrappedModule` | modify (add only) |
| `tracing_test.go` | `TestInjectionTracing_LogsFieldSetsAndResolutionsWhenEnabled` (package `dingo`) | create |
| `circular_test.go` | rename `TestDingoCircula` → `TestDingoCircular` | modify |

### Part B: v2 PR

| Path | Responsibility |
|---|---|
| `go.work`, `go.work.sum` | workspace over `.` and `./v2` |
| `v2/go.mod`, `v2/go.sum` | module `flamingo.me/dingo/v2`, `go 1.27`, requires root `v0.4.1` (inert under `go.work`) and testify |
| `v2/doc.go` | package documentation with the canonical example |
| `v2/errors.go` | `ErrInvalidBinding`, the re-exported root sentinels, `bindError`, `invalid` |
| `v2/typename.go` | `qualified`, `typeName[T]`, `carrier`, `keyType[T]` |
| `v2/injector.go` | `Injector`, `mustRoot`, `attachedOf`, `NewInjector`, `Child`, `InitModules`, `SetBuildEagerSingletons`, `BuildEagerSingletons`, `RequestInjection`, `BindScope`, `EnableCircularTracing`, `EnableInjectionTracing` |
| `v2/module.go` | `Module`, `ModuleFunc`, `Depender`, `TryModule`, the `asModule` adapter, `asModules`, the `init` installing the hooks.Attach, hooks.RootOf, and hooks.AsModule |
| `v2/scope.go` | `Scope`, `SingletonScope`, `ChildSingletonScope` aliases; `Singleton`, `ChildSingleton`; `NewSingletonScope`, `NewChildSingletonScope` |
| `v2/bind.go` | `callOf`, `keyOf[T]`, `collectionKeyOf[T]`, `Bind`, `BindMulti`, `BindMap`, `Override` |
| `v2/binding.go` | `Binding[T]` and its seven methods plus `must`, `setTarget`, `checkTarget`, `setScope` |
| `v2/interceptor.go` | `BindInterceptor[T, I]` |
| `v2/resolve.go` | `GetInstance[T]`, `GetAnnotatedInstance[T]`, `resolve[T]`, `adapt[T]` |
| `v2/inspect.go` | `Inspector`, `Inspect` |
| `v2/compat/doc.go`, `v2/compat/compat.go` | the four compat functions and the `rootModule` adapter |
| `v2/fixtures_test.go`, `v2/helpers_test.go` | fixtures; the six helpers and their four contract tests |
| `v2/bind_key_test.go` … `v2/inspect_test.go`, `v2/tracing_test.go` | the catalogue's test files, one subject each |
| `v2/compilefail_test.go`, `v2/testdata/compilefail/` | compile-fail driver and corpus module |
| `v2/catalogue_test.go`, `v2/testdata/catalogue.txt` | the behavior-ID gate and its data |
| `v2/readme_test.go`, `v2/README.md` | README and its drift guard |
| `v2/example_*_test.go`, `v2/compat/example_test.go` | whole-file Examples |
| `v2/testdata/moduleidentity/{commerce,om3}/cart/module.go` | same-name fixture modules on the v2 API |
| `v2/example/` | the root example ported to v2 |
| `.github/workflows/main.yml`, `.github/workflows/golangci-lint.yml`, `.github/workflows/semanticore.yml` | CI changes |

---

# Part A — the root PR (`feat/v2-naming-hooks-root`)

> **Landed on fork `v2` (do not re-execute A0–A8).** Part A is already on
> `kgrigorev/dingo` branch `v2` as of tip `20f2e0d`: root bridge work in PRs that landed as
> `cb9854f` / `4e11736` / `d31f060`, then Phase 0.5 naming freeze + docs alignment as
> [PR #11](https://github.com/kgrigorev/dingo/pull/11) (`20f2e0d`). The per-task checkboxes and
> commit mandates below remain as the historical execution script; treat them as done, not as
> work to redo. Part B starts from current `v2` HEAD (or an equivalent renamed Part A HEAD).

- [ ] **Task A0: branch**

```bash
git -C /home/user/dingo fetch origin master
git -C /home/user/dingo checkout -b feat/v2-naming-hooks-root origin/master
```

(If the session mandates a fixed development branch, use that branch name instead and keep everything else unchanged.)

### Task A1: `internal/typename` with `Qualified`

**Files:**
- Create: `internal/typename/typename.go`
- Create: `internal/typename/typename_test.go`
- Modify: `module.go:240-277` (`qualifiedTypeName` delegates)

**Interfaces:**
- Consumes: nothing.
- Produces: `func Qualified(typ reflect.Type) string` in `flamingo.me/dingo/internal/typename`. Named types print as `<import path>.<Name>`; pointer, slice, array, map and channel recurse; anything else falls back to `reflect.Type.String()`. Used by A4, A6 and every v2 message.

The root's existing `TestQualifiedTypeName` (`module_test.go:233`) is **not touched**: it keeps passing through the delegating wrapper, which makes it the delegation check the spec asks for, without breaking the "no test edited" constraint.

- [ ] **Step 1: Write the failing test**

`internal/typename/typename_test.go`:

```go
package typename_test

import (
	"reflect"
	"testing"

	"flamingo.me/dingo/internal/typename"
	"github.com/stretchr/testify/assert"
)

type named struct{}

type namedIface interface{ M() }

// TestQualified_PrintsFullImportPaths pins the printer shared by the root injector's module
// diagnostics and the typed API's bind-time messages.
// Catches: a printer that falls back to reflect.Type.String for named types, which would make two
// same-named types from different packages indistinguishable in an error message.
func TestQualified_PrintsFullImportPaths(t *testing.T) {
	t.Parallel()

	const pkg = "flamingo.me/dingo/internal/typename_test."

	tests := []struct {
		name string
		typ  reflect.Type
		want string
	}{
		{name: "named struct", typ: reflect.TypeFor[named](), want: pkg + "named"},
		{name: "named interface", typ: reflect.TypeFor[namedIface](), want: pkg + "namedIface"},
		{name: "pointer to named", typ: reflect.TypeFor[*named](), want: "*" + pkg + "named"},
		{name: "pointer to pointer", typ: reflect.TypeFor[**named](), want: "**" + pkg + "named"},
		{name: "slice of interface", typ: reflect.TypeFor[[]namedIface](), want: "[]" + pkg + "namedIface"},
		{name: "array", typ: reflect.TypeFor[[3]named](), want: "[3]" + pkg + "named"},
		{name: "map", typ: reflect.TypeFor[map[string]named](), want: "map[string]" + pkg + "named"},
		{name: "receive channel", typ: reflect.TypeFor[<-chan named](), want: "<-chan " + pkg + "named"},
		{name: "send channel", typ: reflect.TypeFor[chan<- named](), want: "chan<- " + pkg + "named"},
		{name: "bidirectional channel", typ: reflect.TypeFor[chan named](), want: "chan " + pkg + "named"},
		{name: "basic type", typ: reflect.TypeFor[string](), want: "string"},
		{name: "anonymous struct falls back", typ: reflect.TypeFor[struct{ A int }](), want: "struct { A int }"},
		{name: "func falls back", typ: reflect.TypeFor[func() int](), want: "func() int"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, typename.Qualified(tt.typ))
		})
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/typename/ -v`
Expected: FAIL — the package `flamingo.me/dingo/internal/typename` does not exist.

- [ ] **Step 3: Write the implementation**

`internal/typename/typename.go` — the body is `qualifiedTypeName` moved verbatim from `module.go:246-277`:

```go
// Package typename prints reflect types with their full import path.
//
// It is shared by the root injector (module graph diagnostics) and the typed API (bind-time
// messages), so that both name a type the same way. It is internal: Go's internal rule is
// import-path based, so flamingo.me/dingo/v2 may import it while nothing outside this repository
// can.
package typename

import (
	"fmt"
	"reflect"
)

// Qualified is like reflect.Type.String but uses the full import path instead of the short
// package name for named types. It handles common composite types recursively (pointer, slice,
// array, map, channel) so that any named element or key type inside them is also fully
// qualified. Anonymous composite types (struct, interface, func) fall back to
// reflect.Type.String, as does anything else not covered above.
func Qualified(typ reflect.Type) string {
	if typ.PkgPath() != "" {
		return typ.PkgPath() + "." + typ.Name()
	}

	//nolint:exhaustive // only kinds that can wrap a named type are qualified, everything else falls back to reflect.Type.String
	switch typ.Kind() {
	case reflect.Pointer:
		return "*" + Qualified(typ.Elem())
	case reflect.Slice:
		return "[]" + Qualified(typ.Elem())
	case reflect.Array:
		return fmt.Sprintf("[%d]%s", typ.Len(), Qualified(typ.Elem()))
	case reflect.Map:
		return "map[" + Qualified(typ.Key()) + "]" + Qualified(typ.Elem())
	case reflect.Chan:
		var prefix string

		switch typ.ChanDir() {
		case reflect.RecvDir:
			prefix = "<-chan "
		case reflect.SendDir:
			prefix = "chan<- "
		case reflect.BothDir:
			prefix = "chan "
		}

		return prefix + Qualified(typ.Elem())
	}

	return typ.String()
}
```

In `module.go`, replace the **body** of `qualifiedTypeName` and its doc comment, keeping the function so `moduleKey.name` and the untouched root test still compile:

```go
// qualifiedTypeName delegates to typename.Qualified; the printer is shared with the typed API.
func qualifiedTypeName(typ reflect.Type) string {
	return typename.Qualified(typ)
}
```

Add `"flamingo.me/dingo/internal/typename"` to `module.go`'s imports. Keep `fmt` — `Sort` and `addModule` still use it.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/typename/ -v && go test -run 'TestQualifiedTypeName' -v .`
Expected: PASS for the new table (13 subtests) and for the untouched root table.

- [ ] **Step 5: Format, vet, full suite**

Run: `gofmt -l . && go vet ./... && CGO_ENABLED=1 go test -race ./...`
Expected: no gofmt output, vet clean, all PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/typename module.go
git commit -m "refactor: move qualifiedTypeName into internal/typename for sharing with v2"
```

### Task A2: `internal/hooks` with `Unwrap` and the hooks

**Files:**
- Create: `internal/hooks/hooks.go`
- Create: `internal/hooks/hooks_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces (package `flamingo.me/dingo/internal/hooks`):
  - `type Unwrapper interface { DingoUnwrap() any }`
  - `const MaxUnwrapDepth = 8`
  - `func Unwrap(module any) any`
  - `var Attached func(root any) any` — installed by the root in A5
  - `var Attach, RootOf, AsModule func(any) any` — installed by v2 in B2, nil until then

- [ ] **Step 1: Write the failing test**

`internal/hooks/hooks_test.go`:

```go
package hooks_test

import (
	"testing"

	"flamingo.me/dingo/internal/hooks"
	"github.com/stretchr/testify/assert"
)

type leaf struct{ name string }

// wrapper returns whatever inner holds; a nil inner models a broken adapter.
type wrapper struct{ inner any }

func (w *wrapper) DingoUnwrap() any { return w.inner }

// selfWrapper returns itself: a fixed point.
type selfWrapper struct{}

func (s *selfWrapper) DingoUnwrap() any { return s }

// funcWrapper is a wrapper of uncomparable dynamic type, the shape a v2 ModuleFunc adapter takes.
type funcWrapper struct{ inner any }

func (f funcWrapper) DingoUnwrap() any { return f.inner }

// TestUnwrap_StopsOnNilFixedPointAndDepth pins the unwrap contract the root injector keys its
// module graph on.
// Catches: a nil-returning adapter collapsing every wrapped module onto one nil key so that only
// the first is configured; a self-returning adapter looping forever; two distinct wrapped modules
// becoming one; and a comparability panic when the wrapped value is a function.
func TestUnwrap_StopsOnNilFixedPointAndDepth(t *testing.T) {
	t.Parallel()

	t.Run("an unwrapped value is its own result", func(t *testing.T) {
		t.Parallel()

		module := &leaf{name: "a"}
		assert.Same(t, module, hooks.Unwrap(module))
	})

	t.Run("one layer unwraps to the inner value", func(t *testing.T) {
		t.Parallel()

		module := &leaf{name: "a"}
		assert.Same(t, module, hooks.Unwrap(&wrapper{inner: module}))
	})

	t.Run("two layers unwrap to the inner value", func(t *testing.T) {
		t.Parallel()

		module := &leaf{name: "a"}
		assert.Same(t, module, hooks.Unwrap(&wrapper{inner: &wrapper{inner: module}}))
	})

	t.Run("an adapter returning nil keeps its own identity", func(t *testing.T) {
		t.Parallel()

		broken := &wrapper{inner: nil}
		assert.Same(t, broken, hooks.Unwrap(broken))
	})

	t.Run("an adapter returning itself stops", func(t *testing.T) {
		t.Parallel()

		fixed := &selfWrapper{}
		assert.Same(t, fixed, hooks.Unwrap(fixed))
	})

	t.Run("two wrappers around distinct values stay distinct", func(t *testing.T) {
		t.Parallel()

		first, second := &leaf{name: "a"}, &leaf{name: "b"}
		assert.NotSame(t, hooks.Unwrap(&wrapper{inner: first}), hooks.Unwrap(&wrapper{inner: second}))
	})

	t.Run("depth is capped at MaxUnwrapDepth layers", func(t *testing.T) {
		t.Parallel()

		const layers = 2 * hooks.MaxUnwrapDepth

		chain := make([]*wrapper, layers)

		var inner any = &leaf{name: "deep"}

		for i := layers - 1; i >= 0; i-- {
			chain[i] = &wrapper{inner: inner}
			inner = chain[i]
		}

		assert.Same(t, chain[hooks.MaxUnwrapDepth], hooks.Unwrap(chain[0]))
	})

	t.Run("a wrapper holding a function unwraps without a comparability panic", func(t *testing.T) {
		t.Parallel()

		fn := func() {}

		assert.NotPanics(t, func() {
			assert.IsType(t, fn, hooks.Unwrap(funcWrapper{inner: fn}))
		})
	})

	t.Run("nil is nil", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, hooks.Unwrap(nil))
	})
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/hooks/ -v`
Expected: FAIL — the package does not exist.

- [ ] **Step 3: Write the implementation**

`internal/hooks/hooks.go`:

```go
// Package hooks is the private contract between the dingo root injector (flamingo.me/dingo) and
// the typed API (flamingo.me/dingo/v2). Go's internal-package rule is import-path based, so the v2
// module may import it while nothing outside this repository can.
//
// Values are typed any because neither package can import the other's types without a cycle; each
// side asserts its own types.
package hooks

import "reflect"

// Unwrapper is implemented by module adapters. The root injector keys the module graph by the
// unwrapped module. The method name carries the Dingo prefix so that a third-party module with an
// unrelated DingoUnwrap method is not unwrapped by accident.
type Unwrapper interface {
	DingoUnwrap() any
}

// MaxUnwrapDepth bounds Unwrap. Two adapter layers is the deepest legitimate nesting
// (ToRoot(FromRoot(m))); the cap exists so that a self-returning adapter cannot loop.
const MaxUnwrapDepth = 8

// Unwrap follows DingoUnwrap until a value does not implement Unwrapper, returns nil, returns
// itself, or MaxUnwrapDepth is reached. A nil return stops at the last non-nil value, so a broken
// adapter keeps its own identity instead of collapsing onto the nil key.
func Unwrap(module any) any {
	current := module

	for range MaxUnwrapDepth {
		wrapped, ok := current.(Unwrapper)
		if !ok {
			return current
		}

		inner := wrapped.DingoUnwrap()
		if inner == nil || same(inner, current) {
			return current
		}

		current = inner
	}

	return current
}

// same reports whether left and right are the same value. It uses reflect.Value.Comparable
// rather than reflect.Type.Comparable, because a struct type holding an interface field is a
// comparable *type* while a value of it holding a function is not a comparable *value*: a plain
// == on those two panics, and a ModuleFunc adapter is exactly that shape.
func same(left, right any) bool {
	value := reflect.ValueOf(left)
	if !value.IsValid() || !value.Comparable() || value.Type() != reflect.TypeOf(right) {
		return false
	}

	return left == right
}

// Attached returns the typed injector attached to a root injector, or nil when none was attached.
// Installed by package flamingo.me/dingo in an init function.
var Attached func(root any) any

// Installed by package flamingo.me/dingo/v2 in an init function. Nil until then, so a binary that
// does not link v2 attaches nothing.
var (
	// Attach creates the typed injector for a root injector and binds it into the root. The root's
	// NewInjector calls it, inside the root injector's construction, before any module runs.
	Attach func(root any) any
	// RootOf returns the root injector behind a typed injector.
	RootOf func(attached any) any
	// AsModule wraps a v2 module in the root Module adapter.
	AsModule func(module any) any
)
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/hooks/ -v`
Expected: PASS, nine subtests.

- [ ] **Step 5: Lint the new package**

Run: `golangci-lint run ./internal/...`
Expected: no findings. `varnamelen`'s `max-distance` is 10 and every name here is longer than one character, so nothing should fire.

- [ ] **Step 6: Commit**

```bash
git add internal/hooks
git commit -m "feat: add internal/hooks, the private contract between engine and typed API"
```

### Task A3: export `ErrPointerToInterface`

**Files:**
- Modify: `dingo.go:18-25` (the var block)
- Modify: `dingo_test.go` (append one test)

**Interfaces:**
- Produces: `var ErrPointerToInterface = errors.New("pointer to interface is not allowed")` in package `dingo`; `errPointersToInterface` becomes an alias of it and `dingo.go:782` keeps wrapping that name. v2 re-exports the same value (B2) and `GetInstance[*Iface]()` wraps it (B3).

- [ ] **Step 1: Write the failing test**

Append to `dingo_test.go` (package `dingo`; `someStructWithInvalidInterfacePointer`, `testInterface` and `interfaceImpl1` already exist in that file):

```go
// TestInjection_PointerToInterfaceWrapsExportedSentinel pins the exported sentinel and its
// message, which the typed API re-exports and matches with errors.Is.
// Catches: a second, unexported error value being wrapped, which would make
// errors.Is(err, dingo.ErrPointerToInterface) false for a caller of either package.
func TestInjection_PointerToInterfaceWrapsExportedSentinel(t *testing.T) {
	t.Parallel()

	injector, err := NewInjector()
	require.NoError(t, err)

	injector.Bind((*testInterface)(nil)).To(interfaceImpl1{})

	_, err = injector.GetInstance(new(someStructWithInvalidInterfacePointer))
	require.ErrorIs(t, err, ErrPointerToInterface)
	assert.ErrorContains(t, err, "pointer to interface is not allowed")
}
```

`dingo_test.go` does not import `require` yet; add `"github.com/stretchr/testify/require"` to its import block.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test -run 'TestInjection_PointerToInterfaceWrapsExportedSentinel' -v .`
Expected: compile error `undefined: ErrPointerToInterface`.

- [ ] **Step 3: Write the implementation**

In `dingo.go`, replace the var block at lines 18-25:

```go
var (
	ErrInitModules           = errors.New("initialization of modules failed")
	ErrInvalidInjectReceiver = errors.New("usage of 'Inject' method with struct receiver is not allowed")
	// ErrPointerToInterface is wrapped by the injection error for a pointer-to-interface field,
	// and by the typed API's GetInstance for a pointer-to-interface request.
	ErrPointerToInterface = errors.New("pointer to interface is not allowed")
	// errPointersToInterface keeps the old name so that a grep for it still finds the declaration.
	errPointersToInterface = ErrPointerToInterface

	traceCircular    []circularTraceEntry
	injectionTracing = false
)
```

`dingo.go:782` stays as it is; it now wraps the exported value.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `CGO_ENABLED=1 go test -race ./...`
Expected: PASS, including the pre-existing `TestInjectionOfInterfacePointer`, which asserts no message text.

- [ ] **Step 5: Commit**

```bash
git add dingo.go dingo_test.go
git commit -m "feat: export ErrPointerToInterface with the message 'pointer to interface is not allowed'"
```

### Task A4: wrap the `ErrModuleSort` cause and mark the two unreachable branches

**Files:**
- Modify: `module.go:150`
- Modify: `dingo.go:113-116`

No test: gonum's `topo.SortStabilized` reports cycles through the `Unorderable` branch above, so the fallback is unreachable through the public API (spec, review decision 8). The `// coverage:` comments are the documented excuse the catalogue names for M-08's add-failure branch and for this line.

- [ ] **Step 1: Edit `module.go`**

Replace `return nil, ErrModuleSort` at the end of `Sort`:

```go
	// coverage: unreachable through the public API; topo.SortStabilized reports cycles through the branch above
	return nil, fmt.Errorf("%w: %w", ErrModuleSort, err)
```

- [ ] **Step 2: Edit `dingo.go`**

Above the `mg.Add` error branch in `InitModules`:

```go
	err := mg.Add(modules...)
	// coverage: unreachable through the public API; Depends() []Module cannot fail, so Add never returns an error
	if err != nil {
		return fmt.Errorf("%w: failed adding modules to the graph: %w", ErrInitModules, err)
	}
```

- [ ] **Step 3: Verify**

Run: `gofmt -l . && go vet ./... && CGO_ENABLED=1 go test -race ./...`
Expected: clean and PASS. `errors.Is(err, ErrModuleSort)` is unaffected by the wrap; only the message grows.

- [ ] **Step 4: Commit**

```bash
git add module.go dingo.go
git commit -m "fix: keep the cause when sorting modules fails and mark the unreachable branches"
```

### Task A5: the attached slot, eager idempotent attachment and the `Attached` hook

**Files:**
- Modify: `dingo.go` — imports, a new `init`, the `Injector` struct, `NewInjector`, `Child`, a new `attach` method
- Modify: `dingo_test.go` (append one test)

**Interfaces:**
- Consumes: `hooks.Attach`, `hooks.Attached` (A2).
- Produces: `Injector.attached any`, an unexported slot; the root's `init` installs `hooks.Attached`; `NewInjector` fills the slot. v2's `attachedOf` (B2) and `compat.Injector` (B17) read it through `hooks.Attached`.

Read "Verified deviations", item 3, before writing this task: attachment is **written once and idempotent**, and the test proves `Child` produces exactly one typed injector for the child engine.

- [ ] **Step 1: Write the failing test**

Append to `dingo_test.go` (package `dingo`). It installs a process-wide hook, so it is a non-parallel island; Go runs non-parallel top-level tests to completion before any paused parallel sibling resumes, so the hook is never visible to another test.

```go
// TestNewInjector_AttachesEagerlyAndExactlyOnce pins eager attachment: the hook runs
// inside construction, the slot is readable through hooks.Attached, the typed injector is bound
// into the root injector, and a child gets exactly one attached injector of its own.
// Catches: a lazily created typed injector writing the unsynchronized binding map after
// InitModules while resolutions read it; and a second attachment in Child, which would append an
// unequal duplicate binding for the attached key and make the child's next InitModules fail.
//
//nolint:paralleltest // installs a process-wide hooks.Attach for its duration
func TestNewInjector_AttachesEagerlyAndExactlyOnce(t *testing.T) {
	type fakeAttached struct{ root *Injector }

	var created []*Injector

	previous := hooks.Attach
	hooks.Attach = func(root any) any {
		e, ok := root.(*Injector)
		require.True(t, ok)

		created = append(created, e)
		attached := &fakeAttached{root: e}
		e.Bind(fakeAttached{}).ToInstance(attached)

		return attached
	}

	t.Cleanup(func() { hooks.Attach = previous })

	injector, err := NewInjector()
	require.NoError(t, err)
	require.Len(t, created, 1)
	assert.Same(t, injector, created[0])

	attached, ok := hooks.Attached(injector).(*fakeAttached)
	require.True(t, ok)
	assert.Same(t, injector, attached.root)

	bound, err := injector.GetInstance(new(fakeAttached))
	require.NoError(t, err)
	assert.Same(t, attached, bound)

	child, err := injector.Child()
	require.NoError(t, err)
	require.Len(t, created, 2, "Child attaches exactly once, through the NewInjector it calls")

	childAttached, ok := hooks.Attached(child).(*fakeAttached)
	require.True(t, ok)
	assert.Same(t, child, childAttached.root)
	assert.NotSame(t, attached, childAttached)

	// the child holds exactly one binding for the attached key, so a later InitModules on it (what
	// Flamingo does per config area) does not hit the duplicate-binding check
	bindings := 0

	child.Inspect(Inspector{InspectBinding: func(of reflect.Type, _ string, _ reflect.Type, _, _ *reflect.Value, _ Scope) {
		if of == reflect.TypeOf(fakeAttached{}) {
			bindings++
		}
	}})
	assert.Equal(t, 1, bindings)
	require.NoError(t, child.InitModules())

	assert.Nil(t, hooks.Attached(nil))
	assert.Nil(t, hooks.Attached((*Injector)(nil)))
	assert.Nil(t, hooks.Attached("not a root injector"))
}
```

Add `"flamingo.me/dingo/internal/hooks"` and `"reflect"` to `dingo_test.go`'s imports if they are not there yet.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test -run 'TestNewInjector_AttachesEagerly' -v .`
Expected: FAIL — `hooks.Attached` is a nil func value, so the first `hooks.Attached(injector)` panics.

- [ ] **Step 3: Write the implementation**

In `dingo.go`, extend the imports:

```go
import (
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"flamingo.me/dingo/internal/hooks"
	"flamingo.me/dingo/internal/typename"
)
```

(`typename` is first used in A6. Adding both imports now keeps A6 to one concern; if the build complains about an unused import before A6, add `typename` there instead.)

Add the hook installation next to the var block:

```go
func init() {
	hooks.Attached = func(root any) any {
		injector, ok := root.(*Injector)
		if !ok || injector == nil {
			return nil
		}

		return injector.attached
	}
}
```

Add the slot to the struct:

```go
	// Injector defines bindings and multibindings
	// it is possible to have a parent-injector, which can be asked if no resolution is available
	Injector struct {
		bindings             map[reflect.Type][]*Binding          // list of available bindings for a concrete type
		multibindings        map[reflect.Type][]*Binding          // list of multi-bindings for a concrete type
		mapbindings          map[reflect.Type]map[string]*Binding // list of map-bindings for a concrete type
		interceptor          map[reflect.Type][]reflect.Type      // list of interceptors for a type
		overrides            []*override                          // list of overrides for a binding
		parent               *Injector                            // parent injector reference
		scopes               map[reflect.Type]Scope               // scope-bindings
		stage                uint                                 // current stage
		delayed              []interface{}                        // delayed bindings
		buildEagerSingletons bool                                 // whether to build singletons
		attached             any                                  // the typed injector, attached by hooks.Attach; nil when v2 is not linked
	}
```

In `NewInjector`, between the scope bindings and the `InitModules` call:

```go
	// bind default scopes
	injector.BindScope(Singleton)
	injector.BindScope(ChildSingleton)

	// attach the typed injector when the v2 package is linked into the binary
	injector.attach()

	// init current modules
	return injector, injector.InitModules(modules...)
```

`Child` is **not** changed: it calls `NewInjector` (`dingo.go:95`), which attaches the child's typed injector. Add the method next to `Child`:

```go
// attach fills the attached slot through the hooks package when the v2 package is linked. It
// runs inside NewInjector, before any module, so the binding the hook adds never races a
// resolution; a lazily created typed injector would write the unsynchronized binding map from
// compat.Injector or Inspect while resolutions read it.
//
// It is idempotent by design: Child() builds its root injector with NewInjector, so a child is
// attached there and exactly once. A second attachment would bind a second, unequal typed
// injector for the same key and break the child's next InitModules.
func (injector *Injector) attach() {
	if injector.attached != nil || hooks.Attach == nil {
		return
	}

	injector.attached = hooks.Attach(injector)
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `CGO_ENABLED=1 go test -race ./...`
Expected: PASS. `require.Len(t, created, 2)` proves `Child` attaches once, and the `bindings == 1` assertion plus `child.InitModules()` proves the child stays initializable.

- [ ] **Step 5: Commit**

```bash
git add dingo.go dingo_test.go
git commit -m "feat: attach a v2 attached slot eagerly in NewInjector through hooks.Attach"
```

### Task A6: module identity and injection through adapters, with the pointer guard

**Files:**
- Modify: `module.go:215-224` (`moduleKeyOf`)
- Modify: `dingo.go:123-129` (the `InitModules` loop), add `moduleTypeName`
- Modify: `dingo_test.go` (append three tests and two fixtures)
- Modify: `module_test.go` (append one test)

**Interfaces:**
- Consumes: `hooks.Unwrap` (A2), `typename.Qualified` (A1).
- Produces: `InitModules` injects `hooks.Unwrap(module)` and names the unwrapped type without calling `Elem()` on a non-pointer; `moduleKeyOf` keys by the unwrapped module, by value for the root's `ModuleFunc` and for any non-root function value. v2's `asModule` (B2) and compat's `rootModule` (B17) rely on both.

Read "Verified deviations", item 2: the pointer guard is a bug fix on a path v0 callers can already reach, so it gets an unwrapped test case as well as a wrapped one.

- [ ] **Step 1: Write the failing tests**

Append to `dingo_test.go` (package `dingo`):

```go
// wrappedModule is a minimal module adapter: it implements hooks.Unwrapper structurally and
// forwards nothing else, so the root injector's Unwrap path is what these tests exercise.
type wrappedModule struct{ inner any }

func (w *wrappedModule) Configure(*Injector) {}

func (w *wrappedModule) DingoUnwrap() any { return w.inner }

// valueModuleWithUnresolvableField is a value-typed module whose inject field cannot be resolved,
// so injection fails before any field is set. A value-typed module reaching the error branch is
// what makes the unguarded reflect.TypeOf(module).Elem() panic.
type valueModuleWithUnresolvableField struct {
	Missing testInterface `inject:"nobody-binds-this"`
}

func (valueModuleWithUnresolvableField) Configure(*Injector) {}

// TestInitModules_NamesAValueTypedModuleWithoutPanicking pins the pointer guard on the module type
// named in InitModules' injection error (review decision 12).
// Catches: reflect.TypeOf(module).Elem() on a value-typed module panicking with "reflect: Elem of
// invalid type", which today turns a reportable injection failure into a panic and which
// hooks.Unwrap would otherwise make reachable for every adapted value module.
func TestInitModules_NamesAValueTypedModuleWithoutPanicking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		module Module
	}{
		{name: "unwrapped, reachable through TryModule today", module: valueModuleWithUnresolvableField{}},
		{name: "wrapped in an adapter", module: &wrappedModule{inner: valueModuleWithUnresolvableField{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			injector, err := NewInjector()
			require.NoError(t, err)

			var initErr error

			require.NotPanics(t, func() { initErr = injector.InitModules(tt.module) })
			require.Error(t, initErr)
			assert.ErrorContains(t, initErr,
				`injection into "flamingo.me/dingo.valueModuleWithUnresolvableField" failed`)
		})
	}
}

// TestInitModules_InjectsTheUnwrappedModule pins that a wrapped module's fields are set before
// Configure, on the inner value rather than on the adapter.
// Catches: an adapter being injected instead of the module it wraps, which leaves every adapted
// module's dependencies nil inside Configure.
func TestInitModules_InjectsTheUnwrappedModule(t *testing.T) {
	t.Parallel()

	injector, err := NewInjector()
	require.NoError(t, err)

	injector.Bind((*testInterface)(nil)).To(interfaceImpl1{})

	inner := &struct {
		Dependency testInterface `inject:""`
	}{}

	require.NoError(t, injector.InitModules(&wrappedModule{inner: inner}))
	assert.NotNil(t, inner.Dependency)
}
```

Append to `module_test.go` (package `dingo`; `tryModuleOk` and `tryModuleFail` already exist there):

```go
// TestModuleKeyOf_KeysWrappedModulesByTheUnwrappedModule pins module identity through adapters.
// Catches: a wrapped module keyed by the adapter's type, which collapses every adapted module
// into one graph node so only the first is configured; and a wrapped ModuleFunc keyed by type
// alone, which would merge two distinct closures.
func TestModuleKeyOf_KeysWrappedModulesByTheUnwrappedModule(t *testing.T) {
	t.Parallel()

	inner := new(tryModuleOk)
	assert.Equal(t, moduleKeyOf(inner), moduleKeyOf(&wrappedModule{inner: inner}))
	assert.Equal(t, moduleKeyOf(inner), moduleKeyOf(&wrappedModule{inner: &wrappedModule{inner: inner}}))
	assert.NotEqual(t, moduleKeyOf(inner), moduleKeyOf(&wrappedModule{inner: new(tryModuleFail)}))

	first, second := ModuleFunc(func(*Injector) {}), ModuleFunc(func(*Injector) {})
	assert.Equal(t, moduleKeyOf(first), moduleKeyOf(&wrappedModule{inner: first}))
	assert.NotEqual(t, moduleKeyOf(&wrappedModule{inner: first}), moduleKeyOf(&wrappedModule{inner: second}))

	// a function value that is not a root Module — the shape of a v2 ModuleFunc — is keyed by value
	foreignFirst, foreignSecond := func(string) {}, func(string) {}
	assert.NotEqual(t,
		moduleKeyOf(&wrappedModule{inner: foreignFirst}),
		moduleKeyOf(&wrappedModule{inner: foreignSecond}))
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test -run 'TestInitModules_NamesAValueTyped|TestInitModules_InjectsTheUnwrapped|TestModuleKeyOf_KeysWrappedModulesByTheUnwrapped' -v .`
Expected: the guard test FAILS on both subtests (`NotPanics` reports `reflect: Elem of invalid type flamingo.me/dingo.valueModuleWithUnresolvableField`); the unwrapped-injection test FAILS because `Dependency` is nil (the adapter was injected); the key test FAILS on the first `assert.Equal`.

- [ ] **Step 3: Write the implementation**

`module.go`, replace `moduleKeyOf`:

```go
// moduleKeyOf returns a comparable key that uniquely identifies a module. Adapters implementing
// hooks.Unwrapper are looked through, so a module and its adapted forms are one module.
// Ordinary modules are keyed by reflect.Type. ModuleFunc values also include the wrapped func
// value so that distinct funcs — including distinct closures created from the same func literal —
// remain distinct modules. A wrapped function value that is not a root Module (another package's
// ModuleFunc) is keyed by value the same way. An unwrapped module takes exactly the path it took
// before adapters existed.
func moduleKeyOf(module Module) moduleKey {
	inner := hooks.Unwrap(module)
	modType := reflect.TypeOf(inner)
	key := moduleKey{typ: modType}

	if modType == typeOfModuleFunc {
		key.function = reflect.ValueOf(inner)

		return key
	}

	if _, isRootModule := inner.(Module); !isRootModule && modType != nil && modType.Kind() == reflect.Func {
		key.function = reflect.ValueOf(inner)
	}

	return key
}
```

Add `"flamingo.me/dingo/internal/hooks"` to `module.go`'s imports.

`dingo.go`, the `InitModules` loop:

```go
	for _, module := range modules {
		if err := injector.requestInjection(hooks.Unwrap(module), traceCircular); err != nil {
			return fmt.Errorf("initmodules: injection into %q failed: %w", moduleTypeName(module), err)
		}
		module.Configure(injector)
	}
```

and, next to `InitModules`:

```go
// moduleTypeName names the unwrapped module's type for InitModules' injection error. It strips one
// pointer level only when there is one, so a value-typed module (MyModule{}, a ModuleFunc, or any
// value module behind an adapter) does not panic on reflect.Type.Elem. For the pointer modules
// every existing caller passes, the result is the same "<import path>.<Name>" as before.
func moduleTypeName(module Module) string {
	typ := reflect.TypeOf(hooks.Unwrap(module))
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	return typename.Qualified(typ)
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `CGO_ENABLED=1 go test -race ./...`
Expected: PASS.

Note for the reviewer, to carry into the PR body: an unwrapped **anonymous** pointer module (`&struct{ *cart.Module }{}`) used to be named `"."` in this error and is now named `struct { *cart.Module }`. Both are failure-path text and no test asserted the old string.

- [ ] **Step 5: Commit**

```bash
git add module.go dingo.go dingo_test.go module_test.go
git commit -m "feat: key and inject modules by their unwrapped module, guard the type name in InitModules"
```

### Task A7: injection tracing test and the circular test rename

**Files:**
- Create: `tracing_test.go` (package `dingo`)
- Modify: `circular_test.go:32` (rename)

**Interfaces:**
- Produces: the root half of R-27, which `v2/tracing_test.go` (B16) pairs with.

- [ ] **Step 1: Write the tracing test**

`tracing_test.go`:

```go
package dingo

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tracedDependency struct{}

type tracedTarget struct {
	Dependency *tracedDependency `inject:""`
}

// TestInjectionTracing_LogsFieldSetsAndResolutionsWhenEnabled pins R-27: the switch makes the
// engine emit one slog line per field set ("SETTING FIELD") and per resolution ("INJECTING").
// Catches: the switch being read but never acted on, which would make the only debugging aid for
// a mis-wired graph silently useless.
//
//nolint:paralleltest // swaps the process-wide slog default and the package-level tracing switch
func TestInjectionTracing_LogsFieldSetsAndResolutionsWhenEnabled(t *testing.T) {
	var buffer bytes.Buffer

	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buffer, nil)))

	t.Cleanup(func() {
		slog.SetDefault(previous)

		injectionTracing = false
	})

	EnableInjectionTracing()

	injector, err := NewInjector()
	require.NoError(t, err)

	_, err = injector.GetInstance(new(tracedTarget))
	require.NoError(t, err)

	assert.Contains(t, buffer.String(), "SETTING FIELD: Dependency")
	assert.Contains(t, buffer.String(), "INJECTING: flamingo.me/dingo#tracedTarget")
	assert.Contains(t, buffer.String(), "INJECTING: flamingo.me/dingo#tracedDependency")
}
```

- [ ] **Step 2: Run it and confirm it passes**

Run: `go test -run 'TestInjectionTracing' -v .`
Expected: PASS — this pins existing behavior, there is nothing to implement. If the `INJECTING` text does not match, read `dingo.go:411-417`: the format is `INJECTING: <pkgpath>#<Name> "<annotation>"` and the assertion above deliberately stops before the quoted annotation.

- [ ] **Step 3: Rename the circular test**

In `circular_test.go:32`, change `func TestDingoCircula(t *testing.T) {` to `func TestDingoCircular(t *testing.T) {`. This is the only edit to an existing test in Part A.

- [ ] **Step 4: Full verification**

Run: `gofmt -l . && go vet ./... && CGO_ENABLED=1 go test -race ./... && golangci-lint run ./...`
Expected: clean; lint shows at most the pre-existing `unconvert` finding at `module_test.go:206`, which is not yours to fix.

- [ ] **Step 5: Commit**

```bash
git add tracing_test.go circular_test.go
git commit -m "test: pin injection tracing output and fix the circular test's name"
```

### Task A8: root PR wrap-up

**Files:** none new.

- [ ] **Step 1: Behavior check**

Run: `CGO_ENABLED=1 go test -shuffle=on -race -count=3 ./...`
Expected: PASS three times.

Diff the exported API against `master`:

```bash
go doc -all . > /tmp/api-after.txt
git worktree add -q /tmp/api-base origin/master
(cd /tmp/api-base && go doc -all . > /tmp/api-before.txt)
git worktree remove --force /tmp/api-base
diff /tmp/api-before.txt /tmp/api-after.txt
```

Do not use `git stash` here. By this point every task is committed, so the stash is a no-op and the comparison is `master` against itself — it passes no matter what the branch changed.

Expected: exactly one addition, `ErrPointerToInterface`. Anything else is a mistake to fix before pushing.

- [ ] **Step 2: Push and open the draft PR**

```bash
git push -u origin feat/v2-naming-hooks-root
```

Open a **draft** PR against `master`, no reviewers assigned. Title: `feat: hooks and root changes for the typed API`.

Body sections:
- *What the hooks package is for* — v2 is a typed injector over this engine; `internal/hooks` is the only coupling; Go's import-path-based internal rule lets the `v2/` module import it while nothing outside the repository can.
- *What existing callers see* — every existing call takes the same code path; the API diff shows one exported addition. Name the three behavior changes honestly: the pointer-to-interface message now reads `pointer to interface is not allowed`; the module-sort error now carries its cause; and `InitModules`' injection error no longer panics for a value-typed module (link `TestInitModules_NamesAValueTypedModuleWithoutPanicking` and say that `TryModule(ValueModule{})` panics on `master` today). Do **not** claim the guard is unreachable — see "Verified deviations", item 2.
- *Eager, idempotent typed-API attachment* — why lazy attachment would race the unsynchronized binding map, and why a second attachment in `Child` would break a child's `InitModules`.

Do not add reviewers and do not merge from this session; semanticore opens `Release v0.5.0` when the PR merges. Only the release cut in `i-love-flamingo/dingo` reaches the `flamingo.me/dingo` vanity path; one cut in a staging fork does not. Either way Part B does not wait for it — see step 3.

- [ ] **Step 3: Hand-off to Part B**

There is **no release gate**. Part B resolves the engine through the committed `go.work`, not through the proxy — see "How the v2 module resolves the engine". Part B may start as soon as Part A's code is on the branch it builds from: either Part A is merged to `master` and Part B branches from there, or Part B branches from Part A's head while that PR is in review.

Confirm the hand-off with a workspace build rather than a proxy lookup — this is what Task B1 step 3 will run:

```bash
cd /home/user/dingo && go build ./... && go test ./internal/...
```

Expected: clean, with `internal/hooks` and `internal/typename` present. That is everything Part B consumes from Part A.

---

# Part B — the v2 PR (`feat/v2-generic-api`)

Part B starts once Part A's code is on the branch it builds from (Task A8 step 3). Every path below is relative to the repository root; the v2 package's Go files live in `v2/` and declare `package dingo`, importing the engine as `v0 "flamingo.me/dingo"`.

**Naming conventions used by every Part B task, so the tasks stay consistent:**

| Concept | Spelling |
|---|---|
| the engine | `v0.Injector`, imported as `v0 "flamingo.me/dingo"` |
| the typed injector type | `Injector` (package `dingo`, directory `v2/`) |
| a bind-time failure | `panic(invalid(call, format, args...))`, message `dingo: <call>: <reason>` |
| the entry-point call string | `Bind[K]`, `BindMulti[K]`, `BindMap[K]("key")`, `Override[K]("ann")`, stored on `Binding[T].call` |
| a method rejection's call string | `<Binding.call> + "." + <method>`, e.g. `Bind[pkg.Iface].To[pkg.Impl]` |
| the engine type carrier | `carrier(typ)` = `reflect.New(typ).Interface()`, which the engine strips back to `typ` |
| the key of `T` | `keyType[T]()` = `T` with one pointer level removed |

- [ ] **Task B0: branch**

```bash
git -C /home/user/dingo fetch origin master
git -C /home/user/dingo checkout -b feat/v2-generic-api origin/master
```

### Task B1: the workspace, the v2 module and the toolchain gate

**Files:**
- Create: `go.work`
- Create: `v2/go.mod`, `v2/doc.go`
- Modify: `.gitignore` (only if it currently ignores `go.work*`)

**Interfaces:**
- Produces: a buildable `flamingo.me/dingo/v2` module in workspace mode. Every later Part B task builds on it.

- [ ] **Step 1: Confirm the golangci-lint pin parses generic methods**

This is the spec's gating step (Delivery item 2) and it must run before any generic-method file is written, because a golangci-lint binary built with a Go older than 1.27 refuses such a file outright instead of reporting findings.

```bash
mkdir -p /tmp/lintgate && cd /tmp/lintgate
cat > go.mod <<'MOD'
module lintgate

go 1.27
MOD
cat > gate.go <<'GO'
package lintgate

import "reflect"

type Holder struct{ n int }

func (h *Holder) Key[T any]() reflect.Type { return reflect.TypeFor[T]() }

func Magic() int { return 42 }
GO
GOTOOLCHAIN=go1.27.1 go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run --no-config --default=none --enable=mnd ./...
```

Expected: the run **reports** the `mnd` finding on `42` (exit code 1 with a finding). A message containing `Go language version used to build golangci-lint is lower than the targeted Go version` means the pin cannot lint v2; stop and raise it with the user before continuing, because it blocks the whole v2 lint job.

- [ ] **Step 2: Create the workspace and the module**

`go.work`:

```
go 1.27

use (
	.
	./v2
)
```

`v2/go.mod`:

```
module flamingo.me/dingo/v2

go 1.27

require (
	flamingo.me/dingo v0.4.1
	github.com/stretchr/testify v1.12.1
)
```

`v0.4.1` is a placeholder, not the engine this module is built against. `flamingo.me/dingo` is a vanity path serving the upstream module, and the committed `go.work` resolves it to the local `.` module, so the version here is inert in every workspace build; it exists only so `v2/go.mod` is resolvable outside the workspace. Do not add a `replace`. See "How the v2 module resolves the engine"; Task B24 step 1a bumps this line to `v0.5.0` once Part A is released upstream, which is the expected end state rather than a contingency.

`v2/doc.go`:

```go
// Package dingo is the type-safe binding API of the dingo dependency injection container.
//
// It is a typed injector over the reflection engine in flamingo.me/dingo: both APIs bind into one
// injector, so a code base can migrate module by module. Package
// flamingo.me/dingo/v2/compat adapts injectors and modules in both directions.
//
// Bindings are declared in a module's Configure method and read back with GetInstance:
//
//	type Module struct{}
//
//	func (Module) Configure(injector *dingo.Injector) {
//		injector.Bind[TransactionLog]().To[DatabaseTransactionLog]()
//		injector.BindMulti[Filter]().To[MetricsFilter]()
//	}
//
//	injector, err := dingo.NewInjector(Module{})
//	if err != nil {
//		return err
//	}
//
//	service, err := injector.GetInstance[*BillingService]()
//
// Misuse that Go can express is a compile error. Everything else panics the moment the binding
// method runs, inside Configure, with an error wrapping ErrInvalidBinding that names the call and
// the types involved, so TryModule turns it into a returned error for module tests.
//
// This package requires Go 1.27 for generic methods.
package dingo
```

Check `.gitignore`: if it contains `go.work` or `go.work.sum`, remove those lines — both are committed for this repository.

- [ ] **Step 3: Verify both modules build**

```bash
cd /home/user/dingo
go build ./... ./v2/...
go vet ./... ./v2/...
CGO_ENABLED=1 go test -race ./... ./v2/...
go version   # GOTOOLCHAIN=auto should now report 1.27.x inside the workspace
```

Expected: all clean. `v2` has no tests yet, which `go test` reports as `[no test files]`, not a failure.

These commands all run **in workspace mode**, where `flamingo.me/dingo` resolves to the local `.` module. If the build cannot find `flamingo.me/dingo/internal/hooks`, Part A's code is not on this branch; rebase onto it rather than touching `v2/go.mod`.

Do not run `go mod tidy` inside `v2/` with `GOWORK=off` — outside the workspace it resolves the `require` line to the upstream `v0.4.1`, which has no `internal/hooks`, and will rewrite the file to something that cannot build.

- [ ] **Step 4: Commit**

```bash
git add go.work v2/go.mod v2/doc.go
git commit -m "feat(v2): add the flamingo.me/dingo/v2 module and the workspace"
```

### Task B2: the typed injector's lifecycle — injector, modules, scopes, errors

**Files:**
- Create: `v2/errors.go`, `v2/typename.go`, `v2/scope.go`, `v2/injector.go`, `v2/module.go`
- Create: `v2/helpers_test.go` (five of the six helpers; `get[T]` lands in B3)
- Create: `v2/module_test.go`
- Create: `v2/testdata/moduleidentity/commerce/cart/module.go`, `v2/testdata/moduleidentity/om3/cart/module.go`

**Interfaces:**
- Consumes: `hooks.Attached`, `hooks.Attach`, `hooks.RootOf`, `hooks.AsModule` (A2, A5); `v0.Injector`, `v0.Module`, `v0.Scope`, the root sentinels (A3).
- Produces, and every later Part B task depends on these exact signatures:
  - `type Injector struct{ root *v0.Injector }` with `func (injector *Injector) mustRoot(call string) *v0.Injector` and `func attachedOf(root *v0.Injector) *Injector`
  - `func NewInjector(modules ...Module) (*Injector, error)`, `func (injector *Injector) Child() (*Injector, error)`, `InitModules(modules ...Module) error`, `SetBuildEagerSingletons(build bool)`, `BuildEagerSingletons(includeParent bool) error`, `RequestInjection(object any) error`, `BindScope(scope Scope)`
  - `func EnableCircularTracing()`, `func EnableInjectionTracing()`
  - `type Module interface{ Configure(injector *Injector) }`, `type ModuleFunc func(injector *Injector)`, `type Depender interface{ Depends() []Module }`, `func TryModule(modules ...Module) error`
  - `type Scope = v0.Scope`, `type SingletonScope = v0.SingletonScope`, `type ChildSingletonScope = v0.ChildSingletonScope`, `var Singleton, ChildSingleton Scope`, `func NewSingletonScope() *SingletonScope`, `func NewChildSingletonScope() *ChildSingletonScope`
  - `var ErrInvalidBinding, ErrPointerToInterface, ErrInitModules, ErrInvalidInjectReceiver, ErrModuleCycle, ErrModuleSort error`
  - `func invalid(call, format string, args ...any) error`, `func qualified(typ reflect.Type) string`, `func typeName[T any]() string`, `func keyType[T any]() reflect.Type`, `func carrier(typ reflect.Type) any`
  - test helpers `newInjector(t, mods...) *Injector`, `childOf(t, parent) *Injector`, `bindErr(fn) error`, `bindPanic(t, fn) error`, `requireInvalidBinding(t, err, parts...)`

- [ ] **Step 1: Write the failing tests**

`v2/testdata/moduleidentity/commerce/cart/module.go`:

```go
// Package cart holds a module type whose name collides with another package's on purpose.
package cart

import dingo "flamingo.me/dingo/v2"

// Module is one of two same-named modules used to prove that module identity is the type, not
// the type's name.
type Module struct{ Configured *int }

// Configure counts its own invocation.
func (m *Module) Configure(_ *dingo.Injector) { *m.Configured++ }
```

`v2/testdata/moduleidentity/om3/cart/module.go`: identical file, same package name `cart`, same type `Module`. The two differ only by import path, which is the point.

`v2/helpers_test.go`:

```go
package dingo_test

import (
	"sync"
	"testing"

	dingo "flamingo.me/dingo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// privateScopes maps an injector to the private singleton scope newInjector gave it, so childOf
// can register the same object on a child.
var privateScopes sync.Map

// newInjector builds an injector with private Singleton and ChildSingleton caches.
//
// dingo.Singleton and dingo.ChildSingleton are package-level objects that every root injector
// registers, so two parallel tests binding the same type In(Singleton) would otherwise share one
// cache. Construction is two-phase because NewInjector(mods...) would build eager singletons
// against the global scopes before the swap.
func newInjector(t *testing.T, mods ...dingo.Module) *dingo.Injector {
	t.Helper()

	injector, err := dingo.NewInjector()
	require.NoError(t, err)

	singleton := dingo.NewSingletonScope()
	injector.BindScope(singleton)
	injector.BindScope(dingo.NewChildSingletonScope())
	privateScopes.Store(injector, singleton)

	require.NoError(t, injector.InitModules(mods...))

	return injector
}

// childOf derives a child that shares its parent's private singleton cache, as a production child
// shares the global one. The child keeps the fresh child-singleton scope Child() gave it.
func childOf(t *testing.T, parent *dingo.Injector) *dingo.Injector {
	t.Helper()

	child, err := parent.Child()
	require.NoError(t, err)

	singleton, ok := privateScopes.Load(parent)
	require.True(t, ok, "childOf needs a parent built by newInjector")

	scope, ok := singleton.(dingo.Scope)
	require.True(t, ok)

	child.BindScope(scope)
	privateScopes.Store(child, scope)

	return child
}

// bindErr runs fn as a module through TryModule, the route a module author's own test takes.
// TryModule disables eager singletons, so a nil return proves bind-time acceptance only.
func bindErr(fn func(*dingo.Injector)) error {
	return dingo.TryModule(dingo.ModuleFunc(fn))
}

// bindPanic runs fn outside TryModule and returns the recovered panic value as an error, so a
// test can assert on the raw panic path rather than on TryModule's conversion.
func bindPanic(t *testing.T, fn func(*dingo.Injector)) (recovered error) {
	t.Helper()

	defer func() {
		value := recover()
		require.NotNil(t, value, "expected a bind-time panic")

		err, ok := value.(error)
		require.True(t, ok, "a bind-time panic value must be an error, got %T", value)

		recovered = err
	}()

	injector := newInjector(t)
	fn(injector)

	return nil
}

// requireInvalidBinding asserts the sentinel, the message prefix and every expected substring.
func requireInvalidBinding(t *testing.T, err error, parts ...string) {
	t.Helper()

	require.ErrorIs(t, err, dingo.ErrInvalidBinding)
	assert.Contains(t, err.Error(), "dingo: ")

	for _, part := range parts {
		assert.Contains(t, err.Error(), part)
	}
}
```

`v2/module_test.go` — the catalogue's `module_test.go` rows M-01 to M-08; M-09 (eager singletons) is added in Task B11, where `AsEagerSingleton` is under test:

```go
package dingo_test

import (
	"errors"
	"sync/atomic"
	"testing"

	dingo "flamingo.me/dingo/v2"
	commerce "flamingo.me/dingo/v2/testdata/moduleidentity/commerce/cart"
	om3 "flamingo.me/dingo/v2/testdata/moduleidentity/om3/cart"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingModule appends its name to a shared log when Configure runs.
type recordingModule struct {
	name string
	log  *[]string
	deps []dingo.Module
}

func (m *recordingModule) Configure(_ *dingo.Injector) { *m.log = append(*m.log, m.name) }

func (m *recordingModule) Depends() []dingo.Module { return m.deps }

var errConfigure = errors.New("configure failed")

type panickingModule struct{ value any }

func (m panickingModule) Configure(_ *dingo.Injector) { panic(m.value) }

// TestModule_ConfigureCalledOncePerModule
// Covers M-01.
// Catches: a module registered twice being configured twice, which would double every binding it
// makes and trip duplicate detection.
func TestModule_ConfigureCalledOncePerModule(t *testing.T) {
	t.Parallel()

	var log []string

	newInjector(t, &recordingModule{name: "a", log: &log}, &recordingModule{name: "b", log: &log})
	assert.Equal(t, []string{"a", "b"}, log)
}

// TestModuleFunc_DistinctClosuresAreDistinctModules
// Covers M-02.
// Catches: ModuleFunc values keyed by type alone, which would drop every closure after the first.
func TestModuleFunc_DistinctClosuresAreDistinctModules(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64

	increment := func() dingo.ModuleFunc {
		return func(_ *dingo.Injector) { calls.Add(1) }
	}

	newInjector(t, increment(), increment())
	assert.Equal(t, int64(2), calls.Load())
}

// TestDepender_DependenciesConfiguredBeforeDependents
// Covers M-03.
// Catches: a topological sort that loses registration order among independent modules, which
// would make binding order — and therefore multibinding order — non-deterministic.
func TestDepender_DependenciesConfiguredBeforeDependents(t *testing.T) {
	t.Parallel()

	var log []string

	dependency := &recordingModule{name: "dependency", log: &log}
	dependent := &recordingModule{name: "dependent", log: &log, deps: []dingo.Module{dependency}}
	independent := &recordingModule{name: "independent", log: &log}

	newInjector(t, dependent, independent)
	assert.Equal(t, []string{"dependency", "dependent", "independent"}, log)
}

// TestModule_DuplicateModulesConfiguredOnce
// Covers M-04.
// Catches: identity by type name rather than by type, which would silently drop one of two
// same-named modules from different packages — the commerce/om3 shape.
func TestModule_DuplicateModulesConfiguredOnce(t *testing.T) {
	t.Parallel()

	t.Run("the same module value direct and transitive configures once", func(t *testing.T) {
		t.Parallel()

		var log []string

		shared := &recordingModule{name: "shared", log: &log}
		dependent := &recordingModule{name: "dependent", log: &log, deps: []dingo.Module{shared}}

		newInjector(t, shared, dependent)
		assert.Equal(t, []string{"shared", "dependent"}, log)
	})

	t.Run("same-named module types from different packages are distinct", func(t *testing.T) {
		t.Parallel()

		var commerceCalls, om3Calls int

		newInjector(t, &commerce.Module{Configured: &commerceCalls}, &om3.Module{Configured: &om3Calls})
		assert.Equal(t, 1, commerceCalls)
		assert.Equal(t, 1, om3Calls)
	})
}

// TestModule_CycleErrorNamesThePath
// Covers M-05.
// Catches: a cycle reported without the module names, which leaves the author no way to find it.
func TestModule_CycleErrorNamesThePath(t *testing.T) {
	t.Parallel()

	var log []string

	first := &recordingModule{name: "first", log: &log}
	second := &recordingModule{name: "second", log: &log, deps: []dingo.Module{first}}
	first.deps = []dingo.Module{second}

	err := dingo.TryModule(first)
	require.ErrorIs(t, err, dingo.ErrModuleCycle)
	assert.Contains(t, err.Error(), " → ")
}

// TestTryModule_PassesThroughErrorPanicsUnchanged
// Covers M-06.
// Catches: TryModule wrapping an error panic, which would break errors.Is for every bind-time
// sentinel a module test matches on.
func TestTryModule_PassesThroughErrorPanicsUnchanged(t *testing.T) {
	t.Parallel()

	assert.ErrorIs(t, dingo.TryModule(panickingModule{value: errConfigure}), errConfigure)
}

// TestTryModule_WrapsNonErrorPanicsWithQFormatting
// Covers M-07.
// Catches: a change of the %q rendering, which readers have learned to recognise.
func TestTryModule_WrapsNonErrorPanicsWithQFormatting(t *testing.T) {
	t.Parallel()

	err := dingo.TryModule(panickingModule{value: 42})
	require.Error(t, err)
	assert.Equal(t, `dingo.TryModule panic: '*'`, err.Error())
}

// TestInitModules_WrapsSortFailureInErrInitModules
// Covers M-08.
// Catches: a sort failure reported without ErrInitModules, which callers match on to tell a setup
// failure from a resolution failure.
func TestInitModules_WrapsSortFailureInErrInitModules(t *testing.T) {
	t.Parallel()

	var log []string

	first := &recordingModule{name: "first", log: &log}
	second := &recordingModule{name: "second", log: &log, deps: []dingo.Module{first}}
	first.deps = []dingo.Module{second}

	injector := newInjector(t)
	err := injector.InitModules(first)
	require.ErrorIs(t, err, dingo.ErrInitModules)
	assert.ErrorIs(t, err, dingo.ErrModuleCycle)
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./v2/ -v`
Expected: compile failure — `undefined: dingo.NewInjector` and every other symbol. That is the expected red.

- [ ] **Step 3: Write the implementation**

`v2/errors.go`:

```go
package dingo

import (
	"errors"
	"fmt"

	v0 "flamingo.me/dingo"
)

var (
	// ErrInvalidBinding is wrapped by the error value every bind-time check panics with, so a
	// module test can assert errors.Is(err, ErrInvalidBinding) on what TryModule returns.
	ErrInvalidBinding = errors.New("invalid binding")

	// ErrPointerToInterface is the engine's sentinel. Field injection wraps it as before, and
	// GetInstance[*Iface]() wraps it too.
	ErrPointerToInterface = v0.ErrPointerToInterface

	// ErrInitModules is returned when module initialization fails.
	ErrInitModules = v0.ErrInitModules
	// ErrInvalidInjectReceiver is returned for an Inject method on a value receiver.
	ErrInvalidInjectReceiver = v0.ErrInvalidInjectReceiver
	// ErrModuleCycle is returned for a cyclic module dependency.
	ErrModuleCycle = v0.ErrModuleCycle
	// ErrModuleSort is returned when the module graph cannot be sorted.
	ErrModuleSort = v0.ErrModuleSort
)

// bindError is the value every bind-time check panics with. It renders as
// "dingo: <call>: <reason>" and unwraps to ErrInvalidBinding.
type bindError struct {
	call   string
	reason string
}

func (e *bindError) Error() string { return "dingo: " + e.call + ": " + e.reason }

func (e *bindError) Unwrap() error { return ErrInvalidBinding }

// invalid builds the bind-time error for call.
func invalid(call, format string, args ...any) error {
	return &bindError{call: call, reason: fmt.Sprintf(format, args...)}
}
```

`v2/typename.go`:

```go
package dingo

import (
	"reflect"

	"flamingo.me/dingo/internal/typename"
)

// qualified prints typ with its full import path, the same way the engine names types in module
// diagnostics.
func qualified(typ reflect.Type) string { return typename.Qualified(typ) }

// typeName prints T with its full import path.
func typeName[T any]() string { return qualified(reflect.TypeFor[T]()) }

// keyType returns the type the engine keys a binding on: T with one pointer level removed. It
// does no validation; keyOf does that and reports the misuse.
func keyType[T any]() reflect.Type {
	typ := reflect.TypeFor[T]()
	if typ.Kind() == reflect.Pointer {
		return typ.Elem()
	}

	return typ
}

// carrier builds the value the engine's reflection API takes as a type carrier for typ: a *typ,
// from which the engine strips exactly one pointer level.
func carrier(typ reflect.Type) any { return reflect.New(typ).Interface() }
```

`v2/scope.go`:

```go
package dingo

import v0 "flamingo.me/dingo"

type (
	// Scope defines a scope's behaviour. It is the engine's Scope, so a scope registered through
	// either API serves In on the other.
	Scope = v0.Scope
	// SingletonScope handles singletons.
	SingletonScope = v0.SingletonScope
	// ChildSingletonScope manages child-specific singletons.
	ChildSingletonScope = v0.ChildSingletonScope
)

var (
	// Singleton is the default SingletonScope. It is one process-wide object that every injector
	// registers, so two independent injectors share singleton instances for the same key and
	// annotation. BindScope with a fresh NewSingletonScope() replaces the registration.
	Singleton = v0.Singleton
	// ChildSingleton scopes an instance to one injector. It is a process-wide object too; only
	// Child() registers a fresh one on the child.
	ChildSingleton = v0.ChildSingleton
)

// NewSingletonScope creates a new singleton scope.
func NewSingletonScope() *SingletonScope { return v0.NewSingletonScope() }

// NewChildSingletonScope creates a new child singleton scope.
func NewChildSingletonScope() *ChildSingletonScope { return v0.NewChildSingletonScope() }
```

`v2/injector.go`:

```go
package dingo

import (
	v0 "flamingo.me/dingo"
	"flamingo.me/dingo/internal/hooks"
)

// zeroInjector is the reason every method reports for a zero or nil Injector.
const zeroInjector = "zero Injector, use NewInjector, Child or compat.Injector"

// Injector defines bindings and multibindings. It is a typed injector over the root injector in
// flamingo.me/dingo; both APIs share one set of bindings, scopes and interceptors.
//
// Generic methods cannot appear in an interface, so *Injector can never be abstracted behind
// one: code that accepts "an injector" takes *Injector directly.
type Injector struct {
	root *v0.Injector
}

// mustRoot returns the root injector, or panics with the zero-value message naming call.
func (injector *Injector) mustRoot(call string) *v0.Injector {
	if injector == nil || injector.root == nil {
		panic(invalid(call, zeroInjector))
	}

	return injector.root
}

// attachedOf returns the typed injector the root attached when it built this root injector.
func attachedOf(root *v0.Injector) *Injector {
	attached, ok := hooks.Attached(root).(*Injector)
	if !ok {
		// coverage: unreachable through the public API; linking this package installs the hook
		// the root calls in every constructor
		panic(invalid("Injector", "root injector has no typed API"))
	}

	return attached
}

// NewInjector builds a new Injector out of a list of Modules.
func NewInjector(modules ...Module) (*Injector, error) {
	engine, err := v0.NewInjector()
	if err != nil {
		return nil, err //nolint:wrapcheck // the engine's error is returned verbatim so errors.Is and the message stay the engine's
	}

	injector := attachedOf(engine)

	return injector, injector.InitModules(modules...)
}

// Child derives a child injector with a new ChildSingletonScope. It returns an error, rather than
// panicking, for a zero or nil Injector, because the engine's Child does the same for a nil
// receiver.
func (injector *Injector) Child() (*Injector, error) {
	if injector == nil || injector.root == nil {
		return nil, invalid("Injector.Child", zeroInjector)
	}

	child, err := injector.root.Child()
	if err != nil {
		return nil, err //nolint:wrapcheck // the engine's error is returned verbatim
	}

	return attachedOf(child), nil
}

// InitModules initializes the injector with the given modules.
func (injector *Injector) InitModules(modules ...Module) error {
	engine := injector.mustRoot("Injector.InitModules")

	return engine.InitModules(asModules(modules)...) //nolint:wrapcheck // the engine's error is returned verbatim
}

// SetBuildEagerSingletons enables or disables building eager singletons during InitModules.
func (injector *Injector) SetBuildEagerSingletons(build bool) {
	injector.mustRoot("Injector.SetBuildEagerSingletons").SetBuildEagerSingletons(build)
}

// BuildEagerSingletons requests one instance of each eager singleton, optionally letting the
// parent injectors do the same.
func (injector *Injector) BuildEagerSingletons(includeParent bool) error {
	engine := injector.mustRoot("Injector.BuildEagerSingletons")

	return engine.BuildEagerSingletons(includeParent) //nolint:wrapcheck // the engine's error is returned verbatim
}

// RequestInjection requests that every field of object tagged with `inject` be filled. During
// InitModules the request is queued and runs after every module has configured; afterwards it
// injects immediately.
func (injector *Injector) RequestInjection(object any) error {
	engine := injector.mustRoot("Injector.RequestInjection")

	return engine.RequestInjection(object) //nolint:wrapcheck // the engine's error is returned verbatim
}

// BindScope makes the injector aware of a scope.
func (injector *Injector) BindScope(scope Scope) {
	injector.mustRoot("Injector.BindScope").BindScope(scope)
}

// EnableCircularTracing activates the engine's circular dependency trace. It is expensive and
// process-wide; use it for debugging only.
func EnableCircularTracing() { v0.EnableCircularTracing() }

// EnableInjectionTracing makes the engine log each resolution and each field it sets. It is
// process-wide and has no off switch.
func EnableInjectionTracing() { v0.EnableInjectionTracing() }
```

`v2/module.go`:

```go
package dingo

import (
	"fmt"

	v0 "flamingo.me/dingo"
	"flamingo.me/dingo/internal/hooks"
)

type (
	// Module is the entry point for dingo modules. Configure is called once during
	// initialization and sets up bindings on the provided Injector.
	Module interface {
		Configure(injector *Injector)
	}

	// ModuleFunc wraps a func(injector *Injector) so that a small function can be a module.
	ModuleFunc func(injector *Injector)

	// Depender lists modules that must be configured before this one.
	Depender interface {
		Depends() []Module
	}
)

// Configure calls the wrapped function.
func (f ModuleFunc) Configure(injector *Injector) { f(injector) }

// asModule adapts a v2 Module so the engine can run it. It carries no inject fields of its
// own: the engine injects hooks.Unwrap(module), which is the v2 module itself, immediately
// before Configure.
type asModule struct{ module Module }

// Configure runs the v2 module against the root injector's typed injector.
func (m asModule) Configure(root *v0.Injector) { m.module.Configure(attachedOf(root)) }

// Depends adapts the v2 module's dependencies, returning nil when it declares none.
func (m asModule) Depends() []v0.Module {
	depender, ok := m.module.(Depender)
	if !ok {
		return nil
	}

	return asModules(depender.Depends())
}

// DingoUnwrap reports the module this adapter wraps, so the engine keys its module graph
// by the unwrapped module.
func (m asModule) DingoUnwrap() any { return m.module }

// asModules adapts a slice of v2 modules, preserving a nil slice as nil.
func asModules(modules []Module) []v0.Module {
	if modules == nil {
		return nil
	}

	adapted := make([]v0.Module, len(modules))
	for i, module := range modules {
		adapted[i] = asModule{module: module}
	}

	return adapted
}

// TryModule tests whether modules are properly bound. It disables eager singletons, so it proves
// bind-time acceptance rather than construction, and turns a panic into a returned error.
func TryModule(modules ...Module) (resultingError error) {
	defer func() {
		if err := recover(); err != nil {
			if err, ok := err.(error); ok {
				resultingError = err

				return
			}

			resultingError = fmt.Errorf("dingo.TryModule panic: %q", err)
		}
	}()

	engine, err := v0.NewInjector()
	if err != nil {
		return err //nolint:wrapcheck // the engine's error is returned verbatim
	}

	injector := attachedOf(engine)
	injector.SetBuildEagerSingletons(false)

	return injector.InitModules(modules...)
}

func init() {
	hooks.Attach = func(root any) any {
		typed, ok := root.(*v0.Injector)
		if !ok {
			return nil
		}

		attached := &Injector{root: typed}
		typed.Bind(Injector{}).ToInstance(attached)

		return attached
	}

	hooks.RootOf = func(attached any) any {
		injector, ok := attached.(*Injector)
		if !ok || injector == nil {
			return nil
		}

		return injector.root
	}

	hooks.AsModule = func(module any) any {
		typed, ok := module.(Module)
		if !ok {
			return nil
		}

		return asModule{module: typed}
	}
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./v2/ -v -run 'TestModule|TestDepender|TestTryModule|TestInitModules'`
Expected: PASS, ten test functions and subtests.

- [ ] **Step 5: Vet, format, lint**

Run: `go vet ./... ./v2/... && gofmt -l . && cd v2 && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./...`
Expected: clean. If `wsl_v5` complains about a missing blank line before a `return` that follows a statement, add it; that rule is the most common finding in this code base.

- [ ] **Step 6: Commit**

```bash
git add v2/errors.go v2/typename.go v2/scope.go v2/injector.go v2/module.go \
        v2/helpers_test.go v2/module_test.go v2/testdata/moduleidentity
git commit -m "feat(v2): add the typed injector, modules, scopes and error sentinels"
```

### Task B3: keys, the binding builder and resolution

**Files:**
- Create: `v2/bind.go`, `v2/binding.go`, `v2/resolve.go`
- Create: `v2/fixtures_test.go`, `v2/bind_key_test.go`
- Modify: `v2/helpers_test.go` (add `get[T]` and the four helper contract tests)

**Interfaces:**
- Consumes: everything B2 produced.
- Produces, relied on by B4 to B21:
  - `func (injector *Injector) Bind[T any]() *Binding[T]`, `BindMulti[T any]() *Binding[T]`, `BindMap[T any](key string) *Binding[T]`, `Override[T any](annotatedWith string) *Binding[T]`
  - `type Binding[T any]` with `To[U any]() *Binding[T]`, `ToType(target reflect.Type) *Binding[T]`, `ToInstance(instance T) *Binding[T]`, `ToProvider(provider any) *Binding[T]`, `AnnotatedWith(annotation string) *Binding[T]`, `In(scope Scope) *Binding[T]`, `AsEagerSingleton() *Binding[T]`
  - `func (injector *Injector) GetInstance[T any]() (T, error)`, `GetAnnotatedInstance[T any](annotatedWith string) (T, error)`
  - test helper `get[T any](t *testing.T, injector *dingo.Injector) T`

This task delivers the builder **complete**: B4 and B5 add no production code, only the tables that pin each rule. Splitting the checks across tasks would mean writing `binding.go` four times.

- [ ] **Step 1: Write the failing tests**

`v2/fixtures_test.go` — the whole fixture set the suite uses, grouped by the rule each group exists to prove. Later tasks add to it; nothing here is test-specific:

```go
package dingo_test

import (
	"reflect"
	"sync/atomic"

	dingo "flamingo.me/dingo/v2"
)

// greeter and its implementations: value- and pointer-receiver forms of one interface (B-06).
type greeter interface{ Greet() string }

type politeGreeter struct{}

func (politeGreeter) Greet() string { return "hello" }

type loudGreeter struct{}

func (loudGreeter) Greet() string { return "HELLO" }

// ptrOnlyGreeter implements greeter only through *ptrOnlyGreeter.
type ptrOnlyGreeter struct{}

func (p *ptrOnlyGreeter) Greet() string { return "hi" }

// unrelated implements nothing, for the assignability rejections (B-05).
type unrelated struct{}

// counter is a struct with a mutable field, for the GetInstance[Service]() copy assertion (R-02).
// It is never used as a value-typed inject field, which the engine rejects (R-32).
type counter struct{ n int }

// label is a named string: a non-struct value type, legal as a value-typed inject field (R-17,
// R-18).
type label string

// Provider-suffixed function types the engine generates implementations for (R-28..R-30,
// MB-15, MB-16).
type (
	greeterProvider    func() greeter
	greetersProvider   func() []greeter
	greeterMapProvider func() map[string]greeter
)

// plainGreeterFunc has no Provider suffix, so the engine refuses to create it (R-08).
type plainGreeterFunc func() greeter

// countingScope is a user scope that counts constructions (S-03).
type countingScope struct {
	constructed atomic.Int64
	instances   map[string]reflect.Value
}

func newCountingScope() *countingScope {
	return &countingScope{instances: make(map[string]reflect.Value)}
}

func (s *countingScope) ResolveType(
	typ reflect.Type,
	annotation string,
	unscoped func(reflect.Type, string, bool) (reflect.Value, error),
) (reflect.Value, error) {
	key := typ.String() + "|" + annotation
	if instance, ok := s.instances[key]; ok {
		return instance, nil
	}

	instance, err := unscoped(typ, annotation, false)
	if err != nil {
		return reflect.Value{}, err
	}

	s.constructed.Add(1)
	s.instances[key] = instance

	return instance, nil
}

// Interceptor fixtures. Field 0 is exported because the engine writes the intercepted value into
// it with reflect.Value.Set, which panics for an unexported field, and v2 rejects that shape at
// bind time (B-34a).
type recordingInterceptor1 struct {
	Wrapped greeter
	Log     *[]string `inject:""`
}

func (i *recordingInterceptor1) Greet() string { return "1(" + i.Wrapped.Greet() + ")" }

type recordingInterceptor2 struct{ Wrapped greeter }

func (i *recordingInterceptor2) Greet() string { return "2(" + i.Wrapped.Greet() + ")" }

// greeterModule is the smallest module that binds something, for tests that need one.
type greeterModule struct{}

func (greeterModule) Configure(injector *dingo.Injector) {
	injector.Bind[greeter]().To[politeGreeter]()
}
```

Add `get[T]` and the four contract tests to `v2/helpers_test.go`:

```go
// get resolves T and fails the test when resolution errors.
func get[T any](t *testing.T, injector *dingo.Injector) T {
	t.Helper()

	instance, err := injector.GetInstance[T]()
	require.NoError(t, err)

	return instance
}

// TestHelper_PrivateSingletonScopeIsolatesTwoRoots pins newInjector's Singleton swap.
// Catches: two parallel tests sharing one global singleton cache, an order dependence that
// -shuffle=on finds and that makes an unrelated test fail.
func TestHelper_PrivateSingletonScopeIsolatesTwoRoots(t *testing.T) {
	t.Parallel()

	bind := dingo.ModuleFunc(func(injector *dingo.Injector) {
		injector.Bind[counter]().In(dingo.Singleton)
	})

	first := get[*counter](t, newInjector(t, bind))
	second := get[*counter](t, newInjector(t, bind))
	assert.NotSame(t, first, second)
}

// TestHelper_PrivateChildSingletonScopeIsolatesTwoRoots pins newInjector's ChildSingleton swap.
// Catches: ChildSingleton being a package-level object too, which only Child() replaces, so two
// roots would share one cache for a root-level In(ChildSingleton) binding.
func TestHelper_PrivateChildSingletonScopeIsolatesTwoRoots(t *testing.T) {
	t.Parallel()

	bind := dingo.ModuleFunc(func(injector *dingo.Injector) {
		injector.Bind[counter]().In(dingo.ChildSingleton)
	})

	first := get[*counter](t, newInjector(t, bind))
	second := get[*counter](t, newInjector(t, bind))
	assert.NotSame(t, first, second)
}

// TestHelper_ChildSharesParentsPrivateSingletonScope pins childOf's scope swap.
// Catches: a child registering the global Singleton, which would break the parent/child sharing
// every scope test depends on.
func TestHelper_ChildSharesParentsPrivateSingletonScope(t *testing.T) {
	t.Parallel()

	parent := newInjector(t, dingo.ModuleFunc(func(injector *dingo.Injector) {
		injector.Bind[counter]().In(dingo.Singleton)
	}))
	child := childOf(t, parent)

	assert.Same(t, get[*counter](t, parent), get[*counter](t, child))
}

// TestHelper_RequireInvalidBindingAssertsSentinelAndPrefix pins the shared assertion helper.
// Catches: a helper that passes on any error, which would make every rejection table vacuous.
func TestHelper_RequireInvalidBindingAssertsSentinelAndPrefix(t *testing.T) {
	t.Parallel()

	err := bindErr(func(injector *dingo.Injector) { injector.Bind[*greeter]() })
	requireInvalidBinding(t, err, "Bind[", "pointer to interface")
	require.ErrorIs(t, err, dingo.ErrInvalidBinding)
	assert.True(t, strings.HasPrefix(err.Error(), "dingo: "))
}
```

Add `"strings"` to `helpers_test.go`'s imports.

`v2/bind_key_test.go` — what `T` means and the shapes all four entry points reject:

```go
package dingo_test

import (
	"testing"

	dingo "flamingo.me/dingo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBind_KeyIsTWithOnePointerLevelRemoved documents what T means for each shape.
// Covers K-01, K-02, K-03, K-04, K-05, K-06, K-07.
// Catches: a key computation that strips every pointer level, which would make Bind[**T]() and
// Bind[*T]() collide, or that strips none, which would make Bind[*S]().ToInstance(&s) unreadable
// from an S-typed injection site.
func TestBind_KeyIsTWithOnePointerLevelRemoved(t *testing.T) {
	t.Parallel()

	t.Run("an interface T keys on the interface", func(t *testing.T) {
		t.Parallel()

		injector := newInjector(t, dingo.ModuleFunc(func(i *dingo.Injector) {
			i.Bind[greeter]().To[politeGreeter]()
		}))
		assert.Equal(t, "hello", get[greeter](t, injector).Greet())
	})

	t.Run("a struct value T keys on the struct", func(t *testing.T) {
		t.Parallel()

		injector := newInjector(t, dingo.ModuleFunc(func(i *dingo.Injector) {
			i.Bind[counter]().ToInstance(&counter{n: 3})
		}))
		assert.Equal(t, 3, get[*counter](t, injector).n)
	})

	t.Run("Bind[*Service] and Bind[Service] key to the same slot", func(t *testing.T) {
		t.Parallel()

		injector := newInjector(t, dingo.ModuleFunc(func(i *dingo.Injector) {
			i.Bind[*counter]().ToInstance(&counter{n: 7})
		}))
		assert.Equal(t, 7, get[*counter](t, injector).n)
		assert.Equal(t, 7, get[counter](t, injector).n)
	})

	t.Run("a basic type T keys on the basic type", func(t *testing.T) {
		t.Parallel()

		injector := newInjector(t, dingo.ModuleFunc(func(i *dingo.Injector) {
			i.Bind[string]().ToInstance("value")
		}))
		assert.Equal(t, "value", get[string](t, injector))
	})

	t.Run("a slice T keys on the slice type", func(t *testing.T) {
		t.Parallel()

		injector := newInjector(t, dingo.ModuleFunc(func(i *dingo.Injector) {
			i.Bind[[]greeter]().ToInstance([]greeter{politeGreeter{}})
		}))
		assert.Len(t, get[[]greeter](t, injector), 1)
	})

	t.Run("a map T keys on the map type", func(t *testing.T) {
		t.Parallel()

		injector := newInjector(t, dingo.ModuleFunc(func(i *dingo.Injector) {
			i.Bind[map[string]greeter]().ToInstance(map[string]greeter{"a": politeGreeter{}})
		}))
		assert.Len(t, get[map[string]greeter](t, injector), 1)
	})

	t.Run("a Provider-suffixed func type needs no Bind of its own", func(t *testing.T) {
		t.Parallel()

		injector := newInjector(t, dingo.ModuleFunc(func(i *dingo.Injector) {
			i.Bind[greeter]().To[politeGreeter]()
		}))
		assert.Equal(t, "hello", get[greeterProvider](t, injector)().Greet())
	})
}

// TestBind_RejectsIllegalTypeParameters documents which T the four entry points accept.
// Covers B-01, B-02, B-03, B-04, K-08, K-09.
// v0: Bind[*Iface] was not expressible; Bind(new(*Foo)) silently made a binding nothing reads.
// Catches: a check that strips pointers before testing for interface kind, which would accept
// Bind[*Iface]() and create a key no injection site can reach.
func TestBind_RejectsIllegalTypeParameters(t *testing.T) {
	t.Parallel()

	entries := []struct {
		name              string
		pointerToIface    func(*dingo.Injector)
		plainIface        func(*dingo.Injector)
		pointerToPointer  func(*dingo.Injector)
		pointerToStruct   func(*dingo.Injector)
		call              string
	}{
		{
			name:             "Bind",
			pointerToIface:   func(i *dingo.Injector) { i.Bind[*greeter]() },
			plainIface:       func(i *dingo.Injector) { i.Bind[greeter]().To[politeGreeter]() },
			pointerToPointer: func(i *dingo.Injector) { i.Bind[**counter]() },
			pointerToStruct:  func(i *dingo.Injector) { i.Bind[*counter]().ToInstance(&counter{n: 1}) },
			call:             "Bind[",
		},
		{
			name:             "BindMulti",
			pointerToIface:   func(i *dingo.Injector) { i.BindMulti[*greeter]() },
			plainIface:       func(i *dingo.Injector) { i.BindMulti[greeter]().To[politeGreeter]() },
			pointerToPointer: func(i *dingo.Injector) { i.BindMulti[**counter]() },
			pointerToStruct:  func(i *dingo.Injector) { i.BindMulti[*counter]().ToInstance(&counter{n: 1}) },
			call:             "BindMulti[",
		},
		{
			name:             "BindMap",
			pointerToIface:   func(i *dingo.Injector) { i.BindMap[*greeter]("k") },
			plainIface:       func(i *dingo.Injector) { i.BindMap[greeter]("k").To[politeGreeter]() },
			pointerToPointer: func(i *dingo.Injector) { i.BindMap[**counter]("k") },
			pointerToStruct:  func(i *dingo.Injector) { i.BindMap[*counter]("k").ToInstance(&counter{n: 1}) },
			call:             "BindMap[",
		},
		{
			name:             "Override",
			pointerToIface:   func(i *dingo.Injector) { i.Override[*greeter]("a") },
			plainIface:       func(i *dingo.Injector) { i.Override[greeter]("a").To[politeGreeter]() },
			pointerToPointer: func(i *dingo.Injector) { i.Override[**counter]("a") },
			pointerToStruct:  func(i *dingo.Injector) { i.Override[*counter]("a").ToInstance(&counter{n: 1}) },
			call:             "Override[",
		},
	}

	for _, entry := range entries {
		t.Run(entry.name, func(t *testing.T) {
			t.Parallel()

			t.Run("a pointer to an interface is rejected, as it is at injection sites", func(t *testing.T) {
				t.Parallel()

				requireInvalidBinding(t, bindErr(entry.pointerToIface),
					entry.call, "flamingo.me/dingo/v2_test.greeter", "pointer to interface")
			})

			t.Run("a plain interface is accepted", func(t *testing.T) {
				t.Parallel()

				require.NoError(t, bindErr(entry.plainIface))
			})

			t.Run("a pointer to a pointer is rejected", func(t *testing.T) {
				t.Parallel()

				requireInvalidBinding(t, bindErr(entry.pointerToPointer),
					entry.call, "flamingo.me/dingo/v2_test.counter", "pointer to pointer")
			})

			t.Run("a pointer to a struct is accepted and keys on the struct", func(t *testing.T) {
				t.Parallel()

				require.NoError(t, bindErr(entry.pointerToStruct))
			})
		})
	}
}

// TestBinding_ZeroValuePanics
// Covers B-39.
// Catches: a Binding[T] whose methods silently no-op when it was not created by an Injector, so a
// misuse produces an injector with no binding and a resolution failure far from the cause.
// dingo.Binding[T]{}.To[U]() does not compile ("cannot call pointer method"), so the test uses an
// addressable variable.
func TestBinding_ZeroValuePanics(t *testing.T) {
	t.Parallel()

	t.Run("a zero value", func(t *testing.T) {
		t.Parallel()

		var binding dingo.Binding[greeter]

		err := recovered(t, func() { binding.To[politeGreeter]() })
		requireInvalidBinding(t, err, "Binding[", "flamingo.me/dingo/v2_test.greeter", "not created by an Injector")
	})

	t.Run("a new pointer", func(t *testing.T) {
		t.Parallel()

		err := recovered(t, func() { new(dingo.Binding[greeter]).ToInstance(politeGreeter{}) })
		requireInvalidBinding(t, err, "Binding[", "not created by an Injector")
	})
}

// TestInjector_ZeroValuePanics
// Covers B-40.
// Catches: a nil engine field reaching the engine and producing a nil-pointer dereference instead
// of a message that says which constructor to use.
func TestInjector_ZeroValuePanics(t *testing.T) {
	t.Parallel()

	calls := map[string]func(*dingo.Injector){
		"Bind":                    func(i *dingo.Injector) { i.Bind[greeter]() },
		"BindMulti":               func(i *dingo.Injector) { i.BindMulti[greeter]() },
		"BindMap":                 func(i *dingo.Injector) { i.BindMap[greeter]("k") },
		"Override":                func(i *dingo.Injector) { i.Override[greeter]("a") },
		"BindInterceptor":         func(i *dingo.Injector) { i.BindInterceptor[greeter, recordingInterceptor2]() },
		"BindScope":               func(i *dingo.Injector) { i.BindScope(dingo.Singleton) },
		"InitModules":             func(i *dingo.Injector) { _ = i.InitModules() },
		"SetBuildEagerSingletons": func(i *dingo.Injector) { i.SetBuildEagerSingletons(false) },
		"BuildEagerSingletons":    func(i *dingo.Injector) { _, _ = nil, i.BuildEagerSingletons(false) },
		"RequestInjection":        func(i *dingo.Injector) { _ = i.RequestInjection(&counter{}) },
		"Inspect":                 func(i *dingo.Injector) { i.Inspect(dingo.Inspector{}) },
		"GetInstance":             func(i *dingo.Injector) { _, _ = i.GetInstance[greeter]() },
		"GetAnnotatedInstance":    func(i *dingo.Injector) { _, _ = i.GetAnnotatedInstance[greeter]("a") },
	}

	for name, call := range calls {
		t.Run(name+" on a zero Injector", func(t *testing.T) {
			t.Parallel()

			var injector dingo.Injector

			requireInvalidBinding(t, recovered(t, func() { call(&injector) }), "Injector."+name, "zero Injector")
		})

		t.Run(name+" on a nil Injector", func(t *testing.T) {
			t.Parallel()

			requireInvalidBinding(t, recovered(t, func() { call(nil) }), "Injector."+name, "zero Injector")
		})
	}
}

// TestInjector_ZeroValueChildReturnsAnError
// Covers B-45.
// Catches: Child panicking on a zero Injector, which would lose the engine's nil-receiver
// behavior that callers handle as an error today.
func TestInjector_ZeroValueChildReturnsAnError(t *testing.T) {
	t.Parallel()

	var zero dingo.Injector

	child, err := zero.Child()
	assert.Nil(t, child)
	requireInvalidBinding(t, err, "Injector.Child", "zero Injector")

	var nilInjector *dingo.Injector

	child, err = nilInjector.Child()
	assert.Nil(t, child)
	requireInvalidBinding(t, err, "Injector.Child", "zero Injector")
}
```

`recovered` is a second panic helper, for calls that do not take an injector; add it to `helpers_test.go`:

```go
// recovered runs fn and returns the error it panicked with.
func recovered(t *testing.T, fn func()) (err error) {
	t.Helper()

	defer func() {
		value := recover()
		require.NotNil(t, value, "expected a panic")

		panicked, ok := value.(error)
		require.True(t, ok, "a bind-time panic value must be an error, got %T", value)

		err = panicked
	}()

	fn()

	return nil
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./v2/ -v`
Expected: compile failure — `Bind`, `Binding`, `GetInstance`, `Inspect` and `BindInterceptor` are undefined. `Inspect` and `BindInterceptor` are referenced by `TestInjector_ZeroValuePanics`; they arrive in B14 and B15. To keep this task's red honest, comment those two map entries out now and re-enable them in B14 and B15, where the task says so.

- [ ] **Step 3: Write the implementation**

`v2/bind.go`:

```go
package dingo

import (
	"reflect"
	"strconv"
	"strings"
)

// callOf renders an entry point with its type argument for a message: Bind[pkg.Iface].
func callOf[T any](entry string) string { return entry + "[" + typeName[T]() + "]" }

// keyOf validates T as a binding key for the entry point named entry and returns the key the
// engine binds on: T with one pointer level removed.
//
// A pointer to an interface is rejected because the engine rejects a *Iface injection site before
// it looks a binding up, so the key would be unreachable. A pointer to a pointer is rejected
// because a **T site strips one level and looks up *T, whose entries live under T after the
// engine's own stripping: v0's Bind(new(*X)) was read by nothing.
func keyOf[T any](call, entry string) reflect.Type {
	typ := reflect.TypeFor[T]()
	if typ.Kind() != reflect.Pointer {
		return typ
	}

	elem := typ.Elem()

	if elem.Kind() == reflect.Interface {
		panic(invalid(call, "pointer to interface, use %s[%s]", entry, qualified(elem)))
	}

	if elem.Kind() == reflect.Pointer {
		panic(invalid(call, "pointer to pointer is not allowed"))
	}

	return elem
}

// collectionKeyOf validates T as a multibinding or map-binding key. It adds one rejection to
// keyOf's: a Provider-suffixed function type. A []XProvider or map[string]XProvider injection site
// does not look up XProvider; the engine takes Out(0) of the element type and looks up X's
// collection, so entries registered under XProvider are read by nothing and v0 resolved such a
// site to an empty slice with no error.
func collectionKeyOf[T any](call, entry, collection string) reflect.Type {
	key := keyOf[T](call, entry)

	if key.Kind() != reflect.Func || !strings.HasSuffix(key.Name(), "Provider") {
		return key
	}

	if key.NumOut() == 0 {
		panic(invalid(call, "a Provider-suffixed function type is never read from a %s", collection))
	}

	panic(invalid(call, "a Provider-suffixed function type is never read from a %s, use %s[%s]",
		collection, entry, qualified(key.Out(0))))
}

// Bind creates a new binding for T. The binding key is T with one pointer level removed, so
// Bind[Iface](), Bind[Service]() and Bind[*Service]() key on Iface, Service and Service.
func (injector *Injector) Bind[T any]() *Binding[T] {
	engine := injector.mustRoot("Injector." + callOf[T]("Bind"))
	call := callOf[T]("Bind")

	return &Binding[T]{binding: engine.Bind(carrier(keyOf[T](call, "Bind"))), call: call}
}

// BindMulti adds one entry to the multibinding of T, injected as []T.
func (injector *Injector) BindMulti[T any]() *Binding[T] {
	engine := injector.mustRoot("Injector." + callOf[T]("BindMulti"))
	call := callOf[T]("BindMulti")
	key := collectionKeyOf[T](call, "BindMulti", "multibinding")

	return &Binding[T]{binding: engine.BindMulti(carrier(key)), call: call}
}

// BindMap adds one entry to the map binding of T under key, injected as map[string]T.
func (injector *Injector) BindMap[T any](key string) *Binding[T] {
	engine := injector.mustRoot("Injector." + callOf[T]("BindMap"))
	call := callOf[T]("BindMap") + "(" + strconv.Quote(key) + ")"
	bindType := collectionKeyOf[T](call, "BindMap", "map binding")

	return &Binding[T]{binding: engine.BindMap(carrier(bindType), key), call: call}
}

// Override replaces every binding of T carrying annotatedWith, once every module has configured.
// Overriding a key nothing bound is silently accepted and becomes a plain binding: the engine
// binds first and evaluates overrides afterwards, so the "unknown binding" branch is unreachable.
func (injector *Injector) Override[T any](annotatedWith string) *Binding[T] {
	engine := injector.mustRoot("Injector." + callOf[T]("Override"))
	call := callOf[T]("Override") + "(" + strconv.Quote(annotatedWith) + ")"
	key := keyOf[T](call, "Override")

	binding := &Binding[T]{binding: engine.Override(carrier(key), annotatedWith), call: call}
	binding.annotation = &annotatedWith

	return binding
}
```

`v2/binding.go`:

```go
package dingo

import (
	"reflect"
	"strconv"
)

const (
	providerMinResults = 1
	providerMaxResults = 2
	providerErrorIndex = 1
)

var errorType = reflect.TypeFor[error]()

// Binding configures how one bound type is resolved. Only an Injector creates usable values;
// every method on a zero Binding panics with an error wrapping ErrInvalidBinding.
//
// Methods may be called in any order. A binding has at most one target, one annotation and one
// scope; a second, different one panics and names the call that set the first.
type Binding[T any] struct {
	binding *v0Binding
	call    string

	target     string  // the method that set the target, empty while none has
	annotation *string // nil while unset, so that an empty annotation is still "set"
	scope      Scope
}

// must returns the engine binding, or panics with the zero-value message naming method.
func (b *Binding[T]) must(method string) *v0Binding {
	if b == nil || b.binding == nil {
		panic(invalid("Binding["+typeName[T]()+"]."+method,
			"not created by an Injector, use injector.Bind"))
	}

	return b.binding
}

// setTarget records that method set this binding's target, rejecting a second one.
func (b *Binding[T]) setTarget(call, method string) {
	if b.target != "" {
		panic(invalid(call, "target already set by %s", b.target))
	}

	b.target = method
}

// checkTarget validates a binding target and returns it with every pointer level removed, which
// is what the engine stores. A pointer to an interface is rejected before stripping, so that
// To[Iface2] — binding one interface to another bound interface — stays legal.
func (b *Binding[T]) checkTarget(call string, target reflect.Type) reflect.Type {
	if target.Kind() == reflect.Pointer && target.Elem().Kind() == reflect.Interface {
		panic(invalid(call, "pointer to interface, use %s", qualified(target.Elem())))
	}

	for target.Kind() == reflect.Pointer {
		target = target.Elem()
	}

	key := keyType[T]()

	if target == key {
		panic(invalid(call, "binding to itself"))
	}

	if !target.AssignableTo(key) && !reflect.PointerTo(target).AssignableTo(key) {
		panic(invalid(call, "%s is not assignable to %s", qualified(target), qualified(key)))
	}

	return target
}

// setScope records this binding's scope, rejecting a different one. Scopes are compared by Go
// type because that is how the engine looks them up.
func (b *Binding[T]) setScope(call string, scope Scope) {
	if b.scope != nil && reflect.TypeOf(b.scope) != reflect.TypeOf(scope) {
		panic(invalid(call, "scope already set to %s", qualified(reflect.TypeOf(b.scope))))
	}

	b.scope = scope
}

// To binds U as the target. U is used with every pointer level removed, and U or *U must be
// assignable to the key. The engine constructs the target and injects into it.
func (b *Binding[T]) To[U any]() *Binding[T] {
	binding := b.must("To")
	method := "To[" + typeName[U]() + "]"
	call := b.call + "." + method

	b.setTarget(call, method)
	binding.To(carrier(b.checkTarget(call, reflect.TypeFor[U]())))

	return b
}

// ToType is To[U] for a U known only at run time, the v2 form of v0's .To(value). The target is
// still constructed and injected, unlike ToInstance, which returns the value verbatim.
func (b *Binding[T]) ToType(target reflect.Type) *Binding[T] {
	binding := b.must("ToType")

	if target == nil {
		panic(invalid(b.call+".ToType", "nil type"))
	}

	method := "ToType(" + qualified(target) + ")"
	call := b.call + "." + method

	b.setTarget(call, method)
	binding.To(carrier(b.checkTarget(call, target)))

	return b
}

// ToInstance binds an existing value. The compiler checks its type; a nil interface value is
// rejected, which is only possible when T is an interface. A typed nil pointer stays accepted and
// resolves to exactly that.
func (b *Binding[T]) ToInstance(instance T) *Binding[T] {
	binding := b.must("ToInstance")
	call := b.call + ".ToInstance"

	b.setTarget(call, "ToInstance")

	value := any(instance)
	if value == nil {
		panic(invalid(call, "nil instance"))
	}

	binding.ToInstance(value)

	return b
}

// ToProvider binds a function that builds the value. It must be a non-nil function with one or
// two results; the first must be assignable to the key or to a pointer to it, and a second must
// implement error. The provider's arguments are resolved by the injector when it runs.
//
// The error result is not propagated at resolution time, as in v0.4.1.
func (b *Binding[T]) ToProvider(provider any) *Binding[T] {
	binding := b.must("ToProvider")
	call := b.call + ".ToProvider"

	b.setTarget(call, "ToProvider")

	if provider == nil {
		panic(invalid(call, "nil provider"))
	}

	typ := reflect.TypeOf(provider)
	if typ.Kind() != reflect.Func {
		panic(invalid(call, "provider must be a function, got %s", qualified(typ)))
	}

	if reflect.ValueOf(provider).IsNil() {
		panic(invalid(call, "nil provider"))
	}

	if typ.NumOut() < providerMinResults || typ.NumOut() > providerMaxResults {
		panic(invalid(call, "provider must return 1 or 2 values, got %d", typ.NumOut()))
	}

	key := keyType[T]()

	result := typ.Out(0)
	if !result.AssignableTo(key) && !result.AssignableTo(reflect.PointerTo(key)) {
		panic(invalid(call, "provider returns %s, not assignable to %s", qualified(result), qualified(key)))
	}

	if typ.NumOut() == providerMaxResults && !typ.Out(providerErrorIndex).Implements(errorType) {
		panic(invalid(call, "second return value must be error, got %s",
			qualified(typ.Out(providerErrorIndex))))
	}

	binding.ToProvider(provider)

	return b
}

// AnnotatedWith sets the binding's annotation. Repeating the same value is a no-op; a different
// value panics.
func (b *Binding[T]) AnnotatedWith(annotation string) *Binding[T] {
	binding := b.must("AnnotatedWith")

	if b.annotation != nil {
		if *b.annotation == annotation {
			return b
		}

		panic(invalid(b.call+".AnnotatedWith("+strconv.Quote(annotation)+")",
			"annotation already set to %s", strconv.Quote(*b.annotation)))
	}

	b.annotation = &annotation
	b.call += "(" + strconv.Quote(annotation) + ")"

	binding.AnnotatedWith(annotation)

	return b
}

// In sets the binding's scope. A nil scope panics; setting the same scope twice is a no-op.
func (b *Binding[T]) In(scope Scope) *Binding[T] {
	binding := b.must("In")

	if scope == nil {
		panic(invalid(b.call+".In", "nil scope"))
	}

	b.setScope(b.call+".In("+qualified(reflect.TypeOf(scope))+")", scope)
	binding.In(scope)

	return b
}

// AsEagerSingleton scopes the binding to Singleton and builds it during InitModules. It panics
// when a different scope is already set, and composes with a preceding In(Singleton).
func (b *Binding[T]) AsEagerSingleton() *Binding[T] {
	binding := b.must("AsEagerSingleton")

	b.setScope(b.call+".AsEagerSingleton", Singleton)
	binding.AsEagerSingleton()

	return b
}
```

Add the engine binding alias at the top of `v2/binding.go`'s import block region, so the file names no v0 identifier below its declarations:

```go
// v0Binding is the engine's binding builder, which every method here forwards to. The engine's
// own checks stay in place behind v2's, so a gap in a v2 check still fails the way v0 did.
type v0Binding = v0.Binding
```

with `v0 "flamingo.me/dingo"` added to the imports.

`v2/resolve.go`:

```go
package dingo

import (
	"fmt"
	"reflect"
)

// GetInstance resolves T. A pointer-to-interface T returns an error wrapping
// ErrPointerToInterface; a pointer-to-pointer T returns an error wrapping ErrInvalidBinding
// rather than resolving one level too few, which panics inside the engine.
func (injector *Injector) GetInstance[T any]() (T, error) {
	return resolve[T](injector, "GetInstance", "")
}

// GetAnnotatedInstance resolves T carrying annotatedWith. The prefix "map:" selects one entry of
// a map binding, so annotations starting with it are reserved.
func (injector *Injector) GetAnnotatedInstance[T any](annotatedWith string) (T, error) {
	return resolve[T](injector, "GetAnnotatedInstance", annotatedWith)
}

func resolve[T any](injector *Injector, entry, annotation string) (T, error) {
	var zero T

	call := callOf[T](entry)
	engine := injector.mustRoot("Injector." + call)

	typ := reflect.TypeFor[T]()
	if typ.Kind() == reflect.Pointer {
		if typ.Elem().Kind() == reflect.Interface {
			return zero, fmt.Errorf("dingo: %s: %w", call, ErrPointerToInterface)
		}

		if typ.Elem().Kind() == reflect.Pointer {
			return zero, invalid(call, "pointer to pointer is not allowed")
		}
	}

	value, err := engine.GetAnnotatedInstance(typ, annotation)
	if err != nil {
		return zero, err //nolint:wrapcheck // the engine's error is returned verbatim so its v0 wording is preserved
	}

	return adapt[T](value)
}

// adapt converts what the engine returned into T.
//
// The engine keys on T with one pointer level removed and hands back whatever the binding
// produced, so a T-typed caller can receive a *T (the usual construction path) or, for a provider
// returning a value, a T where a *T was asked for. Dereference while the value is a pointer that
// does not fit T, then take the address of a copy when T is a pointer and a value arrived.
func adapt[T any](value any) (T, error) {
	var zero T

	if typed, ok := value.(T); ok {
		return typed, nil
	}

	if value == nil {
		// an interface-typed T whose provider returned nil: the zero T is that same nil, which is
		// what v0 handed back
		return zero, nil
	}

	target := reflect.TypeFor[T]()
	current := reflect.ValueOf(value)

	for current.Kind() == reflect.Pointer && !current.Type().AssignableTo(target) {
		if current.IsNil() {
			return zero, fmt.Errorf("dingo: %s resolved to a nil %s",
				typeName[T](), qualified(current.Type()))
		}

		current = current.Elem()
	}

	if target.Kind() == reflect.Pointer && current.Type().AssignableTo(target.Elem()) {
		addressable := reflect.New(target.Elem())
		addressable.Elem().Set(current)
		current = addressable
	}

	typed, ok := current.Interface().(T)
	if !ok {
		return zero, fmt.Errorf("dingo: resolved %s is not assignable to %s",
			qualified(reflect.TypeOf(value)), typeName[T]())
	}

	return typed, nil
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./v2/ -v`
Expected: PASS. `TestBind_RejectsIllegalTypeParameters` runs 16 subtests, `TestInjector_ZeroValuePanics` two per entry in its map.

- [ ] **Step 5: Vet, format, lint, race**

Run: `go vet ./... ./v2/... && gofmt -l . && CGO_ENABLED=1 go test -shuffle=on -race ./... ./v2/... && cd v2 && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./...`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add v2/bind.go v2/binding.go v2/resolve.go v2/fixtures_test.go v2/bind_key_test.go v2/helpers_test.go
git commit -m "feat(v2): add the generic binding builder, key validation and resolution"
```

### Task B4: the target rules

**Files:**
- Create: `v2/bind_target_test.go`
- Modify: `v2/fixtures_test.go` (add `routeModule`, used by B-41)

**Interfaces:**
- Consumes: `Binding[T]`'s seven methods (B3). No production code changes; if a row fails, the bug is in `v2/binding.go` and is fixed there.

- [ ] **Step 1: Write the test file**

Add to `v2/fixtures_test.go`:

```go
// routeModule is the shape Flamingo's web.BindRoutes binds: a value whose dynamic type is the
// target, with inject fields that must still be filled (B-41).
type routeModule struct {
	Greeter greeter `inject:""`
	built   bool
}

func (r *routeModule) Routes() string { return r.Greeter.Greet() }
```

`v2/bind_target_test.go`:

```go
package dingo_test

import (
	"errors"
	"reflect"
	"testing"

	dingo "flamingo.me/dingo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBindingTo_RequiresAssignability
// Covers B-05, B-06.
// Catches: an assignability check that forgets *U, which would reject every pointer-receiver
// implementation, the most common shape in the ecosystem.
func TestBindingTo_RequiresAssignability(t *testing.T) {
	t.Parallel()

	t.Run("a value-receiver implementation is accepted", func(t *testing.T) {
		t.Parallel()

		require.NoError(t, bindErr(func(i *dingo.Injector) { i.Bind[greeter]().To[politeGreeter]() }))
	})

	t.Run("a pointer-receiver-only implementation is accepted through *U", func(t *testing.T) {
		t.Parallel()

		require.NoError(t, bindErr(func(i *dingo.Injector) { i.Bind[greeter]().To[ptrOnlyGreeter]() }))
	})

	t.Run("an unrelated concrete type is rejected", func(t *testing.T) {
		t.Parallel()

		requireInvalidBinding(t, bindErr(func(i *dingo.Injector) { i.Bind[greeter]().To[unrelated]() }),
			"Bind[flamingo.me/dingo/v2_test.greeter].To[flamingo.me/dingo/v2_test.unrelated]",
			"is not assignable to")
	})
}

// TestBindingTo_RejectsSelfBinding
// Covers B-07.
// v0: reported "circular from X to X" at resolution time, far from the module that caused it.
// Catches: a self-binding accepted at bind time, which turns a typo into a runtime failure in an
// unrelated code path.
func TestBindingTo_RejectsSelfBinding(t *testing.T) {
	t.Parallel()

	requireInvalidBinding(t, bindErr(func(i *dingo.Injector) { i.Bind[counter]().To[counter]() }),
		"Bind[flamingo.me/dingo/v2_test.counter].To[flamingo.me/dingo/v2_test.counter]", "binding to itself")
}

// TestBindingTo_RejectsPointerToInterfaceTarget
// Covers B-08.
// Catches: stripping pointers before the interface check, which would silently accept To[*Iface]
// and bind to the interface it points at.
func TestBindingTo_RejectsPointerToInterfaceTarget(t *testing.T) {
	t.Parallel()

	type otherGreeter interface{ greeter }

	requireInvalidBinding(t, bindErr(func(i *dingo.Injector) { i.Bind[greeter]().To[*otherGreeter]() }),
		"pointer to interface")
}

// TestBindingTo_ChainsThroughAnotherInterfaceBinding
// Covers B-09.
// Catches: an over-eager pointer-to-interface check that also rejects a legal interface target,
// which would break every two-hop binding.
func TestBindingTo_ChainsThroughAnotherInterfaceBinding(t *testing.T) {
	t.Parallel()

	type frontGreeter interface{ Greet() string }

	injector := newInjector(t, dingo.ModuleFunc(func(i *dingo.Injector) {
		i.Bind[frontGreeter]().To[greeter]()
		i.Bind[greeter]().To[politeGreeter]()
	}))
	assert.Equal(t, "hello", get[frontGreeter](t, injector).Greet())
}

// TestBindingToType_BindsARuntimeChosenTargetLikeTo
// Covers B-41.
// Catches: ToType implemented over ToInstance, which returns the value verbatim with no
// injection, so every Flamingo route module would silently lose its injected fields.
func TestBindingToType_BindsARuntimeChosenTargetLikeTo(t *testing.T) {
	t.Parallel()

	original := &routeModule{built: true}

	injector := newInjector(t, dingo.ModuleFunc(func(i *dingo.Injector) {
		i.Bind[greeter]().To[politeGreeter]()
		i.Bind[routeModule]().ToType(reflect.TypeOf(original))
	}))

	resolved := get[*routeModule](t, injector)
	assert.NotSame(t, original, resolved, "the target is constructed, not handed over")
	assert.False(t, resolved.built)
	require.NotNil(t, resolved.Greeter, "the constructed target is injected")
	assert.Equal(t, "hello", resolved.Routes())
}

// TestBindingToType_AppliesTheToRules
// Covers B-42.
// Catches: ToType skipping the checks To[U] makes, which would let a runtime-typed helper bind
// anything at all.
func TestBindingToType_AppliesTheToRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		configure func(*dingo.Injector)
		wantParts []string // empty: the binding must be accepted
	}{{
		name:      "a nil type is rejected",
		configure: func(i *dingo.Injector) { i.Bind[greeter]().ToType(nil) },
		wantParts: []string{"Bind[flamingo.me/dingo/v2_test.greeter].ToType", "nil type"},
	}, {
		name:      "a non-assignable type is rejected and named from the reflect.Type",
		configure: func(i *dingo.Injector) { i.Bind[greeter]().ToType(reflect.TypeFor[unrelated]()) },
		wantParts: []string{"ToType(flamingo.me/dingo/v2_test.unrelated)", "is not assignable to"},
	}, {
		name:      "the key itself is rejected",
		configure: func(i *dingo.Injector) { i.Bind[counter]().ToType(reflect.TypeFor[counter]()) },
		wantParts: []string{"binding to itself"},
	}, {
		name:      "a pointer to an interface is rejected",
		configure: func(i *dingo.Injector) { i.Bind[greeter]().ToType(reflect.TypeFor[*greeter]()) },
		wantParts: []string{"pointer to interface"},
	}, {
		name:      "a pointer to an implementation is accepted",
		configure: func(i *dingo.Injector) { i.Bind[greeter]().ToType(reflect.TypeFor[*politeGreeter]()) },
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := bindErr(tt.configure)
			if len(tt.wantParts) == 0 {
				require.NoError(t, err)

				return
			}

			requireInvalidBinding(t, err, tt.wantParts...)
		})
	}
}

// TestBindingToInstance_RejectsNilInterfaceValue
// Covers B-10.
// v0: crashed with a nil-pointer dereference inside reflect.
// Catches: a missing nil check turning a common mistake into a stack trace with no call site.
func TestBindingToInstance_RejectsNilInterfaceValue(t *testing.T) {
	t.Parallel()

	requireInvalidBinding(t, bindErr(func(i *dingo.Injector) { i.Bind[greeter]().ToInstance(nil) }),
		"Bind[flamingo.me/dingo/v2_test.greeter].ToInstance", "nil instance")
}

// TestBindingToInstance_AcceptsTypedNilPointer
// Covers B-11.
// Catches: a nil check that also rejects a typed nil pointer, which is a legal instance: the
// interface value is not nil, it holds a nil pointer, and resolution returns exactly that.
func TestBindingToInstance_AcceptsTypedNilPointer(t *testing.T) {
	t.Parallel()

	injector := newInjector(t, dingo.ModuleFunc(func(i *dingo.Injector) {
		i.Bind[greeter]().ToInstance((*ptrOnlyGreeter)(nil))
	}))

	resolved := get[greeter](t, injector)
	assert.NotNil(t, resolved, "the interface value is not nil")

	typed, ok := resolved.(*ptrOnlyGreeter)
	require.True(t, ok)
	assert.Nil(t, typed, "it holds a nil pointer")
}

// TestBindingToProvider_ValidatesShape
// Covers B-13, B-14, B-15, B-16, B-17, B-18, B-19, B-44.
// v0: accepted three results and read only the first; accepted a non-error second result and
// ignored it; panicked inside reflect for zero results and for a nil function.
// Catches: a check that reads Out(0) before counting the results, which panics inside reflect for
// a zero-result provider instead of naming the problem.
func TestBindingToProvider_ValidatesShape(t *testing.T) {
	t.Parallel()

	var nilProvider func() greeter

	tests := []struct {
		name      string
		configure func(*dingo.Injector)
		wantParts []string
	}{{
		name:      "a non-function value is rejected",
		configure: func(i *dingo.Injector) { i.Bind[greeter]().ToProvider("not a function") },
		wantParts: []string{"ToProvider", "provider must be a function, got string"},
	}, {
		name:      "an untyped nil is rejected",
		configure: func(i *dingo.Injector) { i.Bind[greeter]().ToProvider(nil) },
		wantParts: []string{"ToProvider", "nil provider"},
	}, {
		name:      "a nil function value is rejected",
		configure: func(i *dingo.Injector) { i.Bind[greeter]().ToProvider(nilProvider) },
		wantParts: []string{"ToProvider", "nil provider"},
	}, {
		name:      "zero results are rejected",
		configure: func(i *dingo.Injector) { i.Bind[greeter]().ToProvider(func() {}) },
		wantParts: []string{"provider must return 1 or 2 values, got 0"},
	}, {
		name: "three results are rejected",
		configure: func(i *dingo.Injector) {
			i.Bind[greeter]().ToProvider(func() (greeter, error, int) { return nil, nil, 0 }) //nolint:revive // the point of the case
		},
		wantParts: []string{"provider must return 1 or 2 values, got 3"},
	}, {
		name:      "one result assignable to the key is accepted",
		configure: func(i *dingo.Injector) { i.Bind[greeter]().ToProvider(func() greeter { return politeGreeter{} }) },
	}, {
		name:      "one result not assignable to the key is rejected",
		configure: func(i *dingo.Injector) { i.Bind[greeter]().ToProvider(func() unrelated { return unrelated{} }) },
		wantParts: []string{"provider returns flamingo.me/dingo/v2_test.unrelated", "not assignable to"},
	}, {
		name:      "one result assignable only through *U is accepted",
		configure: func(i *dingo.Injector) { i.Bind[counter]().ToProvider(func() *counter { return &counter{} }) },
	}, {
		name: "a second result that is not an error is rejected",
		configure: func(i *dingo.Injector) {
			i.Bind[greeter]().ToProvider(func() (greeter, string) { return politeGreeter{}, "" })
		},
		wantParts: []string{"second return value must be error, got string"},
	}, {
		name: "a second result of type error is accepted",
		configure: func(i *dingo.Injector) {
			i.Bind[greeter]().ToProvider(func() (greeter, error) { return politeGreeter{}, nil })
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := bindErr(tt.configure)
			if len(tt.wantParts) == 0 {
				require.NoError(t, err)

				return
			}

			requireInvalidBinding(t, err, tt.wantParts...)
		})
	}
}

// TestBindingToProvider_ArgumentsAreInjectedFreely
// Covers B-20.
// Catches: a bind-time check on the argument list, which would reject providers whose arguments
// are bound by another module that has not configured yet.
func TestBindingToProvider_ArgumentsAreInjectedFreely(t *testing.T) {
	t.Parallel()

	require.NoError(t, bindErr(func(i *dingo.Injector) {
		i.Bind[greeter]().ToProvider(func(_ label, _ *counter, _ greeter) greeter { return politeGreeter{} })
	}))
}

// TestBinding_RejectsASecondTarget
// Covers B-21, B-43.
// v0: an instance silently won over a provider, which silently won over a To type, so the last
// two calls in a chain were dead code the author never learned about.
// Catches: a one-target rule that only guards one method, or that reports the second call instead
// of the first, which hides which module set the target that actually wins.
func TestBinding_RejectsASecondTarget(t *testing.T) {
	t.Parallel()

	targets := map[string]struct {
		apply func(*dingo.Binding[greeter])
		named string
	}{
		"To":         {apply: func(b *dingo.Binding[greeter]) { b.To[politeGreeter]() }, named: "To[flamingo.me/dingo/v2_test.politeGreeter]"},
		"ToType":     {apply: func(b *dingo.Binding[greeter]) { b.ToType(reflect.TypeFor[loudGreeter]()) }, named: "ToType(flamingo.me/dingo/v2_test.loudGreeter)"},
		"ToInstance": {apply: func(b *dingo.Binding[greeter]) { b.ToInstance(politeGreeter{}) }, named: "ToInstance"},
		"ToProvider": {apply: func(b *dingo.Binding[greeter]) { b.ToProvider(func() greeter { return politeGreeter{} }) }, named: "ToProvider"},
	}

	for firstName, first := range targets {
		for secondName, second := range targets {
			t.Run(firstName+" then "+secondName, func(t *testing.T) {
				t.Parallel()

				err := bindErr(func(i *dingo.Injector) {
					binding := i.Bind[greeter]()
					first.apply(binding)
					second.apply(binding)
				})
				requireInvalidBinding(t, err, "."+secondName, "target already set by "+first.named)
			})
		}
	}
}

// TestBinding_TargetlessBindingStaysLegal
// Covers B-22.
// Catches: a builder that requires a target, which would break Bind[T]().In(scope) — how a
// concrete type gets a scope — and Bind[T]().AsEagerSingleton().
func TestBinding_TargetlessBindingStaysLegal(t *testing.T) {
	t.Parallel()

	injector := newInjector(t, dingo.ModuleFunc(func(i *dingo.Injector) {
		i.Bind[counter]().In(dingo.Singleton)
	}))
	assert.Same(t, get[*counter](t, injector), get[*counter](t, injector))
}

var _ = errors.Is // keep the import when a row is temporarily commented out
```

Drop the trailing `var _ = errors.Is` line and the `errors` import if nothing else in the file uses them.

- [ ] **Step 2: Run the tests**

Run: `go test ./v2/ -run 'TestBinding' -v`
Expected: PASS. `TestBinding_RejectsASecondTarget` runs 16 subtests (4×4, same method twice included).

- [ ] **Step 3: Lint and commit**

```bash
cd v2 && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./... && cd ..
git add v2/bind_target_test.go v2/fixtures_test.go
git commit -m "test(v2): pin the binding target rules"
```

### Task B5: the attribute rules

**Files:**
- Create: `v2/bind_attribute_test.go`

**Interfaces:**
- Consumes: `AnnotatedWith`, `In`, `AsEagerSingleton` (B3). No production code changes.

- [ ] **Step 1: Write the test file**

`v2/bind_attribute_test.go`, one test per catalogue row:

```go
package dingo_test

import (
	"testing"

	dingo "flamingo.me/dingo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBindingAnnotatedWith_SameValueTwiceIsANoOp
// Covers B-23.
// Catches: a set-once rule implemented as "set at most once", which would reject the legal
// Override[T](a) followed by AnnotatedWith(a).
func TestBindingAnnotatedWith_SameValueTwiceIsANoOp(t *testing.T) {
	t.Parallel()

	require.NoError(t, bindErr(func(i *dingo.Injector) {
		i.Bind[greeter]().AnnotatedWith("a").AnnotatedWith("a").To[politeGreeter]()
	}))
}

// TestBindingAnnotatedWith_DifferentValuePanics
// Covers B-24.
// Catches: a second annotation silently winning, which moves a binding to a key nothing reads.
func TestBindingAnnotatedWith_DifferentValuePanics(t *testing.T) {
	t.Parallel()

	requireInvalidBinding(t, bindErr(func(i *dingo.Injector) {
		i.Bind[greeter]().AnnotatedWith("a").AnnotatedWith("b")
	}), `AnnotatedWith("b")`, `annotation already set to "a"`)
}

// TestBindingIn_RejectsNilScope
// Covers B-25.
// Catches: a nil scope stored and then dereferenced at resolution time.
func TestBindingIn_RejectsNilScope(t *testing.T) {
	t.Parallel()

	requireInvalidBinding(t, bindErr(func(i *dingo.Injector) { i.Bind[counter]().In(nil) }),
		"Bind[flamingo.me/dingo/v2_test.counter].In", "nil scope")
}

// TestBindingIn_SameScopeTwiceIsANoOpDifferentScopePanics
// Covers B-26.
// Catches: a second scope silently replacing the first, which changes an instance's lifetime
// without telling anyone.
func TestBindingIn_SameScopeTwiceIsANoOpDifferentScopePanics(t *testing.T) {
	t.Parallel()

	t.Run("the same scope twice is a no-op", func(t *testing.T) {
		t.Parallel()

		require.NoError(t, bindErr(func(i *dingo.Injector) {
			i.Bind[counter]().In(dingo.Singleton).In(dingo.Singleton)
		}))
	})

	t.Run("a different scope panics", func(t *testing.T) {
		t.Parallel()

		requireInvalidBinding(t, bindErr(func(i *dingo.Injector) {
			i.Bind[counter]().In(dingo.Singleton).In(dingo.ChildSingleton)
		}), "In(", "ChildSingletonScope", "scope already set to", "SingletonScope")
	})
}

// TestBindingAsEagerSingleton_PanicsOnConflictingScope
// Covers B-27.
// v0: AsEagerSingleton silently overwrote the scope that was already set.
// Catches: a ChildSingleton binding quietly promoted to a process-wide Singleton, which shares
// one instance across every child that was meant to have its own.
func TestBindingAsEagerSingleton_PanicsOnConflictingScope(t *testing.T) {
	t.Parallel()

	requireInvalidBinding(t, bindErr(func(i *dingo.Injector) {
		i.Bind[counter]().In(dingo.ChildSingleton).AsEagerSingleton()
	}), "AsEagerSingleton", "scope already set to", "ChildSingletonScope")
}

// TestBindingAsEagerSingleton_CompatibleWithPrecedingSingletonScope
// Covers B-28.
// Catches: a conflict check that compares scope values instead of scope types, which would reject
// the legal In(Singleton).AsEagerSingleton() pair.
func TestBindingAsEagerSingleton_CompatibleWithPrecedingSingletonScope(t *testing.T) {
	t.Parallel()

	require.NoError(t, bindErr(func(i *dingo.Injector) {
		i.Bind[counter]().In(dingo.Singleton).AsEagerSingleton()
	}))
}

// TestBinding_MethodOrderIsFree
// Covers B-29.
// Catches: a builder with a hidden required order, which would break the many call sites that
// write .In(s).To[U]().
func TestBinding_MethodOrderIsFree(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		configure func(*dingo.Injector)
		check     func(*testing.T, *dingo.Injector)
	}{{
		name:      "In before To",
		configure: func(i *dingo.Injector) { i.Bind[greeter]().In(dingo.Singleton).To[politeGreeter]() },
		check: func(t *testing.T, injector *dingo.Injector) {
			t.Helper()

			assert.Equal(t, "hello", get[greeter](t, injector).Greet())
		},
	}, {
		name:      "To before In",
		configure: func(i *dingo.Injector) { i.Bind[greeter]().To[politeGreeter]().In(dingo.Singleton) },
		check: func(t *testing.T, injector *dingo.Injector) {
			t.Helper()

			assert.Equal(t, "hello", get[greeter](t, injector).Greet())
		},
	}, {
		name: "AnnotatedWith before ToInstance",
		configure: func(i *dingo.Injector) {
			i.Bind[label]().AnnotatedWith("a").ToInstance(label("x"))
		},
		check: func(t *testing.T, injector *dingo.Injector) {
			t.Helper()

			value, err := injector.GetAnnotatedInstance[label]("a")
			require.NoError(t, err)
			assert.Equal(t, label("x"), value)
		},
	}, {
		name: "ToInstance before AnnotatedWith",
		configure: func(i *dingo.Injector) {
			i.Bind[label]().ToInstance(label("x")).AnnotatedWith("a")
		},
		check: func(t *testing.T, injector *dingo.Injector) {
			t.Helper()

			value, err := injector.GetAnnotatedInstance[label]("a")
			require.NoError(t, err)
			assert.Equal(t, label("x"), value)
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.check(t, newInjector(t, dingo.ModuleFunc(tt.configure)))
		})
	}
}

// TestBindingOverride_PresetsAnnotation
// Covers B-30.
// Catches: Override forgetting to record the annotation it already applied, which would make a
// following AnnotatedWith with the same value look like a conflict.
func TestBindingOverride_PresetsAnnotation(t *testing.T) {
	t.Parallel()

	require.NoError(t, bindErr(func(i *dingo.Injector) {
		i.Override[greeter]("a").AnnotatedWith("a").To[politeGreeter]()
	}))

	requireInvalidBinding(t, bindErr(func(i *dingo.Injector) {
		i.Override[greeter]("a").AnnotatedWith("b")
	}), `annotation already set to "a"`)
}
```

- [ ] **Step 2: Run, lint, commit**

```bash
go test ./v2/ -run 'TestBinding' -v
cd v2 && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./... && cd ..
git add v2/bind_attribute_test.go
git commit -m "test(v2): pin the annotation, scope and eager-singleton rules"
```

### Task B6: the panic path and the message shape

**Files:**
- Create: `v2/bind_errors_test.go`

**Interfaces:**
- Consumes: `bindPanic`, `requireInvalidBinding` (B2, B3). No production code changes.

Without this task a dropped `dingo: ` prefix or a lost type name would surface as thirty scattered failures instead of one.

- [ ] **Step 1: Write the test file**

`v2/bind_errors_test.go`:

```go
package dingo_test

import (
	"errors"
	"regexp"
	"testing"

	dingo "flamingo.me/dingo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBind_PanicsWithAnErrorWrappingErrInvalidBinding
// Covers B-37.
// Catches: a check panicking with a string, which TryModule would render as
// "dingo.TryModule panic: ..." and which errors.Is could never match.
func TestBind_PanicsWithAnErrorWrappingErrInvalidBinding(t *testing.T) {
	t.Parallel()

	err := bindPanic(t, func(i *dingo.Injector) { i.Bind[greeter]().To[unrelated]() })
	require.ErrorIs(t, err, dingo.ErrInvalidBinding)
}

// TestTryModule_ConvertsBindPanicsToErrors
// Covers B-37.
// Catches: a bind-time panic escaping TryModule, which would make module tests crash instead of
// failing with a message.
func TestTryModule_ConvertsBindPanicsToErrors(t *testing.T) {
	t.Parallel()

	err := dingo.TryModule(dingo.ModuleFunc(func(i *dingo.Injector) { i.Bind[greeter]().To[unrelated]() }))
	require.ErrorIs(t, err, dingo.ErrInvalidBinding)
	assert.True(t, errors.Is(err, dingo.ErrInvalidBinding))
}

// TestBindMessages_HaveTheDocumentedShape
// Covers B-38.
// Catches: a message that drops the prefix, the call or a type name, which is the difference
// between "fix line 42 of this module" and "something is wrong somewhere".
func TestBindMessages_HaveTheDocumentedShape(t *testing.T) {
	t.Parallel()

	shape := regexp.MustCompile(`^dingo: [^:]+: .+$`)

	tests := []struct {
		name      string
		configure func(*dingo.Injector)
		wantCall  string
	}{
		{name: "Bind", configure: func(i *dingo.Injector) { i.Bind[*greeter]() }, wantCall: "Bind[*flamingo.me/dingo/v2_test.greeter]"},
		{name: "BindMulti", configure: func(i *dingo.Injector) { i.BindMulti[*greeter]() }, wantCall: "BindMulti[*flamingo.me/dingo/v2_test.greeter]"},
		{name: "BindMap carries its key", configure: func(i *dingo.Injector) { i.BindMap[*greeter]("k") }, wantCall: `BindMap[*flamingo.me/dingo/v2_test.greeter]("k")`},
		{name: "Override carries its annotation", configure: func(i *dingo.Injector) { i.Override[*greeter]("ann") }, wantCall: `Override[*flamingo.me/dingo/v2_test.greeter]("ann")`},
		{name: "To", configure: func(i *dingo.Injector) { i.Bind[greeter]().To[unrelated]() }, wantCall: "Bind[flamingo.me/dingo/v2_test.greeter].To[flamingo.me/dingo/v2_test.unrelated]"},
		{name: "ToType", configure: func(i *dingo.Injector) { i.Bind[greeter]().ToType(nil) }, wantCall: "Bind[flamingo.me/dingo/v2_test.greeter].ToType"},
		{name: "ToInstance", configure: func(i *dingo.Injector) { i.Bind[greeter]().ToInstance(nil) }, wantCall: "Bind[flamingo.me/dingo/v2_test.greeter].ToInstance"},
		{name: "ToProvider", configure: func(i *dingo.Injector) { i.Bind[greeter]().ToProvider(42) }, wantCall: "Bind[flamingo.me/dingo/v2_test.greeter].ToProvider"},
		{name: "AnnotatedWith", configure: func(i *dingo.Injector) { i.Bind[greeter]().AnnotatedWith("a").AnnotatedWith("b") }, wantCall: `Bind[flamingo.me/dingo/v2_test.greeter].AnnotatedWith("b")`},
		{name: "In", configure: func(i *dingo.Injector) { i.Bind[greeter]().In(nil) }, wantCall: "Bind[flamingo.me/dingo/v2_test.greeter].In"},
		{name: "AsEagerSingleton", configure: func(i *dingo.Injector) { i.Bind[greeter]().In(dingo.ChildSingleton).AsEagerSingleton() }, wantCall: "Bind[flamingo.me/dingo/v2_test.greeter].AsEagerSingleton"},
		{name: "a call after AnnotatedWith carries the annotation", configure: func(i *dingo.Injector) { i.Bind[greeter]().AnnotatedWith("a").To[unrelated]() }, wantCall: `Bind[flamingo.me/dingo/v2_test.greeter]("a").To[flamingo.me/dingo/v2_test.unrelated]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := bindErr(tt.configure)
			require.ErrorIs(t, err, dingo.ErrInvalidBinding)
			assert.Regexp(t, shape, err.Error())
			assert.Contains(t, err.Error(), "dingo: "+tt.wantCall+": ")
		})
	}

	t.Run("BindInterceptor", func(t *testing.T) {
		t.Parallel()

		err := bindErr(func(i *dingo.Injector) { i.BindInterceptor[counter, recordingInterceptor2]() })
		require.ErrorIs(t, err, dingo.ErrInvalidBinding)
		assert.Contains(t, err.Error(),
			"dingo: BindInterceptor[flamingo.me/dingo/v2_test.counter, flamingo.me/dingo/v2_test.recordingInterceptor2]: ")
	})

	t.Run("GetInstance", func(t *testing.T) {
		t.Parallel()

		_, err := newInjector(t).GetInstance[**counter]()
		require.ErrorIs(t, err, dingo.ErrInvalidBinding)
		assert.Contains(t, err.Error(), "dingo: GetInstance[**flamingo.me/dingo/v2_test.counter]: ")
	})
}
```

The `BindInterceptor` subtest depends on Task B14 and the `GetInstance` one on Task B7's guards. Both are written now and will fail; keep them commented out until those tasks land, and the task steps there say to re-enable them. If that is uncomfortable, run this task after B14 instead — nothing else depends on its order.

- [ ] **Step 2: Run, lint, commit**

```bash
go test ./v2/ -run 'TestBind_Panics|TestTryModule_Converts|TestBindMessages' -v
cd v2 && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./... && cd ..
git add v2/bind_errors_test.go
git commit -m "test(v2): pin the bind-time panic path and the message shape"
```

### Task B7: resolution and the `map:` lookup

**Files:**
- Create: `v2/resolution_test.go`

**Interfaces:**
- Consumes: `GetInstance[T]`, `GetAnnotatedInstance[T]`, `adapt[T]` (B3). No production code changes; the two request-side guards are already in `resolve`.

- [ ] **Step 1: Write the test file**

`v2/resolution_test.go`. Rows follow the spec's resolution table in order, so table and spec diff by eye. `T` cannot be table data, so the typed call lives in `check` and nothing else does:

```go
package dingo_test

import (
	"testing"

	dingo "flamingo.me/dingo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetInstance_AdaptsTheResultToT
// Covers R-01, R-02, R-03, R-04, R-05, R-06, G-01.
// Catches: a value-typed T handing out the bound pointer's target rather than a copy, so one
// caller's mutation silently reaches another's instance.
func TestGetInstance_AdaptsTheResultToT(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		configure func(*dingo.Injector)
		check     func(*testing.T, *dingo.Injector)
	}{{
		name:      "GetInstance[*Service] returns the resolver's pointer",
		configure: func(i *dingo.Injector) { i.Bind[counter]().ToInstance(&counter{n: 4}) },
		check: func(t *testing.T, injector *dingo.Injector) {
			t.Helper()

			assert.Equal(t, 4, get[*counter](t, injector).n)
		},
	}, {
		name:      "GetInstance[Service] returns a copy, not an alias",
		configure: func(i *dingo.Injector) { i.Bind[counter]().ToInstance(&counter{n: 4}) },
		check: func(t *testing.T, injector *dingo.Injector) {
			t.Helper()

			copied := get[counter](t, injector)
			copied.n = 99
			assert.Equal(t, 4, get[*counter](t, injector).n)
		},
	}, {
		name:      "GetInstance[Iface] returns the bound implementation",
		configure: func(i *dingo.Injector) { i.Bind[greeter]().To[politeGreeter]() },
		check: func(t *testing.T, injector *dingo.Injector) {
			t.Helper()

			assert.Equal(t, "hello", get[greeter](t, injector).Greet())
		},
	}, {
		name:      "GetInstance[string] returns the bound string",
		configure: func(i *dingo.Injector) { i.Bind[string]().ToInstance("x") },
		check: func(t *testing.T, injector *dingo.Injector) {
			t.Helper()

			assert.Equal(t, "x", get[string](t, injector))
		},
	}, {
		name: "GetInstance[[]X] returns the multibinding slice",
		configure: func(i *dingo.Injector) {
			i.BindMulti[greeter]().To[politeGreeter]()
			i.BindMulti[greeter]().To[loudGreeter]()
		},
		check: func(t *testing.T, injector *dingo.Injector) {
			t.Helper()

			assert.Len(t, get[[]greeter](t, injector), 2)
		},
	}, {
		name:      "GetInstance[map[string]X] returns the map binding",
		configure: func(i *dingo.Injector) { i.BindMap[greeter]("a").To[politeGreeter]() },
		check: func(t *testing.T, injector *dingo.Injector) {
			t.Helper()

			assert.Len(t, get[map[string]greeter](t, injector), 1)
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.check(t, newInjector(t, dingo.ModuleFunc(tt.configure)))
		})
	}
}
```

Then, as named top-level tests in the same file, one per remaining catalogue row. Each keeps the doc-comment shape above:

| Test function | Covers | Body |
|---|---|---|
| `TestGetInstance_ResolvesInstanceBindingsDirectly` | R-09 | bind `Bind[counter]().ToInstance(&counter{n: 1}).In(dingo.Singleton)`, resolve twice, `assert.Same`: the instance bypasses scope, provider and `to` logic |
| `TestGetInstance_FollowsToChains` | R-13, R-14 | subtest one: `Bind[greeter]().To[politeGreeter]()` resolves by construction plus field injection (assert an injected field is set); subtest two: a three-hop `ifaceA → ifaceB → ifaceC → politeGreeter` chain resolves to `"hello"` |
| `TestGetInstance_TargetlessConcreteTypeConstructsDirectly` | R-15 | `Bind[counter]()` with no target; `get[*counter]` succeeds |
| `TestGetInstance_UnboundInterfaceErrors` | R-16 | no binding at all; `injector.GetInstance[greeter]()` returns an error containing `can not instantiate interface`; assert with `require.ErrorContains`, never bare `assert.Error` |
| `TestGetAnnotatedInstance_ResolvesEachBindingShape` | G-02 | table with four accepted rows (`ToInstance`, `ToProvider`, `To`, target-less) each read back through `GetAnnotatedInstance[T]("a")`; one row asserting an unbound annotated request errors; one row binding the Flamingo config shape — `string`, `bool`, `float64`, a map type and a slice type under `"config:"`-prefixed annotations — and reading each back |
| `TestGetAnnotatedInstance_MapPrefixIsASpecialLookup` | G-03 | three subtests: (a) `BindMap[greeter]("k")` and no singular binding — `GetAnnotatedInstance[greeter]("map:k")` returns the entry; (b) a singular binding annotated exactly `"map:k"` **wins** over the map entry, because the exact-annotation match runs first; (c) the four-character annotation `"map:"` alone is not a map lookup and errors like any unbound annotation |
| `TestGetInstance_PointerToInterfaceReturnsWrappedError` | G-04 | `GetInstance[*greeter]()` returns an error with `require.ErrorIs(err, dingo.ErrPointerToInterface)`; also assert the same for `GetAnnotatedInstance[*greeter]("a")` |
| `TestGetInstance_PointerToPointerReturnsWrappedError` | G-05 | `GetInstance[**counter]()` and `GetAnnotatedInstance[**counter]("a")` both return an error wrapping `ErrInvalidBinding` and nothing is resolved. `Catches:` line: on v0 this request **panics** with `reflect: call of reflect.Value.Type on zero Value` inside `requestInjection`, so the check converts a crash into a message — see "Verified deviations", item 1 |
| `TestBind_DirectSliceBindingWinsOverMultibinding` | K-10 | `Bind[[]greeter]().ToInstance([]greeter{loudGreeter{}})` plus `BindMulti[greeter]().To[politeGreeter]()`; `get[[]greeter]` returns the direct binding's slice |
| `TestBind_DirectMapBindingWinsOverBindMap` | K-11 | the same shape for `map[string]greeter` |
| `TestBind_DirectProviderTypeBindingWinsOverGeneration` | K-12 | `Bind[greeterProvider]().ToInstance(greeterProvider(func() greeter { return loudGreeter{} }))` plus `Bind[greeter]().To[politeGreeter]()`; `get[greeterProvider](t, injector)()` returns the loud one, proving bindings are consulted before the `Provider`-suffix rule |

Add one row the engine survey turned up, which no catalogue ID covers because it is v2-only plumbing; give it its own test and no `Covers` line:

| `TestGetInstance_NilProviderResultYieldsTheZeroValue` | — | `Bind[greeter]().ToProvider(func() greeter { return nil })`; `GetInstance[greeter]()` returns `(nil, nil)`, matching what v0 handed back, rather than an adaptation error |

- [ ] **Step 2: Re-enable the `GetInstance` subtest in `bind_errors_test.go`**

Uncomment the `GetInstance` subtest of `TestBindMessages_HaveTheDocumentedShape` (Task B6) and confirm it passes.

- [ ] **Step 3: Run, lint, commit**

```bash
go test ./v2/ -run 'TestGet|TestBind_Direct|TestBindMessages' -v
cd v2 && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./... && cd ..
git add v2/resolution_test.go v2/bind_errors_test.go
git commit -m "test(v2): pin resolution, result adaptation and the map: lookup"
```

### Task B8: providers

**Files:**
- Create: `v2/provider_test.go`

**Interfaces:**
- Consumes: `ToProvider` (B3), the `…Provider` fixtures (B3). No production code changes.

- [ ] **Step 1: Write the test file**

`v2/provider_test.go`, one named top-level test per catalogue row, each with the doc-comment shape used above:

| Test function | Covers | Body |
|---|---|---|
| `TestGetInstance_ReturnsGeneratedProviderFunction` | R-07 | with only `Bind[greeter]().To[politeGreeter]()` bound, `get[greeterProvider](t, injector)()` returns a working greeter: the engine generates the implementation from the `Provider` suffix |
| `TestGetInstance_UnsuffixedFuncTypeErrors` | R-08 | `injector.GetInstance[plainGreeterFunc]()` errors; assert the message contains `Do you want a provider?` |
| `TestProvider_ResolvesWithoutArguments` | R-10 | `ToProvider(func() greeter { calls.Add(1); return politeGreeter{} })`; resolving calls it once |
| `TestProvider_ResolvesArgumentsViaInjection` | R-11 | `Bind[label]().ToInstance(label("x"))`, then `Bind[greeter]().ToProvider(func(l label) greeter {...})`; the provider sees `"x"` |
| `TestProvider_UnboundConcreteArgumentZeroConstructs` | R-11a | a provider taking an unbound `int` receives `0`, and one taking an unbound `*counter` receives a freshly constructed non-nil value. `Catches:` the assumption that an unbound argument is an error, which would make half the ecosystem's providers fail |
| `TestProvider_ErrorResultIsIgnoredAtResolution` | R-12 | `ToProvider(func() (greeter, error) { return politeGreeter{}, errBoom })`; `GetInstance[greeter]()` returns the greeter and a **nil** error. `(decision)` — unchanged from v0.4.1; propagating it is `feat/to-provider-error-handling` |
| `TestGeneratedProvider_NeedsOnlyTheUnderlyingTypeBound` | R-28 | a struct with a `greeterProvider`-typed `inject` field gets a working provider with no binding of its own |
| `TestGeneratedProvider_ForSliceRoutesToMultibinding` | R-29 | a `greetersProvider`-typed field returns every `BindMulti[greeter]` entry when called |
| `TestGeneratedProvider_ForMapRoutesToMapBinding` | R-30 | a `greeterMapProvider`-typed field returns every `BindMap[greeter]` entry when called |

Declare `var errBoom = errors.New("boom")` at file scope; `err113` is excluded for `_test.go`, but a package-level sentinel reads better than an inline `errors.New` anyway.

- [ ] **Step 2: Run, lint, commit**

```bash
go test ./v2/ -run 'TestProvider|TestGeneratedProvider|TestGetInstance_Returns|TestGetInstance_Unsuffixed' -v
cd v2 && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./... && cd ..
git add v2/provider_test.go
git commit -m "test(v2): pin provider resolution and generated Provider types"
```

### Task B9: injection

**Files:**
- Create: `v2/injection_test.go`

**Interfaces:**
- Consumes: `RequestInjection` (B2), `GetInstance` (B3). No production code changes.

- [ ] **Step 1: Write the test file**

`v2/injection_test.go`, one named top-level test per catalogue row:

| Test function | Covers | Body |
|---|---|---|
| `TestInjection_PointerFieldReceivesPointerValueFieldReceivesDereferencedValue` | R-17 | one struct with a `*counter` field and a `label` field; bind `Bind[label]().ToInstance(label("x"))`; the pointer field gets the constructed pointer, the value field gets `"x"`. The value field is of non-struct kind on purpose — see R-32 |
| `TestInjection_StructKindFieldIsAnError` | R-32 | an `inject`-tagged field of type `counter` (not `*counter`) fails with the engine's `can not inject into struct`, with and without a binding present. `(decision)` — the engine rejects struct-kind fields outright, so only `GetInstance[Service]()` ever hands out a struct copy |
| `TestInjection_EachInjectionIsAnIndependentCopy` | R-18 | two structs with a `label` field; mutating one does not affect the other |
| `TestInjection_AnnotatedFieldMatchesExactAnnotationString` | R-19 | `inject:"a"` matches `.AnnotatedWith("a")` and not `.AnnotatedWith("ab")` |
| `TestInjection_OptionalTagLeavesUnresolvableFieldAtZeroValue` | R-20 | two subtests, `inject:"tag,optional"` and `inject:"tag, optional"` (spaced); both leave the field at its zero value with no error |
| `TestInjection_InjectMethodRunsBeforeFieldInjection` | R-21 | a pointer-receiver `Inject(l label)` method is called with the resolved argument, and it runs before tagged fields are set (record the order) |
| `TestInjection_AnonymousAnnotatedStructIsResolvedAsAnArgument` | R-21a | an `Inject(cfg *struct{ V string \`inject:"config:k,optional"\` })` method **and** a `ToProvider` taking the same anonymous struct both receive it populated, with the optional field zero when the annotation is unbound. Flamingo's dominant config idiom; both entry paths in one test |
| `TestInjection_NonPointerReceiverInjectMethodErrors` | R-22 | a value-receiver `Inject` method reached through a value returns `ErrInvalidInjectReceiver`; assert with `require.ErrorIs` |
| `TestRequestInjection_DelayedDuringConfigureRunsAfterAllModules` | R-23 | module A calls `RequestInjection(obj)` in `Configure`; module B binds what `obj` needs afterwards; `obj` is injected once both have configured |
| `TestRequestInjection_AfterInitModulesInjectsImmediately` | R-33 | after `newInjector(t, ...)` returns, `injector.RequestInjection(obj)` fills `obj` before it returns |
| `TestRequestInjection_QueuedFailureReturnsFromInitModulesUnwrapped` | R-34 | a queued object with an unresolvable field makes `InitModules` return the bare injection error: `require.Error`, then `assert.False(t, errors.Is(err, dingo.ErrInitModules))`. `(decision)` — kept so that a caller matching `ErrInitModules` is not misled |
| `TestInjection_PointerToInterfaceFieldErrors` | R-24 | a `*greeter`-typed field errors with `require.ErrorIs(err, dingo.ErrPointerToInterface)`, binding present or not |
| `TestInjection_InjectorSelfBindingReturnsCurrentInjector` | R-31 | a struct with a `*dingo.Injector` field resolved in a child receives the **child**, not the root; `assert.Same` against `childOf`'s return |

- [ ] **Step 2: Run, lint, commit**

```bash
go test ./v2/ -run 'TestInjection|TestRequestInjection' -v
cd v2 && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./... && cd ..
git add v2/injection_test.go
git commit -m "test(v2): pin field injection, Inject methods and RequestInjection"
```

### Task B10: multibindings and map bindings

**Files:**
- Create: `v2/multibinding_test.go`

**Interfaces:**
- Consumes: `BindMulti`, `BindMap`, `collectionKeyOf` (B3). No production code changes: MB-15 and MB-16 are already rejected by `collectionKeyOf`.

- [ ] **Step 1: Write the test file**

`v2/multibinding_test.go`, one named top-level test per catalogue row:

| Test function | Covers | Body |
|---|---|---|
| `TestBindMulti_RegistersInOrder` | MB-01 | three `BindMulti[greeter]()` calls inject as `[]greeter` in registration order |
| `TestBindMulti_DuplicateImplementationsAreAllKept` | MB-02 | two `BindMulti[greeter]().To[politeGreeter]()` calls both survive; length 2, no error. `(decision)` — contrasts with singular duplicate detection |
| `TestBindMulti_AnnotationsProduceDisjointSlices` | MB-03 | entries under `"a"` and `"b"` do not mix |
| `TestBindMulti_EmptyResolvesToNonNilEmptySlice` | MB-04 | with no `BindMulti` at all, `get[[]greeter]` is non-nil and empty (`assert.NotNil` then `assert.Empty`) |
| `TestBindMulti_ChildMergesAfterParent` | MB-05 | a child's slice is parent entries first, then the child's |
| `TestBindMap_LastWriteWinsPerKey` | MB-06 | `BindMap[greeter]("k")` injects as `map[string]greeter` |
| `TestBindMap_DuplicateKeySilentlyOverwrites` | MB-07 | a second `BindMap` for the same key wins, no error. `(decision)` |
| `TestBindMap_AnnotationsProduceDisjointMaps` | MB-08 | as MB-03, for maps |
| `TestBindMap_EmptyResolvesToNonNilEmptyMap` | MB-09 | non-nil, empty |
| `TestBindMap_ChildOverridesParentKey` | MB-10 | a child's entry for a key wins; other parent keys are still inherited |
| `TestBindMulti_ProviderSuffixedElementsResolveIndependently` | MB-11 | a `[]greeterProvider` field gives each element its own generated provider bound to its own implementation |
| `TestBindMap_ProviderSuffixedValuesResolveIndependently` | MB-12 | the same for `map[string]greeterProvider` |
| `TestBindMulti_PerElementScopeIsIndependent` | MB-13 | a `BindMulti` entry and a `BindMap` entry bound `In(dingo.ChildSingleton)` each resolve through the scope per element; relies on `newInjector`'s second scope swap |
| `TestBindMulti_OverrideOfAMultibindingLeavesTheSliceAlone` | MB-14 | `Override[greeter]("a")` on a `BindMulti`-only `greeter` adds a singular binding and leaves the slice intact. `(decision)`; cross-ref B-32 |
| `TestBindMulti_RejectsProviderSuffixedElementType` | MB-15 | `BindMulti[greeterProvider]()` panics naming `BindMulti[flamingo.me/dingo/v2_test.greeter]` as the form to use. `(decision)` — v0 accepted it and the site resolved to an empty slice with no error |
| `TestBindMap_RejectsProviderSuffixedValueType` | MB-16 | `BindMap[greeterProvider]("k")` panics the same way, naming `BindMap[...greeter]` |
| `TestBindMap_FieldWithMapPrefixReceivesOneEntry` | MB-17 | a `greeter` field tagged `inject:"map:k"` receives the map entry `k`; with no such entry and no `optional` it errors |
| `TestBindMulti_TargetlessDirectSliceBindingScopesTheMultibinding` | MB-18 | `Bind[[]greeter]().In(dingo.Singleton)` with **no target** falls through to the `BindMulti` entries and caches the assembled slice: two resolutions return the same slice (`assert.Equal` on the slice header via `reflect.ValueOf(...).Pointer()`, or assert that a construction counter stayed at 1). `(decision)` — contrast with K-10, where a target makes the direct binding win |

- [ ] **Step 2: Run, lint, commit**

```bash
go test ./v2/ -run 'TestBindMulti|TestBindMap' -v
cd v2 && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./... && cd ..
git add v2/multibinding_test.go
git commit -m "test(v2): pin multibindings, map bindings and their Provider-type rejections"
```

### Task B11: scopes and eager singletons

**Files:**
- Create: `v2/scope_test.go`
- Modify: `v2/module_test.go` (add M-09)

**Interfaces:**
- Consumes: `BindScope`, `SetBuildEagerSingletons`, `BuildEagerSingletons`, `Singleton`, `ChildSingleton`, `countingScope` (B2, B3). No production code changes.

Identity is asserted with `assert.Same` for pointers and with a construction counter for values, never with `==` on `any`.

- [ ] **Step 1: Write the test file**

`v2/scope_test.go`:

| Test function | Covers | Body |
|---|---|---|
| `TestScope_IdentityMatrix` | S-01, S-02, S-03, S-08 | one table over `{Singleton, ChildSingleton, a registered countingScope, an unregistered countingScope}` × `{the same injector twice, two children of one parent, two roots, the same type under two annotations}`. Cells: `Singleton` caches across repeated resolution in one injector (S-01); `ChildSingleton` caches independently per child (S-02); a registered custom scope resolves while an unregistered one errors with `unknown scope` (S-03); the same type under two annotations does not collide in the singleton cache (S-08) |
| `TestBindScope_FreshSingletonScopeReplacesTheCache` | S-09 | resolve a `Singleton`-scoped binding, call `injector.BindScope(dingo.NewSingletonScope())`, resolve again: two instances. Repeat for `NewChildSingletonScope()`. `(decision)` — scopes are looked up by Go type; this is the contract `newInjector` and `childOf` rely on |
| `TestEagerSingleton_BuildsWithoutTouchingParent` | S-04 | `BuildEagerSingletons(false)` on a child builds only the child's eager bindings; observe a construction counter **before** any `GetInstance` call |
| `TestEagerSingleton_WithParentBuildsChildBeforeParent` | S-05 | the parent is created with `SetBuildEagerSingletons(false)` — otherwise `NewInjector` builds its eager singletons before the child exists and the order is unobservable; then `child.BuildEagerSingletons(true)` records child before parent |
| `TestEagerSingleton_ConstructionFailureReturnsFromNewInjector` | S-10 | a module with `Bind[X]().AsEagerSingleton()` whose construction fails on an unresolvable `inject` field: `dingo.NewInjector(module)` returns the error, while `dingo.TryModule(module)` returns nil. Both halves in one test. This is one of the three tests allowed to call `dingo.NewInjector` directly; say so in a comment |
| `TestSetBuildEagerSingletons_DisablesAutomaticBuild` | S-06 | with the flag off, nothing is constructed during `InitModules`; a later manual `BuildEagerSingletons(false)` builds them |
| `TestSingleton_ConcurrentResolutionConstructsExactlyOnce` | S-07 | `const concurrentResolvers = 100` goroutines resolving one `Singleton`-scoped binding; `sync.WaitGroup`, an `atomic.Int64` construction count asserted to be exactly 1. The type must have **no** interceptor: the engine's interceptor map is unsynchronized (I-07) and would add a second, unrelated race. No sleeps |

Add to `v2/module_test.go`:

| Test function | Covers | Body |
|---|---|---|
| `TestTryModule_DoesNotBuildEagerSingletons` | M-09 | `dingo.TryModule` on a native v2 module with an `AsEagerSingleton()` binding whose construction would fail returns nil. The v2-native twin of X-15; pairs with S-10 |

- [ ] **Step 2: Run, lint, commit**

```bash
CGO_ENABLED=1 go test -race ./v2/ -run 'TestScope|TestBindScope|TestEagerSingleton|TestSetBuild|TestSingleton_Concurrent|TestTryModule' -v
cd v2 && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./... && cd ..
git add v2/scope_test.go v2/module_test.go
git commit -m "test(v2): pin scope identity, eager singletons and concurrent construction"
```

### Task B12: child injectors

**Files:**
- Create: `v2/child_test.go`

**Interfaces:**
- Consumes: `Child`, `childOf` (B2). No production code changes. `BindInterceptor` (B14) is used by C-03 and I-06; run this task after B14, or write C-03 last.

- [ ] **Step 1: Write the test file**

`v2/child_test.go`. C-01, C-02 and C-05 form one matrix over `{binding kind} × {bound at} × {observed from}`: kind covers single, multi, map, interceptor, scope and the `*Injector` self-binding; bound-at covers parent only, child only and both; observed-from includes a sibling child.

| Test function | Covers | Body |
|---|---|---|
| `TestChild_FallsBackToParentBinding` | C-01 | a child with no binding for `greeter` resolves it from the parent. Two cases, because the inventory flags that the v0 test may have proved provider laziness rather than fallback: one with a plain `To` binding and one with a `ToProvider` binding |
| `TestChild_LocalBindingInvisibleToParentAndSiblings` | C-02 | a child-only binding is invisible to the parent and to a sibling child; assert the parent's resolution errors or yields the parent's own value, never the child's |
| `TestChild_LocalBindingShadowsParentBinding` | C-05 | the `bound at = both` column: a child binding wins for resolutions started at the child, for a singular binding, for a map-binding key (cross-ref MB-10) and for the `*dingo.Injector` self-binding; the parent and a sibling still see the parent's. What every Flamingo per-area override relies on |
| `TestChild_InterceptorsInheritedFromParentUnconditionally` | C-03 | a parent-registered interceptor wraps a child-resolved instance **outside** the child's own interceptors; a resolution started at the parent never sees the child's. Order asserted as a recorded `[]string`. `(decision)` — Guice's rule, kept |
| `TestChild_SingletonScopeIsSharedGloballyNotPerInjector` | C-04 | **the deliberate exception**: calls `dingo.NewInjector` twice with no scope swap, is **not** parallel, and uses a fixture type unique to this test so no parallel test can read its cached instance. Two independently-bound `Singleton`-scoped bindings for the same `(type, annotation)` on two roots collide in one cache. `(decision)` — a sharp edge; the comment must say that `newInjector` would hide exactly the thing under test |

C-04 needs its own `nolint` island and its own fixture:

```go
// singletonSharingFixture exists only for C-04. It is never used by another test, because C-04
// runs against the process-wide dingo.Singleton and a shared type would let a parallel test read
// its cached instance.
type singletonSharingFixture struct{ n int }

// TestChild_SingletonScopeIsSharedGloballyNotPerInjector
// Covers C-04.
// Catches: a reader assuming Singleton is per-injector. dingo.Singleton is one process-wide
// object that every root registers, so two unrelated injectors hand out one instance.
// (decision) Kept as is; a per-injector Singleton is a listed follow-up.
//
//nolint:paralleltest // resolves against the process-wide Singleton, which newInjector swaps away
func TestChild_SingletonScopeIsSharedGloballyNotPerInjector(t *testing.T) {
	module := dingo.ModuleFunc(func(i *dingo.Injector) {
		i.Bind[singletonSharingFixture]().In(dingo.Singleton)
	})

	first, err := dingo.NewInjector(module)
	require.NoError(t, err)

	second, err := dingo.NewInjector(module)
	require.NoError(t, err)

	assert.Same(t, get[*singletonSharingFixture](t, first), get[*singletonSharingFixture](t, second))
}
```

- [ ] **Step 2: Run, lint, commit**

```bash
CGO_ENABLED=1 go test -shuffle=on -race ./v2/ -run 'TestChild' -v
cd v2 && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./... && cd ..
git add v2/child_test.go
git commit -m "test(v2): pin child injector visibility, shadowing and the global Singleton"
```

### Task B13: overrides and duplicate detection

**Files:**
- Create: `v2/override_test.go`

**Interfaces:**
- Consumes: `Override[T]` (B3). No production code changes.

B-31 and B-32 keep their numbers from the earlier catalogue but pin the **opposite** of what it said: the engine never raises "unknown binding" for a key v2 lets through, because `Override` calls `Bind` first, so the key always has at least the override's own binding when the override loop runs.

- [ ] **Step 1: Write the test file**

`v2/override_test.go`:

| Test function | Covers | Body |
|---|---|---|
| `TestOverride_ReplacesEveryMatchingAnnotationEntry` | OV-01 | two prior bindings for `(greeter, "a")`, one `Override[greeter]("a")`: **every** matching entry is replaced, not just the first |
| `TestOverride_UnknownBindingSilentlyBecomesAPlainBinding` | B-31, OV-02 | `Override[greeter]("a")` for a `(greeter, "a")` nothing bound: `InitModules` returns no error and the key resolves as a plain annotated binding. `(decision)` — v0 quirk kept; the `cannot override unknown binding` branch is unreachable through v2, since it needs a pointer-typed key, which B-03 rejects |
| `TestOverride_OfAMultibindingAddsASingularBindingAndLeavesTheSliceAlone` | B-32, OV-03 | `Override[greeter]("a")` on a `BindMulti`-only `greeter`: no error, a singular `(greeter, "a")` binding appears, and `GetInstance[[]greeter]()` still returns every multibinding entry. `(decision)`; cross-ref MB-14 |
| `TestOverride_PresetsAnnotationWithoutDoubleCounting` | OV-04 | the override-created binding is not itself overridden again and does not trip duplicate detection: override evaluation runs before the duplicate check |
| `TestOverride_AfterInitModulesIsNeverEvaluated` | OV-05 | `injector.Override[greeter]("a").To[loudGreeter]()` **after** `InitModules` returned does not replace the existing binding; the earlier one still resolves and no error is reported. `(decision)` — the override loop runs only inside `InitModules`, so a late override is a dangling plain binding appended after the duplicate check |
| `TestDuplicateBinding_EqualBindingsAreTolerated` | DUP-01 | two structurally equal bindings for one `(type, annotation)` do not error |
| `TestDuplicateBinding_UnequalBindingsErrorAtInitModules` | DUP-02 | two subtests. One: two `To`-based bindings with different targets error at `InitModules` and the message names **both** target types — `To`-based on purpose, because the message interpolates only `binding.to`. Two: two `ToInstance` bindings with different instances error too, and only the key and annotation are asserted, because the message renders empty strings for the target |

- [ ] **Step 2: Run, lint, commit**

```bash
go test ./v2/ -run 'TestOverride|TestDuplicateBinding' -v
cd v2 && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./... && cd ..
git add v2/override_test.go
git commit -m "test(v2): pin override semantics and duplicate binding detection"
```

### Task B14: interceptors

**Files:**
- Create: `v2/interceptor.go`, `v2/interceptor_test.go`
- Modify: `v2/bind_key_test.go`, `v2/bind_errors_test.go` (re-enable the two `BindInterceptor` entries)

**Interfaces:**
- Consumes: `mustRoot`, `carrier`, `invalid` (B2, B3).
- Produces: `func (injector *Injector) BindInterceptor[T, I any]()`.

- [ ] **Step 1: Write the failing tests**

`v2/interceptor_test.go`:

| Test function | Covers | Body |
|---|---|---|
| `TestBindInterceptor_ValidatesShape` | B-33, B-34, B-34a, B-35, B-36 | one table: `T` a concrete type panics `is not an interface`; `T` a pointer to an interface panics the same way; `I` not a struct, and `I` a struct whose field 0 is not `T`, panic naming both types; `I` a struct whose field 0 has type `T` but is **unexported** panics at bind time naming both types (B-34a); `*I` not implementing `T` panics; and the accepted counterpart — `T` an interface, `I` a struct with an exported field 0 of type `T`, `*I` implementing `T` |
| `TestBindInterceptor_RejectedInterceptorRegistersNothing` | I-04 | after a rejected `BindInterceptor` recovered with `bindPanic`, resolving `T` on the **same** injector returns the bare bound implementation, unwrapped: v2's checks run before the engine is touched |
| `TestInterceptor_ChainAppliesInRegistrationOrder` | I-01 | two interceptors for one `T` apply in registration order, each wrapping the previous; asserted as a recorded `[]string` so a failure names which ran where |
| `TestInterceptor_Field0ReceivesTheBaseValuePositionally` | I-02 | a **named, non-embedded** field 0 still receives the base value; embedding would obscure a positional bug |
| `TestInterceptor_InjectTaggedFieldsAlsoResolve` | I-03 | an interceptor with both a field-0 target and a separate `inject`-tagged field gets both resolved |
| `TestInterceptor_ReWrapsOnEveryResolutionEvenWithASingletonBase` | I-05 | a `Singleton`-scoped base is constructed once; two resolutions give two wrappers (`assert.NotSame`) over one inner instance (`assert.Same` on field 0). `(decision)` — interception runs after the scope cache, so wrapper state does not survive a resolution |
| `TestInterceptor_AppliesAcrossChildBoundary` | I-06 | a parent interceptor fires on a child-resolved instance: child inner, parent outer. Cross-ref C-03, no separate mechanism |

B-34a needs a fixture with an unexported field 0, which exists **only** for that row; declare it next to the test with a comment saying the suite's other interceptor fixtures export field 0 on purpose:

```go
// unexportedFieldInterceptor exists only for B-34a. Every other interceptor fixture exports field
// 0, because the engine writes the intercepted value into it with reflect.Value.Set, which panics
// for a field obtained through an unexported name.
type unexportedFieldInterceptor struct{ wrapped greeter }

func (i *unexportedFieldInterceptor) Greet() string { return i.wrapped.Greet() }
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./v2/ -run 'TestBindInterceptor|TestInterceptor' -v`
Expected: compile failure, `BindInterceptor` undefined.

- [ ] **Step 3: Write the implementation**

`v2/interceptor.go`:

```go
package dingo

import "reflect"

// BindInterceptor makes I intercept every resolution of T.
//
// T must be an interface. I is used with its pointer levels removed and must be a struct whose
// first field is exported and has type T: resolution builds a fresh I with reflect.New, injects
// into it and writes the intercepted value into field 0 with reflect.Value.Set, which panics for
// a field obtained through an unexported name. *I must implement T.
//
// The wrapper is rebuilt on every resolution, after the scope cache, so an interceptor must not
// keep state it expects to survive between resolutions.
func (injector *Injector) BindInterceptor[T, I any]() {
	call := "BindInterceptor[" + typeName[T]() + ", " + typeName[I]() + "]"
	engine := injector.mustRoot("Injector." + call)

	target := reflect.TypeFor[T]()
	if target.Kind() != reflect.Interface {
		panic(invalid(call, "%s is not an interface", typeName[T]()))
	}

	interceptor := reflect.TypeFor[I]()
	for interceptor.Kind() == reflect.Pointer {
		interceptor = interceptor.Elem()
	}

	if interceptor.Kind() != reflect.Struct || interceptor.NumField() == 0 ||
		interceptor.Field(0).Type != target {
		panic(invalid(call, "%s must be a struct whose first field is %s",
			qualified(interceptor), typeName[T]()))
	}

	if !interceptor.Field(0).IsExported() {
		panic(invalid(call, "field 0 of %s must be exported", qualified(interceptor)))
	}

	if !reflect.PointerTo(interceptor).Implements(target) {
		panic(invalid(call, "*%s does not implement %s", qualified(interceptor), typeName[T]()))
	}

	// the engine stores reflect.TypeOf(interceptor) without stripping and resolution does
	// reflect.New(I).Elem().Field(0), so the stored type must be exactly I
	engine.BindInterceptor(carrier(target), reflect.New(interceptor).Elem().Interface())
}
```

- [ ] **Step 4: Re-enable the deferred entries**

Uncomment the `"BindInterceptor"` entry in `TestInjector_ZeroValuePanics` (`v2/bind_key_test.go`, Task B3) and the `BindInterceptor` subtest of `TestBindMessages_HaveTheDocumentedShape` (`v2/bind_errors_test.go`, Task B6).

- [ ] **Step 5: Run, lint, commit**

```bash
CGO_ENABLED=1 go test -race ./v2/ -v
cd v2 && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./... && cd ..
git add v2/interceptor.go v2/interceptor_test.go v2/bind_key_test.go v2/bind_errors_test.go
git commit -m "feat(v2): add BindInterceptor with bind-time shape validation"
```

### Task B15: the Inspector

**Files:**
- Create: `v2/inspect.go`, `v2/inspect_test.go`
- Modify: `v2/bind_key_test.go` (re-enable the `Inspect` entry)

**Interfaces:**
- Consumes: `mustRoot`, `attachedOf` (B2).
- Produces: `type Inspector struct{...}` and `func (injector *Injector) Inspect(inspector Inspector)`.

- [ ] **Step 1: Write the failing tests**

Add the `spyInspector` fixture to `v2/fixtures_test.go`. `Inspect` ranges over maps, so records are sorted before comparison:

```go
// spyInspector records Inspect callbacks into typed slices.
type spyInspector struct {
	bindings      []string
	multiBindings []string
	mapBindings   []string
	parents       []*dingo.Injector
}

func (s *spyInspector) inspector() dingo.Inspector {
	return dingo.Inspector{
		InspectBinding: func(of reflect.Type, annotation string, to reflect.Type, _, _ *reflect.Value, _ dingo.Scope) {
			s.bindings = append(s.bindings, record(of, annotation, to))
		},
		InspectMultiBinding: func(of reflect.Type, index int, annotation string, to reflect.Type, _, _ *reflect.Value, _ dingo.Scope) {
			s.multiBindings = append(s.multiBindings, record(of, annotation, to)+"#"+strconv.Itoa(index))
		},
		InspectMapBinding: func(of reflect.Type, key, annotation string, to reflect.Type, _, _ *reflect.Value, _ dingo.Scope) {
			s.mapBindings = append(s.mapBindings, record(of, annotation, to)+"@"+key)
		},
		InspectParent: func(parent *dingo.Injector) { s.parents = append(s.parents, parent) },
	}
}

func record(of reflect.Type, annotation string, to reflect.Type) string {
	name := of.String() + "|" + annotation
	if to != nil {
		name += "|" + to.String()
	}

	return name
}
```

`v2/inspect_test.go`:

| Test function | Covers | Body |
|---|---|---|
| `TestInspect_BindingCallbackFiresPerSingularBinding` | INS-01 | fires once per bound type with its target, provider/instance and scope; sort before comparing |
| `TestInspect_MultiBindingCallbackFiresPerEntryWithSharedIndexSpace` | INS-02 | fires per entry with its slice index, and the index space is **not** filtered by annotation: two entries under different annotations get indices 0 and 1. `(decision)` |
| `TestInspect_MapBindingCallbackFiresPerKey` | INS-03 | once per map-binding key |
| `TestInspect_ParentCallbackFiresOnlyWhenAParentExists` | INS-04 | fires once with the parent for an injector from `Child()`, never for one from `NewInjector`, and does not recurse into the grandparent on its own |
| `TestInspect_NilCallbacksAreSkipped` | INS-05 | an `Inspector` with only `InspectBinding` set does not panic and does not invoke the others. **This is the test that catches a typed injector installing no-op callbacks**, which would pass INS-01 to INS-04 and silently change the engine's behavior |
| `TestInspect_ProviderAndInstanceArgsAreNilWhenAbsent` | INS-06 | `provider` and `instance` are nil for a target-less and for a `To`-only binding, and non-nil for the matching binding kinds |

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./v2/ -run 'TestInspect' -v`
Expected: compile failure, `Inspector` and `Inspect` undefined.

- [ ] **Step 3: Write the implementation**

`v2/inspect.go`:

```go
package dingo

import (
	"reflect"

	v0 "flamingo.me/dingo"
)

// Inspector defines the callbacks called during injector inspection. Every field is optional; a
// nil callback is not called.
type Inspector struct {
	InspectBinding      func(of reflect.Type, annotation string, to reflect.Type, provider, instance *reflect.Value, in Scope)
	InspectMultiBinding func(of reflect.Type, index int, annotation string, to reflect.Type, provider, instance *reflect.Value, in Scope)
	InspectMapBinding   func(of reflect.Type, key string, annotation string, to reflect.Type, provider, instance *reflect.Value, in Scope)
	InspectParent       func(parent *Injector)
}

// Inspect walks the injector's bindings, including those made through the engine's API.
func (injector *Injector) Inspect(inspector Inspector) {
	engine := injector.mustRoot("Injector.Inspect")

	// the first three signatures are identical, because Scope is the engine's Scope; assigning
	// them directly keeps each callback's nil-ness, which the engine branches on
	rootInspector := v0.Inspector{
		InspectBinding:      inspector.InspectBinding,
		InspectMultiBinding: inspector.InspectMultiBinding,
		InspectMapBinding:   inspector.InspectMapBinding,
	}

	if inspector.InspectParent != nil {
		rootInspector.InspectParent = func(parent *v0.Injector) {
			inspector.InspectParent(attachedOf(parent))
		}
	}

	engine.Inspect(rootInspector)
}
```

- [ ] **Step 4: Re-enable the deferred entry**

Uncomment the `"Inspect"` entry in `TestInjector_ZeroValuePanics` (`v2/bind_key_test.go`, Task B3).

- [ ] **Step 5: Run, lint, commit**

```bash
CGO_ENABLED=1 go test -race ./v2/ -v
cd v2 && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./... && cd ..
git add v2/inspect.go v2/inspect_test.go v2/fixtures_test.go v2/bind_key_test.go
git commit -m "feat(v2): add Inspect over the engine's bindings"
```

### Task B16: the tracing switch

**Files:**
- Create: `v2/tracing_test.go`

**Interfaces:**
- Consumes: `EnableInjectionTracing` (B2). No production code changes.

The v2 `EnableCircularTracing` wrapper is **excused** as R-26: a cycle without tracing overflows the stack, so a test would crash the test binary rather than fail it, and the root's `TestDingoCircular` pins the traced panic.

- [ ] **Step 1: Write the test file**

`v2/tracing_test.go`. This is the suite's second non-parallel island, after C-04. The switch has no off switch, so the root's own test stays the primary pin and this one proves only that the wrapper reaches the engine:

```go
package dingo_test

import (
	"bytes"
	"log/slog"
	"testing"

	dingo "flamingo.me/dingo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tracedDependency struct{}

type tracedTarget struct {
	Dependency *tracedDependency `inject:""`
}

// TestEnableInjectionTracing_ReachesTheEngine
// Covers R-27 (the v2 view; the engine's own log lines are pinned by the root's
// tracing_test.go).
// Catches: a wrapper that stores a v2-local flag instead of calling the engine's switch, which
// would make the only debugging aid for a mis-wired graph silently do nothing.
//
//nolint:paralleltest // swaps the process-wide slog default and flips a switch with no off switch
func TestEnableInjectionTracing_ReachesTheEngine(t *testing.T) {
	var buffer bytes.Buffer

	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buffer, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	dingo.EnableInjectionTracing()

	injector := newInjector(t)

	_, err := injector.GetInstance[*tracedTarget]()
	require.NoError(t, err)

	assert.Contains(t, buffer.String(), "SETTING FIELD: Dependency")
}
```

- [ ] **Step 2: Run, lint, commit**

```bash
go test ./v2/ -run 'TestEnableInjectionTracing' -v
CGO_ENABLED=1 go test -shuffle=on -race ./v2/
cd v2 && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./... && cd ..
git add v2/tracing_test.go
git commit -m "test(v2): pin that the injection tracing switch reaches the engine"
```

### Task B17: `compat` — two APIs on one engine

**Files:**
- Create: `v2/compat/doc.go`, `v2/compat/compat.go`, `v2/compat/compat_test.go`

**Interfaces:**
- Consumes: `hooks.Attached`, `hooks.RootOf`, `hooks.AsModule` (A2, B2).
- Produces: `func Injector(root *v0.Injector) *dingo.Injector`, `func Root(injector *dingo.Injector) *v0.Injector`, `func FromRoot(module v0.Module) dingo.Module`, `func ToRoot(module dingo.Module) v0.Module`.

- [ ] **Step 1: Write the failing tests**

`v2/compat/compat_test.go` (package `compat_test`). It carries its own `newInjector` for the v0 side, built on the root API the same way, plus `rootModule`/`v2Module` fixture pairs that record their `Configure` calls and carry `inject`-tagged fields of both injector types, and variants with an `Inject` method.

Every rule with two directions runs both as subtests: a v0 module on a v2 injector and a v2 module on a v0 injector. Module identity is asserted by **counting `Configure` calls per module value**, never by comparing adapter values. Injection timing is asserted from **inside `Configure`**: the module records what its injected fields held when `Configure` ran.

| Test function | Covers | Body |
|---|---|---|
| `TestInjector_IsIdempotentPerRoot` | X-01 | `compat.Injector(e)` twice returns the same typed injector (`assert.Same`); `compat.Root(compat.Injector(e))` is `e`; and `Inspect` on `e` lists the `v2.Injector` binding whether or not `compat.Injector` was ever called — the clause that pins eager attachment |
| `TestInjector_IsDistinctPerChildRoot` | X-02 | two subtests. `child_created_through_the_engine`: `compat.Injector(e.Child())` differs from `compat.Injector(e)` and wraps the child engine — Flamingo's `area.Injector = parent.Child()` path. `child_created_through_the_typed_api`: `compat.Injector(e).Child()` differs from the parent typed injector and its engine reports `e`'s typed injector as parent through `Inspect` |
| `TestInjector_SharesBindingsWithTheV0API` | X-03 | both directions: a v0 `Bind(new(I)).To(Impl{})` resolves through `GetInstance[I]` on the typed injector, and a v2 `Bind[I]().To[Impl]()` resolves through the engine's `GetInstance(new(I))` |
| `TestMultibindingsAndMapBindings_ShareAcrossKinds` | X-17 | a v0 `BindMulti` entry and a v2 `BindMulti` entry arrive together in a `[]I` field of a module of either kind and in `GetInstance[[]I]()`; the same for `BindMap` entries |
| `TestToRoot_RunsAV2ModuleOnAV0Injector` | X-04 | `v0.NewInjector(compat.ToRoot(m))` configures `m` once and its bindings resolve through the root API |
| `TestFromRoot_RunsAV0ModuleOnAV2Injector` | X-05 | `dingo.NewInjector(compat.FromRoot(m))` configures `m` once and its bindings resolve through the v2 API |
| `TestAdaptedModules_KeepTheirIdentity` | X-06 | the same module value passed direct **and** adapted, and `ToRoot(FromRoot(m))`, configure once; two distinct module types both configure; `ModuleFunc` closures of either package stay distinct by value. `(decision)` — identity is the unwrapped module |
| `TestAdaptedModules_DependenciesResolveAcrossKinds` | X-07 | a v2 `Depender` under a v0 injector and a v0 `Depender` under a v2 injector: every dependency configures once, before the dependent |
| `TestAdaptedModules_FieldsAreInjectedBeforeConfigure` | X-08 | two subtests. Tagged fields: an adapted module of either kind reads its `inject` fields inside `Configure`, on either injector. `Inject` method: an adapted module with `Inject(cfg *struct{ Level string \`inject:"config:level,optional"\` })` has that method called, with the config visible, before `Configure` — the `core/zap` shape, and a different engine path from tagged fields, so both run |
| `TestAdaptedModules_InjectionFailureReturnsFromInitModules` | X-19 | an adapted module of either kind with an unresolvable `inject` field: `InitModules` on either injector returns the engine's `initmodules: injection into %q failed` error **naming the unwrapped module's type**, `TryModule` of either package returns it, and `Configure` never runs. Runs once with a pointer-typed inner module and once with a value-typed one (`MyModule{}`) — the latter is what the root PR's pointer guard makes possible |
| `TestSelfBinding_EachKindReceivesItsOwnInjector` | X-09 | a `*v0.Injector` field receives the engine, a `*dingo.Injector` field receives its typed injector; inside a child, the child's |
| `TestScopes_AreSharedAcrossKinds` | X-10 | a `Singleton`-scoped binding made through one API and resolved through the other is one instance; a scope registered with `BindScope` on the engine serves `In` on the typed injector |
| `TestInterceptors_ApplyAcrossKinds` | X-11 | a v2 `BindInterceptor[I, W]` wraps a v0-bound value; a v0 `BindInterceptor(new(I), W{})` wraps a v2-bound value |
| `TestBindTimeChecks_SurfaceThroughRootEntryPoints` | X-12 | a v2 misuse inside a `ToRoot` module: the root's `TryModule` returns an error wrapping `dingo.ErrInvalidBinding`; the root's `NewInjector` panics with it |
| `TestEngineErrors_KeepRootWordingThroughTheTypedAPI` | X-18 | an unbound-interface resolution error (`can not instantiate interface`, **returned**, not panicked) and a DUP-02 duplicate-binding error reached through the typed API carry the engine's v0 wording, not the `dingo: <call>: <reason>` shape. `(decision)` |
| `TestConfigLoop_V0RuntimeTypedBindingsReachV2Fields` | X-13 | v0 `Bind(v).AnnotatedWith("config:k").ToInstance(v)` for `string`, `bool` and `float64` is read by a v2 module's `inject:"config:k"` fields and by `GetAnnotatedInstance[T]` — the Flamingo `area.go` shape during adoption steps 0 to 2 |
| `TestInspect_TranslatesTheParentToItsAttached` | X-14 | v2 `Inspect` lists bindings of both origins and hands `InspectParent` the parent's typed injector, the same value `compat.Injector(parentEngine)` returns |
| `TestTryModule_AcceptsAdaptedModules` | X-15 | `dingo.TryModule(compat.FromRoot(m))` and `v0.TryModule(compat.ToRoot(m))` both skip eager singletons and return bind errors |
| `TestSentinels_AreSharedValues` | X-16 | `errors.Is` holds across packages for `ErrPointerToInterface`, `ErrInitModules`, `ErrInvalidInjectReceiver`, `ErrModuleCycle` and `ErrModuleSort` |

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./v2/compat/ -v`
Expected: compile failure, package `compat` does not exist.

- [ ] **Step 3: Write the implementation**

`v2/compat/doc.go`:

```go
// Package compat adapts dingo injectors and modules between flamingo.me/dingo and
// flamingo.me/dingo/v2, so that v0 and v2 modules run on one injector.
//
// It exists for the migration. It is not part of the v2 compatibility promise and is removed in a
// later v2 minor release, announced in at least two shipped minors before that, in the changelog,
// in this documentation and in the v2 README. Projects that still need it pin the last minor that
// ships it.
//
// Using v2 inside a module that keeps the v0 signature:
//
//	func (m *Module) Configure(injector *dingo.Injector) { // v0 signature, unchanged
//		i := compat.Injector(injector)
//		i.Bind[TransactionLog]().To[DatabaseTransactionLog]()
//	}
//
// Listing a v2 module where v0 modules are expected:
//
//	flamingo.App([]dingo.Module{
//		new(locale.Module),         // v0
//		compat.ToRoot(new(MyModule)), // v2
//	})
package compat
```

`v2/compat/compat.go`:

```go
package compat

import (
	v0 "flamingo.me/dingo"
	"flamingo.me/dingo/internal/hooks"
	dingo "flamingo.me/dingo/v2"
)

// Injector returns the v2 injector that shares the root injector's bindings, scopes and
// interceptors. The typed injector is attached when the root injector is created, so every call
// returns the same value and none of them writes to the root injector.
func Injector(root *v0.Injector) *dingo.Injector {
	attached, ok := hooks.Attached(root).(*dingo.Injector)
	if !ok {
		// coverage: unreachable through the public API; importing this package links v2, which
		// installs the hook the root calls in every constructor
		panic("dingo/compat: this root injector has no typed API")
	}

	return attached
}

// Root returns the root injector behind a typed injector.
func Root(injector *dingo.Injector) *v0.Injector {
	root, ok := hooks.RootOf(injector).(*v0.Injector)
	if !ok {
		// coverage: unreachable through the public API; every v2 injector wraps a root injector
		panic("dingo/compat: this injector has no root injector")
	}

	return root
}

// ToRoot adapts a v2 module so a root injector can run it. The root injector keys its module graph
// by the unwrapped module, so m, ToRoot(m) and ToRoot(FromRoot(m)) are one module.
func ToRoot(module dingo.Module) v0.Module {
	adapted, ok := hooks.AsModule(module).(v0.Module)
	if !ok {
		// coverage: unreachable through the public API; the argument is a dingo.Module by type
		panic("dingo/compat: not a v2 module")
	}

	return adapted
}

// FromRoot adapts a v0 module so a v2 injector can run it.
func FromRoot(module v0.Module) dingo.Module {
	return rootModule{module: module}
}

// rootModule runs a v0 module against the engine behind a v2 injector. The engine injects
// hooks.Unwrap(module) — the v0 module itself — immediately before Configure.
type rootModule struct{ module v0.Module }

// Configure runs the v0 module against the engine behind the typed injector.
func (m rootModule) Configure(injector *dingo.Injector) { m.module.Configure(Root(injector)) }

// Depends adapts the v0 module's dependencies, returning nil when it declares none.
func (m rootModule) Depends() []dingo.Module {
	depender, ok := m.module.(v0.Depender)
	if !ok {
		return nil
	}

	dependencies := depender.Depends()

	adapted := make([]dingo.Module, len(dependencies))
	for i, dependency := range dependencies {
		adapted[i] = FromRoot(dependency)
	}

	return adapted
}

// DingoUnwrap reports the module this adapter wraps, so the engine keys its module graph by
// the unwrapped module.
func (m rootModule) DingoUnwrap() any { return m.module }
```

- [ ] **Step 4: Run, lint, commit**

```bash
CGO_ENABLED=1 go test -shuffle=on -race ./... ./v2/...
cd v2 && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./... && cd ..
git add v2/compat
git commit -m "feat(v2): add compat, adapting injectors and modules between v0 and v2"
```

### Task B18: the Flamingo-shaped integration test

**Files:**
- Modify: `v2/compat/compat_test.go` (add X-20)

**Interfaces:**
- Consumes: everything B17 produced. No production code changes.

X-20 is an integration test, not a rule: the one place the Flamingo bootstrap shape is exercised end to end before Flamingo adopts v2. It mirrors `framework/config/area.go:305-340` on one engine.

- [ ] **Step 1: Write the test**

`TestFlamingoShape_MixedModuleTreeOnOneRoot`, in `v2/compat/compat_test.go`.

Setup, all on **one** engine:
- a v0 engine created with `SetBuildEagerSingletons(false)`, an `Bind(area{}).ToInstance(&area{})` binding, and config values bound under `config:`-prefixed annotations;
- one **v0** module whose `Configure` uses `compat.Injector(...)` for its own bindings (adoption step 0) and which declares `Depends()`;
- one `compat.ToRoot(v2Module)` whose `Inject(cfg *struct{...})` method reads config inside `Configure`;
- both kinds contributing `BindMulti` and `BindMap` entries, one of them through a `ToProvider` taking an annotated anonymous struct.

Assertions, in this order:
1. each `Configure` runs exactly once, in dependency order;
2. `Inject` runs before `Configure`, with the config visible;
3. `GetInstance[[]command]()` merges both origins in parent order;
4. `compat.Root(compat.Injector(engine))` is `engine`.

Then `engine.Child()` **on the v0 side**, and:

5. `compat.Injector(child)` differs from the parent typed injector;
6. a v2 module in the child receives both the typed API and the v0 child;
7. a child-bound binding shadows the parent's;
8. `Inspect` lists both origins and `InspectParent` receives exactly the parent typed injector.

The doc comment carries `// Covers X-20.` and a `Catches:` line naming what it protects: a typed injector or module-identity regression that each unit test still passes but that breaks the real bootstrap, which is the only shape that exercises child areas, mixed module kinds, delayed eager singletons and config annotations together.

- [ ] **Step 2: Run, lint, commit**

```bash
CGO_ENABLED=1 go test -shuffle=on -race ./v2/compat/ -v
cd v2 && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./... && cd ..
git add v2/compat/compat_test.go
git commit -m "test(v2): add the Flamingo-shaped mixed module tree integration test"
```

### Task B19: the compile-fail corpus

**Files:**
- Create: `v2/testdata/compilefail/go.mod`, `v2/testdata/compilefail/go.sum`
- Create: `v2/testdata/compilefail/cases/<name>/main.go` (six cases)
- Create: `v2/compilefail_test.go`

**Interfaces:**
- Produces: the only pin for B-12, which is a compile-time-only rule and can have no runtime test.

A nested module keeps the corpus out of the parent module's build, vet, lint and dependency graph, and no build tag can pull it in by accident. One package per case, so one deliberate error cannot mask another.

- [ ] **Step 1: Write the corpus module**

`v2/testdata/compilefail/go.mod`:

```
module flamingo.me/dingo/v2/testdata/compilefail

go 1.27

require flamingo.me/dingo/v2 v2.0.0

replace flamingo.me/dingo/v2 => ../..

replace flamingo.me/dingo => ../../..
```

Generate `go.sum` with `cd v2/testdata/compilefail && GOWORK=off go mod tidy`, and commit it: the corpus builds under `-mod=readonly` and needs the transitive gonum and testify sums.

Each case is `cases/<name>/main.go`, whose **first line** is `// want: <substring>`:

| Case | First line | Body |
|---|---|---|
| `to_instance_wrong_type` | `// want: cannot use` | `injector.Bind[greeter]().ToInstance(42)` — `ToInstance` takes a typed `T`, so v0's runtime panic is now a compile error (B-12) |
| `get_instance_wrong_assignment` | `// want: cannot use` | `var s string = mustGet(injector.GetInstance[greeter]())`, or simply assign the first result of `injector.GetInstance[greeter]()` to a `string` |
| `bind_missing_type_argument` | `// want: cannot infer` | `injector.Bind()` without `[T]` |
| `to_missing_type_argument` | `// want: cannot infer` | `injector.Bind[greeter]().To()` without `[U]` |
| `interceptor_one_type_argument` | `// want: cannot infer` | `injector.BindInterceptor[greeter]()` |
| `to_unrelated_type_compiles` | `// want:` (empty — this case must **build**) | `injector.Bind[greeterA]().To[greeterB]()` with unrelated `greeterB`: documents the compile-time/bind-time boundary from the other side, and proves the corpus module's two `replace` directives still resolve, which renovate cannot see. The rejection itself is B-05 in `bind_target_test.go`, so this case carries **no** catalogue ID: one ID never stands for two opposite assertions |

Go 1.27 reports a missing type argument as `in call to i.Bind, cannot infer T`; the substring `not enough type arguments` never appears, so the want lines use `cannot infer`.

- [ ] **Step 2: Write the driver**

`v2/compilefail_test.go`:

```go
package dingo_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const corpusDir = "testdata/compilefail"

// TestCompileFail_MatchesExpectedErrors drives the compile-fail corpus.
// Covers B-12.
// Catches: a signature change that turns a compile error back into a runtime panic — the whole
// point of the generic API — and a broken replace directive in the corpus module, which would
// make every case "fail to build" for the wrong reason.
func TestCompileFail_MatchesExpectedErrors(t *testing.T) {
	t.Parallel()

	goTool, err := exec.LookPath("go")
	if err != nil {
		if os.Getenv("CI") != "" {
			require.NoError(t, err, "the go tool must be on PATH in CI")
		}

		t.Skip("go tool not found on PATH")
	}

	entries, err := os.ReadDir(filepath.Join(corpusDir, "cases"))
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	for _, entry := range entries {
		t.Run(entry.Name(), func(t *testing.T) {
			t.Parallel()

			source, err := os.ReadFile(filepath.Join(corpusDir, "cases", entry.Name(), "main.go"))
			require.NoError(t, err)

			first, _, ok := strings.Cut(string(source), "\n")
			require.True(t, ok)

			want := strings.TrimSpace(strings.TrimPrefix(first, "// want:"))
			require.NotEqual(t, first, want, "the first line must be a // want: directive")

			// GOWORK=off, because the corpus module is not in go.work and its replace directives
			// must apply
			cmd := exec.CommandContext(t.Context(), goTool, "build", "./cases/"+entry.Name())
			cmd.Dir = corpusDir
			cmd.Env = append(os.Environ(), "GOWORK=off")

			output, err := cmd.CombinedOutput()

			if want == "" {
				require.NoError(t, err, "this case must build: %s", output)

				return
			}

			require.Error(t, err, "this case must fail to build, but it built")
			assert.Contains(t, string(output), want)
			assert.Contains(t, string(output), "main.go")
		})
	}
}
```

- [ ] **Step 3: Run, lint, commit**

```bash
go test ./v2/ -run 'TestCompileFail' -v
cd v2 && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./... && cd ..
git add v2/testdata/compilefail v2/compilefail_test.go
git commit -m "test(v2): add the compile-fail corpus and its driver"
```

### Task B20: the behavior-ID gate

**Files:**
- Create: `v2/testdata/catalogue.txt`
- Create: `v2/catalogue_test.go`

**Interfaces:**
- Produces: the gate that makes "every behavior is pinned" machine-checked rather than remembered. Run this task **after** every test file exists, so a failure means a missing test rather than a missing task.

- [ ] **Step 1: Generate `testdata/catalogue.txt`**

One line per behavior ID, tab-separated:

```
ID<TAB>statement
ID<TAB>statement<TAB>excused: <reason>
```

The statement of an ID is the Case text of the first row that lists it in the appendix's Covers column, or its row under "IDs without a case of their own". Work through `2026-09-08-dingo-v2-generic-api-test-catalogue.md` section by section, in the order the sections appear, and transcribe. The file must end with exactly 182 lines:

B-01..B-45 including B-34a (46), K-01..K-12 (12), R-01..R-34 including R-11a and R-21a (36), MB-01..MB-18 (18), S-01..S-10 (10), C-01..C-05 (5), OV-01..OV-05 (5), DUP-01..DUP-03 (3), I-01..I-07 (7), M-01..M-09 (9), INS-01..INS-06 (6), G-01..G-05 (5), X-01..X-20 (20). Verify with `wc -l v2/testdata/catalogue.txt` and with `cut -f1 v2/testdata/catalogue.txt | sort -u | wc -l`; both must print 182.

The five excused lines are, verbatim in their third column:

| ID | Reason |
|---|---|
| `R-26` | `excused: a tracing-disabled cycle stack-overflows the process instead of returning a catchable panic; a test would crash the test binary, not fail it` |
| `I-07` | `excused: a data race cannot be asserted — under -race it fails the run, without -race it is invisible; synchronizing the interceptor map is a listed follow-up` |
| `DUP-03` | `excused: engine-internal; pinned by the root module's binding_test.go TestBinding_equal` |
| `R-25` | `excused: engine-internal; pinned by the root module's circular_test.go TestDingoCircular` |
| `M-03`, `M-05` | not excused — their **public** views are covered in `v2/module_test.go`; only the white-box halves stay in the root, and those have no ID of their own in this file |

R-27 is **not** excused: it is pinned in both places, by the root's `tracing_test.go` and by `v2/tracing_test.go`.

M-08's add-failure branch is excused at the **production site** — the `// coverage:` comment Task A4 added to `dingo.go` — not in this file, because M-08 itself has a test.

- [ ] **Step 2: Write the gate**

`v2/catalogue_test.go`:

```go
package dingo_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const cataloguePath = "testdata/catalogue.txt"

var coversPattern = regexp.MustCompile(`\b(B|K|R|MB|S|C|OV|DUP|I|M|INS|G|X)-\d+[a-z]?\b`)

// TestCatalogue_EveryIDHasAStatement
// Covers nothing: it is the gate itself.
// Catches: an ID with no definition anywhere, which nobody can write a test for because its
// meaning would have to be guessed from its prefix.
func TestCatalogue_EveryIDHasAStatement(t *testing.T) {
	t.Parallel()

	for id, entry := range readCatalogue(t) {
		assert.NotEmpty(t, entry.statement, "%s has no statement", id)
	}
}

// TestCatalogue_EveryIDIsReferencedOrExcused
// Catches: a behavior silently dropped from the suite — the failure mode this whole catalogue
// exists to prevent.
func TestCatalogue_EveryIDIsReferencedOrExcused(t *testing.T) {
	t.Parallel()

	referenced := referencedIDs(t)

	for id, entry := range readCatalogue(t) {
		if entry.excuse != "" {
			assert.NotContains(t, referenced, id, "%s is excused but also referenced; drop the excuse", id)

			continue
		}

		assert.Contains(t, referenced, id, "%s is neither referenced by a test nor excused", id)
	}
}

type catalogueEntry struct {
	statement string
	excuse    string
}

func readCatalogue(t *testing.T) map[string]catalogueEntry {
	t.Helper()

	content, err := os.ReadFile(cataloguePath)
	require.NoError(t, err)

	entries := make(map[string]catalogueEntry)

	for _, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
		columns := strings.Split(line, "\t")
		require.GreaterOrEqual(t, len(columns), 2, "malformed line: %q", line)

		entry := catalogueEntry{statement: strings.TrimSpace(columns[1])}
		if len(columns) > 2 {
			entry.excuse = strings.TrimSpace(strings.TrimPrefix(columns[2], "excused:"))
		}

		_, duplicate := entries[columns[0]]
		require.False(t, duplicate, "duplicate ID %q", columns[0])

		entries[columns[0]] = entry
	}

	return entries
}

// referencedIDs collects every ID named in a "Covers" comment across the v2 module, compat
// included.
func referencedIDs(t *testing.T) map[string]bool {
	t.Helper()

	referenced := make(map[string]bool)

	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		for _, line := range strings.Split(string(content), "\n") {
			if !strings.Contains(line, "Covers") {
				continue
			}

			for _, id := range coversPattern.FindAllString(line, -1) {
				referenced[id] = true
			}
		}

		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, referenced)

	return referenced
}
```

A `Covers` line may list a range as `R-01, R-02, R-03`; the gate reads individual IDs only, so write them out rather than as `R-01..R-03`.

- [ ] **Step 3: Run the gate and close every gap it reports**

Run: `go test ./v2/ -run 'TestCatalogue' -v`

Expected on the first run: a list of unreferenced IDs. Each one is a missing `Covers` line or a missing test. Fix them in the file that owns the ID per the appendix's section order — do **not** add an excuse to make the gate pass. An excuse is only legitimate for the four IDs listed above.

- [ ] **Step 4: Run, lint, commit**

```bash
CGO_ENABLED=1 go test -shuffle=on -race ./... ./v2/...
cd v2 && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./... && cd ..
git add v2/testdata/catalogue.txt v2/catalogue_test.go
git commit -m "test(v2): add the behavior-ID catalogue and its gate"
```

### Task B21: Examples, the README and the example application

**Files:**
- Create: `v2/example_hello_test.go`, `v2/example_provider_test.go`, `v2/example_annotated_test.go`, `v2/example_multibinding_test.go`, `v2/example_interception_test.go`, `v2/example_child_test.go`, `v2/example_migration_test.go`
- Create: `v2/compat/example_test.go`
- Create: `v2/README.md`, `v2/readme_test.go`
- Create: `v2/example/` (the root `example/` ported to v2)
- Modify: `Readme.md` (the root notice)

**Interfaces:**
- No production code changes.

- [ ] **Step 1: Write the Examples**

Six whole-file Examples, one per README section, plus two migration Examples and two in `compat/`. A file with exactly one `Example`, no `Test` functions and its own package-level declarations renders whole on pkg.go.dev, so **each file owns its exported fixtures** (`Greeter`, `PoliteGreeter`) with names disjoint across the files. Every Example ends in `// Output:`, or `// Unordered output:` where a map is printed.

Examples cannot use `newInjector` and therefore run against the global scopes, so **no Example binds `In(Singleton)`, `In(ChildSingleton)` or `AsEagerSingleton()`** unless its fixture types are unique to that file.

| File | Example | Section |
|---|---|---|
| `example_hello_test.go` | `ExampleInjector_Bind` | bind an interface to an implementation, resolve it |
| `example_provider_test.go` | `ExampleInjector_Bind_provider` | bind through `ToProvider` with injected arguments |
| `example_annotated_test.go` | `ExampleInjector_Bind_annotated` | two annotated bindings of one interface |
| `example_multibinding_test.go` | `ExampleInjector_BindMulti` | three `BindMulti` calls injected as a slice in order |
| `example_interception_test.go` | `ExampleInjector_BindInterceptor` | one interceptor wrapping a bound implementation |
| `example_child_test.go` | `ExampleInjector_Child` | a child injector resolving a parent-bound type |
| `example_migration_test.go` | `ExampleInjector_Bind_migration`, `ExampleInjector_Bind_config` | the v0/v2 pair side by side; the Flamingo config loop |
| `compat/example_test.go` | `Example_rootModule`, `ExampleToRoot` | v2 inside a v0 module; a v2 module in a v0 module list |

The canonical first Example, for shape:

```go
package dingo_test

import (
	"fmt"

	dingo "flamingo.me/dingo/v2"
)

// Greeter is the interface an injection site asks for.
type Greeter interface{ Greet() string }

// PoliteGreeter implements it.
type PoliteGreeter struct{}

func (PoliteGreeter) Greet() string { return "hello" }

// GreeterModule binds them together.
type GreeterModule struct{}

func (GreeterModule) Configure(injector *dingo.Injector) {
	injector.Bind[Greeter]().To[PoliteGreeter]()
}

func ExampleInjector_Bind() {
	injector, err := dingo.NewInjector(GreeterModule{})
	if err != nil {
		panic(err)
	}

	greeter, err := injector.GetInstance[Greeter]() // no type assertion, unlike v0
	if err != nil {
		panic(err)
	}

	fmt.Println(greeter.Greet())
	// Output: hello
}
```

- [ ] **Step 2: Write `v2/README.md`**

The current root `Readme.md`, rewritten. Every one of its 22 code blocks is rewritten to v2 syntax **by hand**: none compiles on its own today, ten are one-to-three-line fragments, and comparable libraries keep no generated code in their READMEs. Each section links to the matching `Example` on pkg.go.dev, which `go test` compiles and runs.

Required content beyond the rewrite:
- the Go 1.27 requirement and the import path;
- fix the wrong assignment in the "Requesting injection" block, where `m.processor = CreditCardProcessor` assigns a type;
- an **Interception** section stating that interceptors are inherited by child injectors like bindings, that a child's apply only to values resolved through that child while the parent's wrap outermost, and that the wrapper is rebuilt on every resolution even for a singleton base, so an interceptor must not keep state it expects to survive;
- a **"Using v2 next to v0"** section: the `compat` package, its four functions, its lifetime, and the two shapes from `compat/doc.go`;
- a **"Migrating from v0.x"** section carrying the spec's full migration table, the bind-time rules, the note that a binding has exactly one target, the two sentinels `ErrInvalidBinding` and `ErrPointerToInterface`, and the `map:` annotation footgun;
- a note that `Override` of a key nothing bound silently becomes a plain binding.

`v2/readme_test.go` is the drift guard:

```go
// TestReadme_UsesV2CallShapes
// Catches: a README block left on v0 syntax after a rewrite, which teaches the wrong API to
// every new reader.
func TestReadme_UsesV2CallShapes(t *testing.T) { /* ... */ }
```

It reads `README.md`, walks its fenced ```go blocks only (the migration table uses inline code, not fences), and fails on any of these substrings: `Bind(`, `BindMulti(`, `BindMap(`, `Override(`, `BindInterceptor(`, `GetInstance(`, `GetAnnotatedInstance(`, and `.To(` not followed by a type argument.

- [ ] **Step 3: Port the example application**

Copy `example/` to `v2/example/`, rewrite every call to the v2 API and keep the two directories structurally parallel, so that `diff -r example v2/example` reads as a migration reference. `v2/example` has no tests; it is compiled by `go test ./...` (a package without tests still fails the run on a compile error) and stays out of the coverage profile.

- [ ] **Step 4: Add the root README notice**

At the top of `Readme.md`, a short notice: `flamingo.me/dingo/v2` is the generic API over the same engine, new code should use it, and v0 keeps receiving fixes and stays the engine until the standalone step. **No `Deprecated:` marker** on the root package while Flamingo imports it, because staticcheck would then flag every Flamingo module.

- [ ] **Step 5: Run, lint, commit**

```bash
CGO_ENABLED=1 go test -shuffle=on -race ./... ./v2/...
go run ./v2/example
cd v2 && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./... && cd ..
git add v2/example_*_test.go v2/compat/example_test.go v2/README.md v2/readme_test.go v2/example Readme.md
git commit -m "docs(v2): add the README, the runnable Examples and the ported example application"
```

### Task B22: CI

**Files:**
- Modify: `.github/workflows/main.yml`, `.github/workflows/golangci-lint.yml`, `.github/workflows/semanticore.yml`

**Interfaces:**
- No production code changes.

- [ ] **Step 1: Read the current workflows**

Run: `cat .github/workflows/main.yml .github/workflows/golangci-lint.yml .github/workflows/semanticore.yml`. Keep every step this plan does not name; the changes below are additive or replace one line at a time.

- [ ] **Step 2: Edit `main.yml`**

| Job | Change |
|---|---|
| `tests` | matrix `['1.27', '1.*']`; run `go test -shuffle=on -race ./... ./v2/...` in workspace mode, so the typed injector is tested against the engine at the same commit. Shuffle is the cheap detector for order dependence through the global `Singleton` |
| `tests-v0` (new) | matrix `['1.25', '1.*']`, `GOWORK=off`, `go test -race ./...`: the root exactly as its consumers build it |
| `tests-v2-published` (new) | `GOWORK=off`, `working-directory: v2`, `go test -race ./...`: v2 against the root version its `go.mod` requires, fetched from the proxy. **`continue-on-error: true` unconditionally** while `v0.5.0` is unpublished, not only on pull requests: until then the engine at the vanity path has no `internal/hooks`, so the job cannot pass on any event, and requiring it on `master` would redden the default branch permanently. It is kept because it is the signal for when Task B24 step 1a becomes possible. Drop `continue-on-error` in the same commit as that bump, after which it is a real gate |
| `coverage` | one profile per module. The v2 profile runs over `go list ./... \| grep -Ev '/(example\|miniexample)$'` — anchored, so a future package whose path merely *contains* "example" stays in the profile. When `v2/coverage.min` exists, a step reads it and fails below it; until Task B24 adds that file the job only reports the figure. The root profile is reported as today and not gated |
| `static-checks` | `go vet ./... ./v2/...`; `gofmt` and `goimports` over the tree; `go generate` with a clean diff (still a no-op); and a new root smoke test: `go run ./miniexample 2>&1 \| grep -q 'here is an example log'` |
| all jobs | replace `go get -v -t -d ./...` with `go mod download` in each module: `go get` in workspace mode targets one module |

- [ ] **Step 3: Edit `golangci-lint.yml`**

Add a matrix over the working directories `.` and `v2`. The root `.golangci.yml` applies to both. Keep `only-new-issues: true`. The pinned version must be the one Task B1 step 1 verified.

- [ ] **Step 4: Edit `semanticore.yml`**

Add a guard that skips the run when the head commit message contains `[skip release]`, so that the v2 PR's squash commit does not open a `Release v0.4.2` pull request before the manual `v2.0.0` tag. A root-only squash commit carries the same marker, so a root merge does not cut a v2 release that carries no change for v2 consumers.

- [ ] **Step 5: Verify locally what can be verified locally**

```bash
CGO_ENABLED=1 go test -shuffle=on -race ./... ./v2/...
GOWORK=off go test -race ./...
cd v2 && GOWORK=off go test -race ./... ; cd ..   # expected to fail: the vanity path serves the upstream engine, which has no internal/hooks
go vet ./... ./v2/...
gofmt -l . && go run golang.org/x/tools/cmd/goimports@latest -w . && git diff --quiet
go generate ./... && git diff --quiet
go run ./miniexample 2>&1 | grep -q 'here is an example log'
```

- [ ] **Step 6: Commit**

```bash
git add .github/workflows
git commit -m "ci: test, lint and cover both modules, and guard semanticore with [skip release]"
```

### Task B23: v2 PR wrap-up

**Files:** none new.

- [ ] **Step 1: Full verification**

```bash
CGO_ENABLED=1 go test -shuffle=on -race -count=2 ./... ./v2/...
go vet ./... ./v2/...
gofmt -l .
go run golang.org/x/tools/cmd/goimports@latest -w . && git diff --quiet
cd v2 && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./... && cd ..
golangci-lint run ./...
```

Expected: all clean. The catalogue gate and the compile-fail driver run as part of the suite; if either is skipped rather than run, find out why before continuing.

- [ ] **Step 2: Measure coverage and record the figure**

```bash
cd v2
CGO_ENABLED=1 go test -shuffle=on -race -covermode=atomic -coverprofile=coverage.txt \
  $(go list ./... | grep -Ev '/(example|miniexample)$')
go tool cover -func=coverage.txt | tail -1
```

Write the figure down for Task B24; do **not** create `v2/coverage.min` yet. Every branch deliberately left uncovered must already carry `// coverage: unreachable through the public API` at the production site; grep for them and confirm each is genuinely unreachable rather than merely untested.

- [ ] **Step 3: Push and open the draft PR**

```bash
git push -u origin feat/v2-generic-api
```

Open a **draft** PR against `master`, no reviewers assigned. Title: `feat: add flamingo.me/dingo/v2, the generic binding API`.

Body sections: a summary; the migration table; the deliberate tightenings against v0; the compat lifetime; and the manual release steps — the `v2.0.0` tag on the merge commit and the `[skip release]` marker the squash commit must carry. The main commit carries a `BREAKING CHANGE:` footer describing the API, for the changelog.

State explicitly in the body that `tests-v2-published` is advisory until `v0.5.0` is published, and why: the engine at the vanity path does not yet carry `internal/hooks`. A reviewer should not read its red as a failure of this PR. The workspace-mode `tests` job is the one that gates.

### Task B24: release and the follow-up commit

**Files:** `v2/coverage.min` (created after the merge)

These steps run **after** the v2 PR is merged, and only with the user's explicit go-ahead: they publish tags and a release, which is not reversible.

- [ ] **Step 1: Tag and release `v2.0.0`**

Semanticore never raises a major, so the tag is manual. On the merge commit:

```bash
git tag v2.0.0 && git push origin v2.0.0
gh release create v2.0.0 --title "v2.0.0" --notes "…"
```

Tags for a major-version subdirectory are **bare** (`v2.0.0`, not `v2/v2.0.0`), as the Go module reference specifies for the `vN/` layout.

- [ ] **Step 1a: Bump the engine requirement to `v0.5.0`**

This is the step that ends the interim described in "How the v2 module resolves the engine". Run it once Part A is merged upstream and `v0.5.0` is on the proxy (`GOWORK=off go list -m -versions flamingo.me/dingo` lists it):

```bash
cd v2 && GOWORK=off go get flamingo.me/dingo@v0.5.0 && GOWORK=off go mod tidy
cd .. && GOWORK=off go test -C v2 -race ./...
```

In the same commit, drop `continue-on-error` from the `tests-v2-published` job so it becomes a real gate: from here on v2 is verified against the published engine, exactly as a consumer builds it. If `v0.5.0` is not yet on the proxy when the rest of B24 runs, this step waits — it does not block the `v2.0.0` tag.

- [ ] **Step 2: Commit the coverage floor**

Create `v2/coverage.min` holding the figure measured in Task B23 step 2, commit it on `master`, and from that commit on the coverage job gates. Lowering the file later takes an explicit commit that says why.

- [ ] **Step 3: Cut the v0 maintenance branch**

In the same session, cut `release/v0.x` from the `v0.5.0` release commit:

```bash
git branch release/v0.x <the v0.5.0 release commit>
git push -u origin release/v0.x
```

From `v2.0.0` on, every master push finds a v2 tag as its nearest tagged ancestor, so v2 releases are automatic. A hand-made `v0.5.1` tag on a master commit **newer** than the v2 tag would become the nearest tagged ancestor and take the automatic stream back to v0; the maintenance branch exists so that nobody tags v0 on master by reflex. An engine fix lands on master first — workspace-mode CI tests the typed injector against the engine at the same commit — then is cherry-picked to `release/v0.x`, tagged there by hand and released with `gh release create`. Afterwards, bump `flamingo.me/dingo` in `v2/go.mod` on master (renovate proposes it) so semanticore releases the v2 patch that carries the fix.

---

## Self-review

Run against the spec and its appendix after the plan was complete.

**1. Spec coverage.** Every section of the design doc maps to a task: Public API → B2, B3, B14, B15; `compat` → B17, B18; Semantics (keys, targets, attributes, interceptors, resolution) → B3 to B14; Interop → B17, B18; Deliberate tightenings → B4 (provider shape, one target), B10 (MB-15, MB-16), B14 (B-34a), B7 (G-05); Kept v0 behaviors → B7, B10, B12, B13; Bind-time messages → B6; Internals → B2, B3; `internal/hooks` → A2, A5, A6; Root changes → A1 to A7; Versioning/CI/release → B1, B22, B24; Documentation → B21; Testing (file layout, naming, helpers, patterns, hermeticity, compile-fail, coverage, lint) → B2, B3, B19, B20, B21, B23. All 182 IDs are assigned to a file by the catalogue's section order, and Task B20's gate fails the build on any that is not.

**2. Placeholder scan.** No step says "add tests", "handle edge cases", "similar to Task N" or "TBD". Every production file appears as complete code. Test tasks either give the code or give the row set with its fixture and assertion; the catalogue gate turns any omission into a build failure rather than an oversight.

**3. Type consistency.** The names used across tasks were checked against each other: `Injector.root`, `mustRoot(call string) *v0.Injector`, `attachedOf(root *v0.Injector) *Injector`, `invalid(call, format string, args ...any) error`, `qualified`, `typeName[T]`, `keyType[T]`, `carrier`, `callOf[T]`, `keyOf[T](call, entry string)`, `collectionKeyOf[T](call, entry, collection string)`, `Binding[T].{must,setTarget,checkTarget,setScope}`, `asModule`/`asModules`, `adapt[T]`, `resolve[T]`, and the helpers `newInjector`, `childOf`, `bindErr`, `bindPanic`, `recovered`, `requireInvalidBinding`, `get[T]`. `keyOf` is the validating form and `keyType` the non-validating one; `binding.go` and `resolve.go` use `keyType`, the four entry points use `keyOf`.

**4. Three ordering dependencies are called out where they bite**, rather than left to be discovered: `TestInjector_ZeroValuePanics` references `Inspect` and `BindInterceptor` before B14 and B15 exist (entries commented out, re-enabled by those tasks); `TestBindMessages_HaveTheDocumentedShape` references `BindInterceptor` and `GetInstance[**T]` likewise; and `child_test.go`'s C-03 needs `BindInterceptor`, so B12 runs after B14 or writes C-03 last.

**5. What this plan deliberately does not do.** It does not restate the 182 catalogue statements — they live in the appendix and are copied into `testdata/catalogue.txt` by Task B20, which is the single source the gate reads. It does not write the README's 22 rewritten blocks out, because the source blocks are in the root `Readme.md` and the rewrite is mechanical per the migration table. It does not cover the Flamingo, commerce or om3 adoption PRs, which the spec lists as out of scope.

