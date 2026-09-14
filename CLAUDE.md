# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`flamingo.me/dingo` is a Guice-style, reflection-based dependency injection library for Go. It is a single package at the repository root (module `flamingo.me/dingo`, `go 1.25.8`). Its main consumer is the Flamingo framework; keep the public API and runtime behavior stable.

`example/` and `miniexample/` are runnable demos (`go run ./example`, `go run ./miniexample`), not part of the library. `testdata/moduleidentity/` holds fixture packages with deliberately colliding type names for module identity tests.

## Commands

```sh
CGO_ENABLED=1 go test -race ./...            # what CI runs
go test -run 'TestName' -v .                  # single test (regex; add /subtest for a subtest)
go vet ./...
gofmt -l .                                    # CI fails if this prints anything
go run golang.org/x/tools/cmd/goimports@latest -w .   # CI fails on a resulting diff
golangci-lint run ./...
```

Lint notes:
- `.golangci.yml` enables a strict explicit list (`wsl_v5`, `paralleltest`, `tparallel`, `testpackage`, `thelper`, `varnamelen`, `mnd`, `err113`, `wrapcheck`, `nolintlint` with required explanations, etc.). CI pins golangci-lint v2.13.2 and reports only new issues; a full local run currently shows one pre-existing `unconvert` finding in `module_test.go` that is not yours to fix unless asked.
- `//nolint` needs both a specific linter and an explanation.

Before editing Go code, invoke the `use-modern-go` skill shipped in `.claude/skills/` and apply its guidance for the go.mod Go version.

## Architecture

Everything lives in five files of `package dingo`:

- `dingo.go`: `Injector` (bindings, multibindings, mapbindings, interceptors, scopes, parent link, stage) and the whole resolution engine.
- `binding.go`: the fluent `Binding` builder (`To`, `ToInstance`, `ToProvider`, `AnnotatedWith`, `In`, `AsEagerSingleton`) and `Provider.Create`.
- `module.go`: `Module`, `ModuleFunc`, `Depender`, `TryModule`, and the module dependency graph (gonum `topo.SortStabilized`, ties broken by insertion order).
- `scope.go`: `Scope` interface, `SingletonScope` / `ChildSingletonScope`, and the package-level `Singleton` / `ChildSingleton` values.
- `inspect.go`: read-only `Inspect` callbacks over an injector's bindings.

### Resolution flow

`GetInstance` → `getInstance` (strips all pointer levels, or accepts a `reflect.Type` directly) → `getInstanceOfTypeWithAnnotation` (find binding, delegate to scope if bound in one, then `intercept`) → `createInstanceOfAnnotatedType` → `requestInjection`.

`createInstanceOfAnnotatedType` decides in this order: bound binding (`resolveBinding`: instance wins over provider wins over `to` type) → func type whose name ends in `Provider` with 1 or 2 results gets an auto-generated provider via `reflect.MakeFunc` → slice type resolves as a multibinding → `map[string]T` resolves as a map binding → otherwise `reflect.New` on a concrete type and inject into it. Interfaces, non-`Provider` funcs and annotated requests fail here unless `optional`.

`requestInjection` walks a worklist (pointer → call `Inject(...)` method if present, then struct fields tagged `inject:"annotation[,optional]"`, then interfaces and slices). An `Inject` method on a value receiver is an error (`ErrInvalidInjectReceiver`); pointer-to-interface fields are rejected.

### Behaviors that are easy to get wrong

- **Binding key** is the requested type with one pointer level removed: `Bind(new(Iface))`, `Bind((*Iface)(nil))` and `Bind(Iface{})` all key on `Iface`.
- **Lookup walks the parent chain** (`findBindingForAnnotatedType`, `joinMultibindings`, `joinMapbindings`, `intercept`). A child created with `Child()` sees parent bindings; the parent never sees the child's.
- **`Singleton` is one process-wide scope object** that every injector registers, so independent injectors share singleton instances. `ChildSingleton` is re-created per `Child()`. Scopes are looked up by their Go type.
- **Interception runs after the scope cache**, so each resolution of an intercepted singleton builds a fresh wrapper around the same cached inner value. Parent interceptors also wrap values resolved through children. Only interfaces can be intercepted.
- **Annotation prefix `map:`** on an inject tag selects one key of a map binding; user annotations starting with `map:` are reserved.
- **`InitModules` phases**: build module graph and sort → inject module fields → `Configure` → apply lazy `Override`s → duplicate-binding check (identical duplicates via `binding.equal` are tolerated, differing ones fail) → run `RequestInjection` calls that were queued while `stage == INIT` → build eager singletons.
- **Module identity** for graph dedup is `reflect.Type`; `ModuleFunc` values are additionally keyed by function value so distinct closures stay distinct.
- **`TryModule`** disables eager singletons and turns panics into errors; it proves bind-time acceptance, not construction.
- **Tracing switches** (`EnableCircularTracing`, `EnableInjectionTracing`) are package-level globals that affect every injector in the process.

## Testing conventions

- Tests use testify (`assert` / `require`). Older files are white-box (`package dingo`); newer ones are black-box (`package dingo_test`), which the `testpackage` linter prefers. Use `t.Parallel()` in new tests (`paralleltest` / `tparallel` are enabled) and `t.Helper()` first in helpers.
- Test names follow `Test<Surface>_<Rule>` (for example `TestInitModules_DistinctPackagesSameTypeName`).
- Anything reachable only through internals (module graph tables, `binding.equal`, circular tracing) stays white-box in the root package.

## Commits and releases

- Conventional commit messages (`feat:`, `fix:`, `chore:`, `refactor:`, `docs:`, optional scope like `feat(dingo):`). Semanticore runs on every push to `master`: a `feat:` in the range yields a minor release, otherwise a patch, and it opens a `Release vX.Y.Z` PR. It never bumps the major.
- `Changelog.md` is generated by that process; do not hand-edit it.
- Renovate manages dependency bumps, including `go run ...@version` pins in workflows.
- Do not raise the `go` directive in `go.mod` casually. The `errors.AsType` TODO in `module.go` waits on a newer toolchain on purpose, and downstream consumers build against `1.25.8`.

## Planned v2 (not yet implemented)

`docs/v2-generic-api/` holds an approved design for `flamingo.me/dingo/v2`: a type-safe generic facade (`injector.Bind[T]().To[U]()`) over this v0 engine, shipped as a `v2/` module in this repository with a `compat` bridge. Read the design doc and its test catalogue before any work touching v2, the `internal/bridge` idea, or root changes the spec labels behavior-neutral. Until that lands, the root package is the only code.
