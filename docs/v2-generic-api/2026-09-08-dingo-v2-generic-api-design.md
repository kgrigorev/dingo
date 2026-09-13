# dingo v2: generic method binding API

Date: 2026-09-08, revised 2026-09-09 (test suite redesigned, review decisions recorded, facade
architecture adopted for Flamingo compatibility)
Repository: github.com/i-love-flamingo/dingo (import paths `flamingo.me/dingo`, `flamingo.me/dingo/v2`)
Status: design approved in brainstorming, implementation plan pending
Appendix: `2026-09-08-dingo-v2-generic-api-test-catalogue.md` (planned test cases by behavior ID)

## Goal

Give dingo a type-safe binding API built on Go 1.27 generic methods, keeping the names and the
fluent chain dingo users know:

```go
injector.Bind[TransactionLog]().To[DatabaseTransactionLog]()
injector.Bind[CreditCardProcessor]().ToInstance(paypal)
injector.Bind[*http.ServeMux]().ToInstance(http.NewServeMux())
injector.BindMulti[web.Filter]().To[MetricsFilter]()
service, err := injector.GetInstance[*BillingService]()
```

Wrong usage fails to compile where Go can express the rule. Everything else panics the moment
the binding method runs, inside `Configure`, with a message that names the call and the types,
so `dingo.TryModule` tests catch it.

Flamingo is dingo's main user. The new API must let Flamingo, its ecosystem modules and the
applications built on them move over module by module, on one injector, without a lockstep
release. The v0 reflection API therefore stays as the engine, and v2 is a facade over it.

## Decisions

| Topic | Decision | Reason |
|---|---|---|
| Delivery | New major `flamingo.me/dingo/v2` as a module in the `v2/` directory of the same repository, first tag `v2.0.0` | Go has no overloading, so `Bind`, `To`, `Binding` and the rest keep their names only in a new major. A separate import path lets v0 and v2 coexist in one build. The subdirectory lets engine and facade change in one commit and share private packages. |
| Architecture | v2 is a generic facade over the v0 engine. The engine stays in the root module and keeps its behavior; v2 requires the root module | Flamingo v3.17 and 245 module `Configure` implementations, plus eight OSS modules, speak v0. With two separate engines every module in the ecosystem would have to move at once inside a Flamingo major. One shared engine runs v0 and v2 modules on one injector, so the ecosystem migrates a module at a time. Runtime behavior is identical by construction, not by re-implementation. |
| Bridge | Package `flamingo.me/dingo/v2/compat` adapts injectors and modules both ways. The core package `flamingo.me/dingo/v2` names no v0 identifier | compat can be removed later without touching the v2 core API. |
| Root changes | Behavior-neutral only: a private bridge package, a facade slot on `Injector`, `moduleKeyOf` and `InitModules` looking through module adapters, a shared type-name package, `ErrPointerToInterface` exported, the `ErrModuleSort` cause wrapped, one test added, one renamed (full list under "Root changes, all behavior-neutral"). Released as v0.5.0 before v2 exists | Every existing v0 call takes the same code path as before. |
| Runtime behavior | Identical to v0.4.1, except two error messages: the pointer-to-interface injection error (review decision 3) and the sort error, which gains its cause (review decision 8) | The only additions are checks at bind time and one check in `GetInstance`. Fixes that land in the engine reach v2 through a version bump. |
| Runtime-typed bindings | None in v2.0 | Across Flamingo, commerce, om3, the tenant apps and two services (~800 call sites) only Flamingo's config loop binds a runtime type. It keeps binding through the v0 engine until Flamingo drops v0; the end-state type switch is in the migration guide. A `BindType(reflect.Type)` escape hatch can be added in a 2.x minor without breaking anyone. |
| Migration | Guide in the v2 README, no rewrite tool | The rewrites are mechanical. A tool can follow in its own PR. |
| Builder shape | One `Binding[T]` type, methods in any order, conflicts panic at bind time | Familiar. Guice-style staged types would add three exported types and a fixed order without improving the checks Go cannot express. |
| Provider typing | `ToProvider(provider any)`, validated at bind time | Go 1.27 cannot infer dependency types through a `~func(A) T \| ~func(A) (T, error)` union constraint (verified: "cannot infer A"). Numbered variants, the shape of closed PR #80, can be added later without breaking. |
| Test suite | Written from scratch, black box only, one rule per test, every behavior ID pinned | The v0 suite is a checklist of behavior, not reusable code. Engine internals are the root module's business and stay pinned by the root suite. |
| Standalone v2 | Planned end state once the ecosystem has migrated: the engine moves into the v2 module, compat and the root requirement go away | See "Path to a standalone v2". Until then the engine has one home, the root module. |

Related work: PR #83 (`type_parameters`) and branch `generic_bind` add function-based generic
wrappers on top of v0. This spec supersedes both; their branches stay untouched. Branch
`feat/to-provider-error-handling` is independent and lands on master on its own.

## Public API

### Package `flamingo.me/dingo/v2`

Carried over from v0.4.1 with the same names and behavior: `NewInjector`, `(*Injector).Child`,
`InitModules`, `SetBuildEagerSingletons`, `BuildEagerSingletons`, `RequestInjection`,
`BindScope`, `Inspect`, `TryModule`, `Module`, `ModuleFunc`, `Depender`, `Scope`, `Singleton`,
`ChildSingleton`, `NewSingletonScope`, `NewChildSingletonScope`, `SingletonScope`,
`ChildSingletonScope`, `Inspector` with its `Inspect*` callbacks, `EnableCircularTracing`,
`EnableInjectionTracing`, `ErrInitModules`, `ErrInvalidInjectReceiver`, `ErrModuleCycle`,
`ErrModuleSort`. `Module`, `ModuleFunc`, `Depender` and `Inspector.InspectParent` refer to the v2
`Injector`. `Scope` and the two scope types are aliases of the root types, `Singleton`,
`ChildSingleton` and the sentinels are the root values, so a scope or an error is one object no
matter which package names it.

New or changed:

```go
func (injector *Injector) Bind[T any]() *Binding[T]
func (injector *Injector) BindMulti[T any]() *Binding[T]
func (injector *Injector) BindMap[T any](key string) *Binding[T]
func (injector *Injector) Override[T any](annotatedWith string) *Binding[T]
func (injector *Injector) BindInterceptor[T, I any]()
func (injector *Injector) GetInstance[T any]() (T, error)
func (injector *Injector) GetAnnotatedInstance[T any](annotatedWith string) (T, error)

// Binding configures how one bound type is resolved. Only an Injector creates usable values;
// every method on a zero Binding panics with an error wrapping ErrInvalidBinding.
type Binding[T any] struct { /* unexported */ }

func (b *Binding[T]) To[U any]() *Binding[T]
func (b *Binding[T]) ToInstance(instance T) *Binding[T]
func (b *Binding[T]) ToProvider(provider any) *Binding[T]
func (b *Binding[T]) AnnotatedWith(annotation string) *Binding[T]
func (b *Binding[T]) In(scope Scope) *Binding[T]
func (b *Binding[T]) AsEagerSingleton() *Binding[T]

// ErrInvalidBinding is wrapped by every bind-time panic value.
var ErrInvalidBinding = errors.New("invalid binding")

// ErrPointerToInterface is the root module's sentinel of the same name. Field injection wraps
// it as before; GetInstance[*Iface]() wraps it too.
var ErrPointerToInterface = dingo.ErrPointerToInterface
```

Not carried into v2: `Bind(interface{})`, `BindMulti(interface{})`, `BindMap(interface{},
string)`, `Override(interface{}, string)`, `BindInterceptor(interface{}, interface{})`,
`GetInstance(interface{})`, `GetAnnotatedInstance(interface{}, string)`, the non-generic
`Binding` with `To`, `ToInstance`, `ToProvider` taking `interface{}`, the types `Instance` and
`Provider` including `Provider.Create`, and the stage constants `INIT` and `DEFAULT`, which have
no exported reader. Nothing outside dingo references `Instance`, `Provider`, `INIT` or `DEFAULT`.
All of them stay in the root module.

The zero `Injector` is unusable like the zero `Binding[T]`: every method panics with an error
wrapping `ErrInvalidBinding` that names the call and says to use `NewInjector`, `Child` or
`compat.Injector`.

### Package `flamingo.me/dingo/v2/compat`

```go
import (
	v0 "flamingo.me/dingo"
	dingo "flamingo.me/dingo/v2"
)

// Injector returns the v2 injector that shares engine's bindings, scopes and interceptors.
// The first call for an engine creates the facade and binds it, so a *dingo.Injector field of
// a v2 module resolves. Later calls return the same facade.
func Injector(engine *v0.Injector) *dingo.Injector

// Engine returns the v0 injector behind a v2 injector.
func Engine(injector *dingo.Injector) *v0.Injector

// FromV0 adapts a v0 module so a v2 injector can run it.
func FromV0(module v0.Module) dingo.Module

// ToV0 adapts a v2 module so a v0 injector can run it.
func ToV0(module dingo.Module) v0.Module
```

The package documentation states its lifetime: it exists for the migration, it is not part of the
v2 compatibility promise, and it is removed in a later v2 minor release after an announcement (see
"Path to a standalone v2").

### Root module `flamingo.me/dingo`

One exported addition: `var ErrPointerToInterface = errors.New("pointer to interface is not
allowed")`. The unexported `errPointersToInterface` becomes an alias of it and the field-injection
error keeps wrapping it. No exported symbol changes otherwise; the sort error keeps its sentinel
`ErrModuleSort` and only gains the wrapped cause. The two new packages `internal/bridge` and
`internal/typename` are not API; nothing outside this repository can import them.

## Semantics

### The type parameter T

`T` is the type an injection site declares: an interface, `*Service`, `string`, `[]Handler`,
`map[string]Iface`, or a `Provider`-suffixed function type. The binding key is `T` with one
pointer level removed, exactly what `Bind(new(X))` and `Bind(X{})` produced.

| v2 call | Key | Replaces |
|---|---|---|
| `Bind[Iface]()` | `Iface` | `Bind(new(Iface))`, `Bind((*Iface)(nil))` |
| `Bind[Service]()` | `Service` | `Bind(Service{})`, `Bind(new(Service))` |
| `Bind[*Service]()` | `Service` | same key; use it when `ToInstance` receives a `*Service` |
| `Bind[string]()` | `string` | `Bind("")`, `Bind(v)` for a string `v` |
| `Bind[[]RequestIdentifier]()` | `[]RequestIdentifier` | `Bind(new([]RequestIdentifier))` |
| `BindMulti[Handler]()` | `Handler` | `BindMulti(new(Handler))`; injected as `[]Handler` |
| `BindMap[Iface]("k")` | `Iface` | `BindMap(new(Iface), "k")`; injected as `map[string]Iface` |
| `Override[Iface]("ann")` | `Iface` | `Override(new(Iface), "ann")` |

Rejected at bind time by all four entry points: `T` is a pointer to an interface (dingo forbids
those at injection sites too), or a pointer to a pointer (`Bind(new(*X))` in v0 silently created a
useless binding).

### Targets

- `To[U]()`: `U` with all pointer levels removed, as v0 did. `U`, or `*U`, must be assignable
  to the key. `U` must differ from the key; v0 reported self-binding only at resolution time.
  A pointer-to-interface `U` is rejected. Resolution is unchanged, including binding an
  interface to another bound interface.
- `ToInstance(instance T)`: the compiler checks the type. The dynamic type and value are stored
  as in v0. An untyped nil is rejected; v0 crashed on it with a nil dereference. Typed nil
  pointers stay accepted.
- `ToProvider(provider any)`: must be a function with one or two results. The first result must
  be assignable to the key or to a pointer to it, without pointer stripping, as in v0. A second
  result must implement `error`. Arguments are free; they are injected when the provider runs.
  The error result is still ignored at resolution, as in v0.4.1; propagating it belongs to
  `feat/to-provider-error-handling`.
- Exactly one target per binding. A second `To`, `ToInstance` or `ToProvider` panics, and the
  message names the first call. In v0 an instance silently won over a provider, which won over a
  type.
- Bindings without a target stay legal, for example `Bind[web.Router]().In(dingo.ChildSingleton)`
  or `Bind[registry]().AsEagerSingleton()`. Their resolution is unchanged.

### Attributes

- `AnnotatedWith(a)`: set once. Repeating the same value is a no-op; a different value panics.
- `In(scope)`: nil panics. Same set-once rule.
- `AsEagerSingleton()`: sets `Singleton` and eager. Panics when a different scope is already set.
  Compatible with a preceding `In(dingo.Singleton)`.
- Method order is free. `.In(s).To[U]()` and `.To[U]().In(s)` both work, as in v0.
- `Override[T](a)` presets the annotation, so a following `AnnotatedWith(a)` with the same
  value is a no-op. Overriding an unknown binding stays an `InitModules` error.

### BindInterceptor[T, I]

`T` must be an interface type, not a pointer to one. `I` has its pointer levels removed and must
be a struct whose first field has type `T`, because resolution writes the intercepted value into
field 0. `*I` must implement `T`; methods with value receivers count. v0 checked only the
interface kind and failed at resolution otherwise. Registration and resolution are unchanged.

### Resolution

`GetInstance[T]()` and `GetAnnotatedInstance[T](a)` first reject a pointer-to-interface `T` with
an error wrapping `ErrPointerToInterface`. This check is new for this entry point: v0's
`GetInstance(new(*Iface))` stripped every pointer level and silently resolved `Iface`. Then they
hand `reflect.TypeFor[T]()` to the engine, which strips one pointer level and resolves the key as
it does for a struct field of type `T`, and adapt the result to `T` the way field injection
already does: dereference while the value is a pointer and not assignable to `T`, take the
address of a copy when `T` is a pointer and the engine returned a value. Failures return the
zero `T` and an error.

| Call | Result |
|---|---|
| `GetInstance[*Service]()` | the `*Service` the resolver creates or the bound instance |
| `GetInstance[Service]()` | a copy of the struct, as a `Service` field would receive |
| `GetInstance[Iface]()` | the bound implementation |
| `GetInstance[string]()` with `Bind[string]().ToInstance("x")` | `"x"` |
| `GetInstance[[]graphql.Service]()` | the multibinding slice |
| `GetInstance[map[string]Iface]()` | the map binding |
| `GetInstance[eventRouterProvider]()` | the generated provider function |
| `GetInstance[func() X]()` (no `Provider` suffix) | error, as in v0 |
| `GetInstance[*Iface]()` | error wrapping `ErrPointerToInterface` |

`RequestInjection(object any) error` is unchanged: during `InitModules` it is queued and runs
after every module has configured.

### Interop: two APIs, one engine

- One engine, one facade. `compat.Injector(e)` returns the same `*Injector` for the same `e`,
  and `compat.Engine` inverts it. `NewInjector` in v2 creates an engine and its facade together;
  `Child` of a facade is the facade of the engine's child.
- Both self-bindings exist on every engine that has a facade. A field of type `*dingo.Injector`
  (root) receives the engine, a field of type `*dingo.Injector` (v2) receives its facade. Inside a
  child, the child's.
- Module identity is the innermost module. The engine keys the module graph by the module inside
  any number of adapter layers, so `m`, `FromV0(m)` and `ToV0(FromV0(m))` are one module:
  configured once, and a `Depender`'s dependencies of either kind are configured once, before it.
  `ModuleFunc` values of either package are keyed by value, as v0 does today.
- An adapted module's `inject` fields are populated immediately before its `Configure` runs, by
  the same `InitModules` loop as for a native module of either kind. It can read them inside
  `Configure`; Flamingo's `core/zap` module reads its injected log level there, and others in
  Flamingo, commerce and om3 do the same (not counted). When injection fails, `InitModules`
  returns the engine's `initmodules: injection into %q failed` error, naming the innermost
  module's type, and `Configure` does not run. `TryModule` of either package returns it.
- Bindings, multibindings, map bindings, scopes and interceptors made through either API are
  visible to both. `Singleton` and `ChildSingleton` are the same scope objects in both packages,
  and a scope registered with `BindScope` on one side serves `In` on the other.
- v2's bind-time checks run before the engine is touched. A v2 misuse inside an adapted module
  panics with the same error wrapping `ErrInvalidBinding`, and the root's `TryModule` returns it.
- `Inspect` on a facade lists bindings of both origins; `InspectParent` receives the parent's
  facade.
- Sentinels are shared values: `errors.Is(err, dingo.ErrPointerToInterface)` with either
  package's name is the same test; likewise the four module and init errors.
- Panics and error messages produced by the engine keep their v0 wording, with two exceptions:
  the pointer-to-interface injection error now reads `pointer to interface is not allowed`
  (review decision 3), and the sort error now carries the cause it used to drop (review
  decision 8). Only v2's own checks use the `dingo: <call>: <reason>` shape.

### Kept v0 behaviors worth knowing

Each of these is v0.4.1 behavior, kept unchanged and pinned by a test marked as a decision, so
nobody mistakes it for an accident. Verified against `dingo.go` on 2026-09-09.

- A direct binding of `[]X` or `map[string]X` takes precedence over the `BindMulti` and `BindMap`
  results for the same injection site. A target-less direct binding falls through to them, which
  is how a multibinding slice can be given a scope.
- The annotation prefix `map:` selects one entry of a map binding: `inject:"map:key"` on a field,
  `GetAnnotatedInstance[X]("map:key")` as a call. User annotations starting with `map:` are
  therefore reserved.
- Interceptors registered on a parent injector also wrap values resolved through its children,
  outside the child's own interceptors. Interceptors bound on a child apply only to resolutions
  started at that child. This is Guice's rule (`createChildInjector` inherits bindings, scopes and
  interceptors; nothing flows up) and what Flamingo's per-area child injectors need. Kept, and
  stated in the README.
- Interception runs after the scope cache (`intercept` is the last call in
  `getInstanceOfTypeWithAnnotation`, `dingo.go:300`), so every resolution of an intercepted
  singleton builds a fresh wrapper with `reflect.New` plus full injection and puts the same
  cached value in field 0. Two `GetInstance` calls return two wrappers over one inner instance.
  An interceptor that holds state per wrapper therefore loses it between resolutions, and the
  wrapper's own `inject` fields are resolved again each time. `intercept` also recurses to the
  root injector on every resolution, whether or not any ancestor has an interceptor.
- The engine's interceptor map is not synchronized. `BindInterceptor` writes it, resolution reads
  it, and `InitModules` separates the two phases. Registering an interceptor on an injector that
  is already serving concurrent resolutions is a data race in v0 and stays one in v2.
- `Singleton` is one process-wide scope object that every injector registers, so two independent
  injectors share singleton instances for the same key and annotation. `ChildSingleton` is one
  scope object per injector. Scopes are looked up by their Go type, so `BindScope` with a fresh
  `NewSingletonScope()` swaps the cache an injector uses.
- An unbound concrete type resolves to a freshly constructed value with its `inject` fields
  filled; an unbound provider argument of type `int` arrives as `0`. Only interfaces, functions
  without the `Provider` suffix and annotated requests fail.
- Identical duplicate bindings for one key and annotation are tolerated; differing ones fail
  `InitModules`. `BindMulti` keeps duplicates. `BindMap` with a repeated key keeps the last one.
  `Override` targets singular bindings only; overriding a type bound only through `BindMulti`
  fails as an unknown binding.
- `TryModule` disables eager singletons, so it proves bind-time acceptance, not construction.

## Bind-time checks and messages

Every check panics with an `error` that wraps `ErrInvalidBinding`. `TryModule` already turns
panicking errors into return values, so module tests can assert
`errors.Is(err, dingo.ErrInvalidBinding)` and check the message.

Message shape: `dingo: <call>: <reason>`. `<call>` is the entry point with its type arguments,
plus the map key or annotation when set. Types are printed with `typename.Qualified` (full import
path). Examples:

| Check | Example |
|---|---|
| pointer to interface as `T` | `dingo: Bind[*flamingo.me/x.Iface]: pointer to interface, use Bind[flamingo.me/x.Iface]` |
| pointer to pointer as `T` | `dingo: BindMulti[**flamingo.me/x.Foo]: pointer to pointer is not allowed` |
| `To[U]` not assignable | `dingo: Bind[flamingo.me/x.Iface].To[flamingo.me/x.Impl]: flamingo.me/x.Impl is not assignable to flamingo.me/x.Iface` |
| `To[U]` self-binding | `dingo: Bind[flamingo.me/x.Foo].To[flamingo.me/x.Foo]: binding to itself` |
| untyped nil instance | `dingo: Bind[flamingo.me/x.Iface].ToInstance: nil instance` |
| provider not a function | `dingo: Bind[flamingo.me/x.Iface].ToProvider: provider must be a function, got string` |
| provider result count | `dingo: Bind[flamingo.me/x.Iface].ToProvider: provider must return 1 or 2 values, got 3` |
| provider result type | `dingo: Bind[flamingo.me/x.Iface].ToProvider: provider returns flamingo.me/x.Other, not assignable to flamingo.me/x.Iface` |
| provider second result | `dingo: Bind[flamingo.me/x.Iface].ToProvider: second return value must be error, got string` |
| second target | `dingo: Bind[flamingo.me/x.Iface].ToInstance: target already set by To[flamingo.me/x.Impl]` |
| annotation conflict | `dingo: Bind[flamingo.me/x.Iface].AnnotatedWith("b"): annotation already set to "a"` |
| scope conflict | `dingo: Bind[flamingo.me/x.Iface].In(*flamingo.me/dingo.ChildSingletonScope): scope already set to *flamingo.me/dingo.SingletonScope` |
| nil scope | `dingo: Bind[flamingo.me/x.Iface].In: nil scope` |
| eager after other scope | `dingo: Bind[flamingo.me/x.Iface].AsEagerSingleton: scope already set to *flamingo.me/dingo.ChildSingletonScope` |
| zero-value `Binding[T]` | `dingo: Binding[flamingo.me/x.Iface].To: not created by an Injector, use injector.Bind` |
| zero-value `Injector` | `dingo: Injector.Bind[flamingo.me/x.Iface]: zero Injector, use NewInjector, Child or compat.Injector` |
| interceptor target | `dingo: BindInterceptor[flamingo.me/x.Foo, flamingo.me/x.Wrap]: flamingo.me/x.Foo is not an interface` |
| interceptor shape | `dingo: BindInterceptor[flamingo.me/x.Iface, flamingo.me/x.Wrap]: flamingo.me/x.Wrap must be a struct whose first field is flamingo.me/x.Iface` |
| interceptor contract | `dingo: BindInterceptor[flamingo.me/x.Iface, flamingo.me/x.Wrap]: *flamingo.me/x.Wrap does not implement flamingo.me/x.Iface` |

Exact wording may differ in implementation; the prefix, the call, and the presence of both type
names are required. Types print with their defining package, so the scope types print as
`flamingo.me/dingo.SingletonScope` while they are aliases of the root types, and types declared
in the external test package print as `flamingo.me/dingo/v2_test.<name>` (verified 2026-09-09).
Tests match on the type name, not on the package path of dingo's own types.

## Internals

### The v2 core over the engine

- `Injector` holds one field, the engine `*v0.Injector`. Every entry method checks the field and
  panics with the zero-value message when it is nil.
- `Bind[T]` computes the key type (`T` minus one pointer level), validates it, and calls the
  engine's `Bind` with a type carrier `reflect.New(key).Interface()`, a `*key` the engine strips
  back to `key`. `BindMulti`, `BindMap` and `Override` do the same. `BindInterceptor[T, I]`
  passes `reflect.New(T)` and the zero value of `I` with its pointers stripped, so the engine
  stores exactly `T` and `I`.
- `Binding[T]` holds the engine's `*v0.Binding` and three flags: the target call already made,
  the annotation set, the scope set. Each method checks the field is set, validates, records the
  flag, then forwards: `To[U]` passes `reflect.New(U)`, `ToInstance` passes `any(instance)`,
  `ToProvider` passes the function, `AnnotatedWith`, `In` and `AsEagerSingleton` forward as is.
  The engine's own checks stay in place behind v2's, so a gap in a v2 check still fails the way v0
  did instead of silently passing.
- `GetInstance[T]` checks for pointer-to-interface, calls the engine's `GetInstance` with
  `reflect.TypeFor[T]()` (the engine accepts a `reflect.Type` directly), and adapts the result to
  `T` as described under Resolution. `GetAnnotatedInstance[T]` likewise.
- `NewInjector` creates an engine with the root's `NewInjector()`, attaches the facade through
  the bridge, then calls `InitModules`. `InitModules` wraps every v2 module in the engine adapter
  and calls the engine's `InitModules`, so the module graph, dedup, `Depender` handling, the
  delayed `RequestInjection` queue and eager singletons are the engine's. `TryModule` builds an
  engine the same way, disables eager singletons, and recovers panics as the root's does.
- The engine adapter implements the root's `Module` and `Depender` and the bridge's
  `WrappedModule`. Its `Configure(engine)` fetches the facade and calls the v2 module's
  `Configure` with it. Its `Depends` adapts the v2 dependencies. The adapter has no `inject`
  fields of its own. The engine's `InitModules` injects `bridge.Innermost(module)`, so the v2
  module's fields are set before `Configure`, and an injection failure returns from
  `InitModules` exactly as for a native module. The adapter never sees that error.
- `Inspector` is a v2 struct with the same reflect-typed callbacks as the root's. `Inspect`
  forwards to the engine and hands `InspectParent` the parent's facade.
- `Scope`, `SingletonScope` and `ChildSingletonScope` are type aliases; `Singleton`,
  `ChildSingleton`, the four `Err*` sentinels and `ErrPointerToInterface` are the root values.
  `EnableCircularTracing` and `EnableInjectionTracing` call the root functions.
- Type names in messages come from `flamingo.me/dingo/internal/typename`, shared with the engine.

### The private contract: `flamingo.me/dingo/internal/bridge`

Go's rule for `internal` packages is import-path based: `flamingo.me/dingo/v2` may import
`flamingo.me/dingo/internal/...` although it is a separate module (verified 2026-09-09 with a
two-module spike; a module outside the path is refused with "use of internal package ... not
allowed"). This package is the only coupling between engine and facade, and nothing outside the
repository can import it.

```go
package bridge

// WrappedModule is implemented by module adapters. The engine keys the module graph by the
// innermost module.
type WrappedModule interface {
	WrappedModule() any
}

// Innermost follows WrappedModule until a value does not implement it.
func Innermost(module any) any

// Installed by package flamingo.me/dingo in an init function.
// Facade returns the facade attached to an engine, creating it through NewFacade on first use.
var Facade func(engine any) any

// Installed by package flamingo.me/dingo/v2 in an init function.
var (
	// NewFacade creates the facade for an engine and binds it into the engine.
	NewFacade func(engine any) any
	// EngineOf returns the engine behind a facade.
	EngineOf func(facade any) any
	// ToEngineModule wraps a v2 module in the engine adapter.
	ToEngineModule func(module any) any
)
```

Values are typed `any` because neither package can import the other's types without a cycle;
each side asserts its own types. The facade lives in a slot on the engine, so it is created once
per engine, is found again by `compat.Injector`, and is collected together with its engine. A
global registry would pin every injector ever created.

### Root changes, all behavior-neutral

- `internal/bridge` as above. The root's `init` installs `Facade`.
- `Injector` gets an unexported slot for the facade, guarded by a mutex.
- `moduleKeyOf` keys by `bridge.Innermost(module)`: the innermost module's type, plus its value
  when the type is the root's `ModuleFunc`, or when it is not a root `Module` and its kind is
  `Func` (a v2 `ModuleFunc`). A module that is not wrapped takes exactly today's path.
- `InitModules` injects `bridge.Innermost(module)` instead of `module` and names the innermost
  module's type in its `initmodules: injection into %q failed` error. For an unwrapped module
  both are the module itself.
- `internal/typename` with `Qualified(reflect.Type) string`; `qualifiedTypeName` delegates to it
  and its test stays where it is.
- `ErrPointerToInterface` exported as described under Public API. The exported name is singular
  while the unexported one is plural; the alias keeps `errPointersToInterface` working, and the
  rename is deliberate, so a grep for the old name still finds the declaration.
- `sortModules` wraps the cause: `module.go:150` returns a bare `ErrModuleSort` and drops the
  error `topo.SortStabilized` returned, so a failure that is not a cycle arrives as "cannot sort
  modules" with nothing to debug. It becomes `fmt.Errorf("%w: %w", ErrModuleSort, err)`.
  `errors.Is(err, ErrModuleSort)` keeps matching; only the message grows. This is the second and
  last carve-out from "identical to v0.4.1" (review decision 8). No root test covers that branch
  today, and gonum's `topo.SortStabilized` reports cycles through the branch above it, so the
  line takes the `// coverage:` comment rather than a test.
- One test added: `EnableInjectionTracing` emits log lines for field sets and resolutions
  (behavior R-27, untested today). The root's `TestDingoCircula` gets its missing letters.
- The `errors.AsType` TODO at `module.go:131` stays as it is. `errors.AsType` needs a `go` line of
  1.26 or later and the root keeps `go 1.25.8`, so only the v2 module can use it (see Testing).
- `Bind`, the module graph, resolution, scopes, interceptors and `Inspect` are untouched. Root
  `go test ./...` passes before and after with no test edited except the rename.

### compat

- `Injector(e)` asserts `bridge.Facade(e)`. `Engine(f)` asserts `bridge.EngineOf(f)`. `ToV0(m)`
  asserts `bridge.ToEngineModule(m)`.
- `FromV0(m)` returns compat's own adapter: it implements the v2 `Module` and `Depender` and
  `WrappedModule`; its `Configure(facade)` calls `m.Configure` with `Engine(facade)`. The engine
  adapter injects `m` before that call, because `Innermost` sees through both layers.
- compat has no state of its own. Deleting the directory removes every v0 name from the v2
  module.

## Versioning, toolchain, CI, release

Layout after the v2 PR:

```
go.mod                  module flamingo.me/dingo, go 1.25.8 (unchanged)
go.work                 go 1.27; use (. ./v2); committed, with go.work.sum when non-empty
internal/bridge/        the private contract
internal/typename/      shared type names
v2/go.mod               module flamingo.me/dingo/v2, go 1.27, require flamingo.me/dingo v0.5.0, no replace
v2/                     package dingo: the facade, README.md, doc.go, coverage.min, the test suite
v2/compat/              package compat and its tests
v2/example/             the root example ported to v2; the two directories diff as a migration reference
v2/testdata/            catalogue.txt, compilefail/, moduleidentity/
```

- The `flamingo.me` vanity host already answers `/dingo/v2?go-get=1` with the same go-import
  meta as the root (checked 2026-09-08). Tags for a major-version subdirectory are bare
  (`v2.0.0`, not `v2/v2.0.0`), as the Go module reference specifies for the `vN/` layout.
  `github.com/googleapis/gax-go` runs exactly this layout today: a root module plus
  `v2/go.mod` declaring `github.com/googleapis/gax-go/v2`, bare tags (`v2.24.1`), and pkg.go.dev
  serving both module paths as separate pages (checked 2026-09-09).
- `go.work` is committed so one command from the root covers both modules. In workspace mode
  `./...` does not cross module boundaries; the pattern is `./... ./v2/...` (verified
  2026-09-09). Workspace mode refuses `GOFLAGS=-mod=mod`; the compile-fail driver and the
  as-published job set `GOWORK=off`. The Go module reference allows a committed `go.work` for a
  repository that holds several modules; consumers never see it, since it only applies where a
  command runs.
- Root `go 1.25.8` stays. A module's `go` line must only be at least its dependencies' lines, and
  the root depends on nothing new. Root consumers keep their toolchain.
- `.github/workflows/main.yml`:
  - `tests`: matrix `['1.27', '1.*']`, `go test -shuffle=on -race ./... ./v2/...` in workspace
    mode, so the facade is tested against the engine at the same commit. Shuffle is the cheap
    detector for order dependence through the global `Singleton`.
  - `tests-v0`: matrix `['1.25', '1.*']`, `GOWORK=off`, `go test -race ./...`: the root as its
    consumers build it.
  - `tests-v2-published`: `GOWORK=off`, working directory `v2`, `go test -race ./...`: v2 against
    the root version its `go.mod` requires, fetched from the proxy. Fails when v2 relies on an
    unreleased root change.
  - `coverage`: one profile per module. The v2 profile runs over
    `go list ./... | grep -Ev '/(example|miniexample)$'`, anchored so a future package whose path
    merely contains "example" stays in the profile. When `v2/coverage.min` exists, a step reads
    it and fails below it; until the follow-up commit adds the file (Delivery step 3) the job
    only reports the figure. The root profile is reported as today and not gated.
  - `static-checks`: `go vet ./... ./v2/...`, `gofmt` and `goimports` over the tree, `go generate`
    with a clean diff (still a no-op), and a new root smoke test, `go run ./miniexample 2>&1 |
    grep -q 'here is an example log'`.
  - `go get -v -t -d ./...` becomes `go mod download` in each module; `go get` in workspace mode
    targets one module.
- `.github/workflows/golangci-lint.yml`: matrix over the working directories `.` and `v2`. The
  root `.golangci.yml` applies to both; golangci-lint 2.13.2 is built with Go 1.27 and lints
  generic methods (see Lint compliance).
- Semanticore does not compare tags. It walks the commits reachable from HEAD in committer time
  order and stops at the first one carrying a tag whose name matches `(v?)(\d+).(\d+).(\d+)`;
  that tag is the base version. A commit between HEAD and that tag whose first line reads
  `Release vX.Y.Z` (optionally with ` (#NN)`) overrides the base and ends the scan. The bump is a
  patch, or a minor when the range holds a `feat:` commit. `-major` is false by default and the
  workflow does not pass it, so semanticore never raises the major, and a `BREAKING CHANGE:`
  footer only reaches the changelog. With the default flags it creates the release and its tag at
  the release commit, then opens or updates the `Release vX.Y.Z` pull request. Read 2026-09-09
  from `internal/repo.go:66-95`, `internal/commit.go:93-111` and `main.go:29-40,60-100`.
  Consequences:
  - The root PR merges first. The nearest tag is `v0.4.1` and the PR is a `feat:`, so semanticore
    opens `Release v0.5.0` and tags it on merge.
  - `v2.0.0` is tagged by hand on the v2 merge commit and released with `gh release create`,
    because semanticore never bumps a major. `.github/workflows/semanticore.yml` gets a guard
    that skips the run when the merge commit message contains `[skip release]`, and the v2 PR's
    squash commit carries it, so no `Release v0.4.2` pull request appears before the manual tag.
  - From `v2.0.0` on, every master push finds a v2 tag as the nearest tagged ancestor, so v2
    releases are automatic.
  - Root releases move off master. A hand-made `v0.5.1` tag on a master commit newer than the
    v2 tag would become the nearest tagged ancestor, and the next run would base off `0.5.1` and
    open `Release v0.5.2`, taking the automatic stream back to v0. Instead a `release/v0.x`
    maintenance branch is cut from the `v0.5.0` release commit when v2.0.0 ships. An engine fix
    lands on master first, because workspace-mode CI tests the facade against the engine at the
    same commit, then is cherry-picked to `release/v0.x`, tagged `v0.5.x` there by hand and
    released with `gh release create`. Semanticore only runs on master pushes, and tags on that
    branch are not in master's history, so it never sees them.
  - After the root release, bump `flamingo.me/dingo` in `v2/go.mod` on master (renovate proposes
    it, since it watches every `go.mod`); semanticore then releases the v2 patch that carries the
    engine fix.
  - Semanticore runs on every push to master, so a root-only merge would cut a v2 release that
    carries no change for v2 consumers (`v2/go.mod` still pins the previous root version).
    Root-only squash commits carry `[skip release]` to prevent that.
- `Changelog.md` stays generated and repo-wide.

## Documentation

Root README: a short notice at the top. `flamingo.me/dingo/v2` is the generic API over the same
engine; new code should use it; v0 keeps receiving fixes and stays the engine until the standalone
step. No `Deprecated:` marker on the root package while Flamingo imports it, because staticcheck
would flag every Flamingo module.

`v2/README.md`, the README for the v2 module, is the current README rewritten:

- Every code block is rewritten to v2 syntax by hand. No generator: none of the 22 blocks
  compiles on its own today, ten are one-to-three-line fragments, and comparable libraries
  (wire, fx, samber/do) keep no generated code in their READMEs.
- Each section links to the matching `Example` on pkg.go.dev, which `go test` compiles and runs.
  `readme_test.go` fails when a fenced Go block in `v2/README.md` still uses a v0 call shape such
  as `Bind(new(` or `.To(`, the most likely drift.
- Fix the wrong assignment in the "Requesting injection" block (`m.processor =
  CreditCardProcessor` assigns a type).
- Interception section: interceptors are inherited by child injectors like bindings; a child's
  interceptors apply only to values resolved through that child, and the parent's wrap outermost.
  Say that the wrapper is rebuilt on every resolution, even for a singleton base, so an
  interceptor must not keep state it expects to survive.
- State the Go 1.27 requirement and the import path.
- New section "Using v2 next to v0": the `compat` package, its four functions, its lifetime, and
  the two shapes below.
- New section "Migrating from v0.x" with this table, the bind-time rules, the note that a
  binding has exactly one target, and the two sentinels `ErrInvalidBinding` and
  `ErrPointerToInterface`:

| v0.x | v2 |
|---|---|
| `injector.Bind(new(I))`, `injector.Bind((*I)(nil))` | `injector.Bind[I]()` |
| `injector.Bind(S{})`, `injector.Bind(new(S))` | `injector.Bind[S]()`, or `injector.Bind[*S]()` when binding a `*S` instance |
| `.To(Impl{})`, `.To(new(Impl))` | `.To[Impl]()` |
| `.ToInstance(v)` | `.ToInstance(v)`; `v` must be assignable to `T` at compile time |
| `.ToProvider(fn)` | `.ToProvider(fn)`; signature checked at bind time |
| `injector.BindMulti(new(I))` | `injector.BindMulti[I]()` |
| `injector.BindMap(new(I), "k")` | `injector.BindMap[I]("k")` |
| `injector.Override(new(I), "a")` | `injector.Override[I]("a")` |
| `injector.BindInterceptor(new(I), W{})` | `injector.BindInterceptor[I, W]()` |
| `i, err := injector.GetInstance(new(S)); s := i.(*S)` | `s, err := injector.GetInstance[*S]()` |
| `injector.GetAnnotatedInstance(new(S), "a")` | `injector.GetAnnotatedInstance[*S]("a")` |
| `injector.Bind(v)` for a runtime-typed `v` | keep it on the v0 engine while compat exists; a type switch afterwards, see below |

Using v2 inside a v0 module, the first step any module can take without changing its signature:

```go
func (m *Module) Configure(injector *dingo.Injector) { // v0 signature, unchanged
	i := compat.Injector(injector)
	i.Bind[TransactionLog]().To[DatabaseTransactionLog]()
	i.BindMulti[web.Filter]().To[MetricsFilter]()
}
```

Running a v2 module where a v0 module list is expected, the shape a Flamingo application uses:

```go
flamingo.App([]dingo.Module{
	new(locale.Module),          // v0
	compat.ToV0(new(MyModule)),  // v2
})
```

Runtime-typed values, the Flamingo config loop, once compat is gone:

```go
for k, v := range m.Flat() {
	annotation := "config:" + k
	switch v := v.(type) {
	case nil:
	case string:
		injector.Bind[string]().AnnotatedWith(annotation).ToInstance(v)
	case bool:
		injector.Bind[bool]().AnnotatedWith(annotation).ToInstance(v)
	case float64:
		injector.Bind[float64]().AnnotatedWith(annotation).ToInstance(v)
		if v == float64(int64(v)) { // whole numbers are also bound as int64 and int, as today
			injector.Bind[int64]().AnnotatedWith(annotation).ToInstance(int64(v))
			injector.Bind[int]().AnnotatedWith(annotation).ToInstance(int(v))
		}
	case Map:
		injector.Bind[Map]().AnnotatedWith(annotation).ToInstance(v)
	case Slice:
		injector.Bind[Slice]().AnnotatedWith(annotation).ToInstance(v)
	default:
		panic(fmt.Sprintf("config %q has unsupported type %T", k, v))
	}
}
```

Package documentation: a short `v2/doc.go` with the canonical example, and `v2/compat/doc.go`
with the lifetime statement.

## Flamingo adoption path

Verified against `flamingo.me/flamingo/v3 v3.17.4` and the workspace on 2026-09-09. Flamingo
requires dingo v0.3.0 today; adding v2 anywhere in an application raises the root to v0.5.0
through minimum version selection, which is backward compatible.

0. No Flamingo change. Any module, in Flamingo or in an application, can write its `Configure`
   body against v2 today through `compat.Injector(injector)` while keeping the v0 signature.
   Applications can list v2 modules with `compat.ToV0`. Nothing in Flamingo's API moves.
1. Flamingo v3 minor releases migrate Flamingo's own module bodies the same way, file by file, in
   any order. Exported module types keep implementing the root's `Module`, so no application
   changes. Flamingo may add v2 twins of its four helpers (`BindEventSubscriber`,
   `BindTemplateFunc`, `web.BindRoutes`, `cmd.Run` in `framework/flamingo/events.go`,
   `framework/flamingo/functions.go`, `framework/web/registry.go`, `framework/cmd/module.go`);
   each is a one-line generic rewrite because each passes `new(Iface)` as a type carrier today.
   Naming them is Flamingo's decision. `framework/config/area.go` (child injector per area, the
   runtime-typed config loop, `Bind(Area{}).ToInstance(area)`) and the `Inspector` in `app.go`
   stay on v0 unchanged.
2. Flamingo v4 flips the signatures: `NewApplication`, `App`, `WithCustomLogger`,
   `config.NewArea`, `config.Area`, `GetInitializedInjector`, `config.TryModules` and every
   `Configure` take v2 types. Third-party v0 modules keep working through `compat.FromV0` at the
   list site. The config loop calls `compat.Engine(injector).Bind(v)` until step 3. Ecosystem
   modules (pugtemplate, graphql, form, httpcache, opentelemetry, commerce, commerce-contrib,
   redirects) migrate on their own schedule, wrapped until they do.
3. Once every maintained module is on v2, Flamingo drops compat: the config loop becomes the
   type switch above, `FromV0` wrappers go, and Flamingo requires only `flamingo.me/dingo/v2`.
   This is the trigger for dingo's standalone step.

Call shapes needing care, from the Flamingo audit:

- `core/silentzap/module.go`: `registry := new(LoggingContextRegistry);
  injector.Bind(registry).AsEagerSingleton()` and `BindEventSubscriber(injector).To(registry)`
  use the pointer as a type carrier only. They become `Bind[LoggingContextRegistry]()
  .AsEagerSingleton()` and `.To[LoggingContextRegistry]()`; the local variable goes.
- `framework/config/area.go`: `Bind(Area{}).ToInstance(area)` with `area *Area` becomes
  `Bind[*Area]().ToInstance(area)`; the key stays `Area` because v2 strips one pointer level.
- `flamingo-om3/searchperience/module.go`: eight `BindMap(search.Endpoint(nil), key)` calls
  become `BindMap[search.Endpoint](key)`; the func type is the key as is, no pointer stripping.

## Path to a standalone v2

The facade is the migration vehicle, not the end state. When the ecosystem no longer imports the
root module, the engine moves into the v2 module and the root is frozen. This section seeds the
spec for that step, written when the preconditions hold; it is not part of this implementation.
It is recorded now so that nothing in the facade design forecloses it.

Preconditions, all observable:

- Flamingo's latest major requires only `flamingo.me/dingo/v2` and imports neither
  `flamingo.me/dingo` nor `flamingo.me/dingo/v2/compat`.
- Every maintained module in the workspace and the i-love-flamingo organization is on v2. A
  `go.mod` search for `flamingo.me/dingo ` (with the trailing space) across both finds only
  frozen or archived repositories.
- The removal of compat has been announced in two shipped v2 minor releases, in `Changelog.md`,
  in the compat package documentation with a `Deprecated:` line naming the target version, and in
  the v2 README. If the notice first ships in `v2.3.0`, the earliest release that may delete
  compat is `v2.5.0`.

Versioning decision: compat removal ships in a v2 minor release. Removing a package is a breaking
change under strict semver, but a new import path `flamingo.me/dingo/v3` for a package nobody
imports any more would put the whole ecosystem through a second path change for nothing. The
compat package documentation states from day one that it is a migration aid excluded from the
compatibility promise and removed after announcement; projects that still need it pin the last
v2 minor that ships it. The v2 core API does not change in that release.

Steps, each its own PR against master:

1. Announce. `v2/compat/doc.go` gets `Deprecated: removed in v2.N.0`, the root README notice says
   the engine moves to v2 in the same release, `Changelog.md` carries the note through
   semanticore's release notes.
2. Move the engine. `dingo.go`, `binding.go`, `scope.go`, `inspect.go`, `module.go` and
   `internal/typename` are copied into the v2 module as unexported implementation, and the facade
   collapses onto them:
   - `Injector` becomes the engine struct itself; the facade slot, the engine adapter,
     `reflect.New` type carriers and the result adaptation copy go away.
   - The unexported `binding` struct holds today's fields plus the three flags; `Binding[T]`
     wraps a `*binding` in a named field and writes the fields directly.
   - `Instance` and `Provider` become `instance` and `provider`; `Provider.Create` becomes
     `provider.create`.
   - `getInstance` takes a `reflect.Type`; the `interface{}` type switch goes away.
   - `NewInjector` and `Child` bind themselves with `Bind[*Injector]().ToInstance(injector)`.
     The root's self-binding disappears with the root.
   - `Scope`, `SingletonScope`, `ChildSingletonScope` become v2 types with the same method sets.
     Every caller compiles unchanged: `In` and `BindScope` take the interface, and the two scope
     types are only ever passed as `Scope`.
   - `moduleKeyOf`, `modGraph`, `TryModule` and the tracing switches move as they are;
     `WrappedModule` handling is deleted with the last adapter.
   - `Inspector` and `Inspect` move as they are.
3. Delete `v2/compat`, drop `require flamingo.me/dingo` from `v2/go.mod`, drop the README
   sentence that points at the root `example/` as the migration twin, and remove the CI jobs that
   only exist for the shared engine (`tests-v2-published`). `go.work` stays while both modules are
   in the repository.
4. Freeze the root. Root `doc.go` gets `Deprecated: use flamingo.me/dingo/v2`; the root README
   says fixes go to v2 only; the bridge package, the facade slot and the `Innermost` call in
   `moduleKeyOf` are removed from the root, since nothing installs the hooks any more. The root
   keeps building and its suite keeps running in CI so the frozen version stays green on new Go
   releases.
5. Release. Semanticore tags the v2 minor from master. The root gets one last tag on
   `release/v0.x`, carrying the deprecation notice.

Acceptance test for step 2: the v2 test suite passes without edits, except that the interop tests
in `compat/` are deleted with the package and the IDs excused to the root suite (`R-25`, `R-27`,
`DUP-03`, the white-box halves of `M-03` and `M-05`) come back into v2 as white-box tests in
`package dingo`, rewritten from the root files they point to. The same PR deletes their excuse
lines from `testdata/catalogue.txt`, so the catalogue gate fails until the rewritten tests exist.

Design constraints that keep this step cheap, honored by the facade design above:

- The v2 public API mentions no root identifier except through aliases that become v2 types
  with identical method sets, and shared sentinel values that become v2 values.
- All coupling goes through `internal/bridge` and the `compat` directory; both are deleted, not
  refactored.
- The v2 suite is black box against the v2 API only, so it survives the engine move unchanged.
- The behavior IDs describe engine behavior as observed through v2, so the catalogue is the
  checklist for "nothing changed" after the move.

## Testing

The v2 suite is written from scratch. No v0 test is ported; the v0 suite serves as a checklist of
behavior and stays in the root module untouched. Every behavior has an ID in the appendix (159
items across bind-time checks B, key normalization K, resolution R, multibindings MB, scopes S,
child injectors C, overrides OV, duplicates DUP, interceptors I, modules M, Inspector INS,
GetInstance G, interop X). Three rules:

1. Black box only: `package dingo_test` and `package compat_test`, exported API only. Engine
   internals with no observable effect through the v2 API are the root suite's job; the
   catalogue excuses their IDs with the root test that pins them.
2. One rule per test name, one subject per file, so `go test -v` reads as dingo's rule list.
3. Every test names the regression it catches. Every ID is referenced by a test or excused with a
   reason, enforced by a gate that runs under `go test`.

### File layout

All paths below are under `v2/`. Reading order matches the README. Package `dingo_test` unless
noted.

| File | Purpose | IDs |
|---|---|---|
| `fixtures_test.go` | Fixture types grouped by role, each group commented with the rules it proves. No tests. | |
| `helpers_test.go` | The six helpers and three tests pinning their contracts. | |
| `example_{hello,provider,annotated,multibinding,interception,child}_test.go` | Six whole-file Examples, one per README section. | |
| `example_migration_test.go` | v0/v2 side by side, plus the Flamingo config-loop shape. | |
| `bind_key_test.go` | What `T` means; the shapes all four entry points reject; the zero `Binding[T]` and the zero `Injector`. | K-01..K-09, B-01..B-04, B-39, B-40 |
| `bind_target_test.go` | `To`, `ToInstance`, `ToProvider`, exactly one target. | B-05..B-22 |
| `bind_attribute_test.go` | `AnnotatedWith`, `In`, `AsEagerSingleton`, free method order, `Override` preset. | B-23..B-30 |
| `bind_errors_test.go` | The panic itself, the sentinel, the message shape. | B-37, B-38 |
| `resolution_test.go` | `GetInstance[T]`, `GetAnnotatedInstance[T]`, adaptation, failures, `map:`, precedence over multi and map bindings, the config-loop shape. | R-01..R-06, R-09, R-13..R-16, G-01..G-04, K-10, K-11 |
| `provider_test.go` | Providers, generated `…Provider` types, the dropped error result. | R-07, R-08, R-10..R-12, R-11a, R-28..R-30 |
| `injection_test.go` | Inject tags, optional, `Inject` methods, `RequestInjection`, `*Injector`. | R-17..R-24, R-31 |
| `multibinding_test.go` | `BindMulti`, `BindMap`. | MB-01..MB-14 |
| `scope_test.go` | Scope identity matrix, custom scopes, eager singletons, concurrency. | S-01..S-08 |
| `child_test.go` | Parent, child, sibling matrix; inherited interceptors; the global `Singleton`. | C-01..C-04 |
| `override_test.go` | `Override[T]`, unknown and multibinding overrides, duplicate detection. | B-31, B-32, OV-01..OV-04, DUP-01, DUP-02 |
| `interceptor_test.go` | Chain order, field-0 write, injection into the interceptor, the wrapper per resolution, validation. | I-01..I-06, B-33..B-36 |
| `module_test.go` | `Module`, `ModuleFunc`, `Depender`, dedup, identity, `InitModules`, `TryModule`. | M-01..M-08 |
| `inspect_test.go` | The four `Inspect*` callbacks. | INS-01..INS-06 |
| `compilefail_test.go` | Drives the compile-fail corpus. | B-12; B-05 again from the compile side |
| `readme_test.go` | Guard: fenced Go blocks in `v2/README.md` use v2 call shapes. | |
| `catalogue_test.go` | Gate: every ID referenced or excused. | |
| `compat/compat_test.go` (package `compat_test`) | Two APIs on one engine: facade identity, adapters both ways, module identity, injection timing, both self-bindings, shared scopes and interceptors, checks through v0 entry points, the config loop, `Inspect`, shared sentinels, shared multibindings, engine wording, injection failure. | X-01..X-19 |
| `compat/example_test.go` (package `compat_test`) | Two whole-file Examples: v2 inside a v0 module, a v2 module in a v0 module list. | |

Engine-internal behavior stays in the root module: the white-box tables for module sorting and
cycles (`module_test.go`), `binding.equal` (`binding_test.go`), circular tracing
(`circular_test.go`) and injection tracing (added by the root PR). The catalogue excuses `M-03`
and `M-05`'s white-box halves, `DUP-03`, `R-25` and `R-27` with those file names; their public
views are covered in v2.

```
testdata/moduleidentity/{commerce,om3}/cart/   fixture packages with same-named modules, imports on /v2
testdata/catalogue.txt                         one behavior ID per line; excused IDs carry a reason
testdata/compilefail/go.mod                    module flamingo.me/dingo/v2/testdata/compilefail; go 1.27;
                                               replace flamingo.me/dingo/v2 => ../.. and flamingo.me/dingo => ../../..
testdata/compilefail/cases/<name>/main.go      one package per case, `// want: <substring>` first line
```

### Naming and documentation in tests

- Top level `Test<Surface>_<Rule>`, the rule a verb phrase in the indicative:
  `TestBind_RejectsPointerToInterface`, `TestSingleton_IsSharedProcessWideAcrossInjectors`. Never
  "works", "ok", "basic", "cases".
- Subtests and table cases are one lowercase sentence, so the `-run` path reads as a sentence:
  `TestChild_Visibility/a_child-only_binding_is_invisible_to_the_parent`.
- Doc comment per test: the rule, then `// Covers <IDs>.`, then `// Catches: <regression>`, then
  `// v0:` only where behavior changed. A test whose `Catches` line cannot be written is not worth
  writing. IDs live in comments, never in names.
- Standout decisions get a named top-level test, not a table row: the one-target rule (B-21),
  eager after another scope (B-27), self-binding (B-07), nil instance (B-10), the dropped provider
  error (R-12), the `map:` prefix (G-03), the global `Singleton` (C-04), inherited interceptors
  (C-03), the per-resolution interceptor wrapper (I-05), the zero `Binding[T]` (B-39), the zero
  `Injector` (B-40), module identity through adapters (X-06), injection before `Configure` for
  adapted modules (X-08).
- Fixtures are named for their role, never for a test: `greeter`, `ptrOnlyGreeter`,
  `countingScope`, `plainGreeterFunc`.

### Helpers and fixtures

Six helpers in `helpers_test.go`, `t.Helper()` first, kept small on purpose. `dingo.ModuleFunc`
is the module wrapper; no helper duplicates it.

```go
func newInjector(t *testing.T, mods ...dingo.Module) *dingo.Injector
func childOf(t *testing.T, parent *dingo.Injector) *dingo.Injector
func bindErr(fn func(*dingo.Injector)) error                       // through dingo.TryModule
func bindPanic(t *testing.T, fn func(*dingo.Injector)) error       // recovers the raw panic value
func requireInvalidBinding(t *testing.T, err error, parts ...string)
func get[T any](t *testing.T, injector *dingo.Injector) T
```

`newInjector` gives each test a private singleton scope. `dingo.Singleton` is a package-level
object registered by every injector, so tests would otherwise share one cache under `t.Parallel`.
Construction is two-phase because `NewInjector(mods...)` would build eager singletons against the
global scope:

```go
injector, err := dingo.NewInjector()               // no modules: nothing constructed yet
require.NoError(t, err)
injector.BindScope(dingo.NewSingletonScope())      // replaces the *SingletonScope registration
require.NoError(t, injector.InitModules(mods...))  // eager singletons build here, isolated
```

`childOf` calls `parent.Child()` and registers the parent's private singleton scope on the child,
because `Child()` creates a fresh engine that registers the global one. Parent and child thus
share singletons exactly as in production, only privately. The helpers keep the injector-to-scope
mapping in a `sync.Map`. Three tests pin these contracts before anything depends on them: two
roots do not share singletons, a child shares its parent's, and `requireInvalidBinding` asserts
`errors.Is(err, ErrInvalidBinding)` and the `dingo: ` prefix. `compat_test.go` has its own copy
of `newInjector` for the v0 side, built on the root API the same way.

`bindErr` takes the user's route through `TryModule`. `TryModule` disables eager singletons, so an
accepted table row proves bind-time acceptance only; eager construction is proved in
`scope_test.go` by a side effect observed before any `GetInstance` call.

Fixtures: `greeter`, `politeGreeter`, `loudGreeter`, `ptrOnlyGreeter` (B-06), `counter` (R-02,
R-18), `greeterProvider`, `greetersProvider`, `greeterMapProvider` (R-28..R-30),
`plainGreeterFunc` (R-08), `countingScope` (S-03), `recordingInterceptor1`, `recordingInterceptor2`
(I-01), `spyInspector` (INS; `Inspect` ranges over maps, so records are sorted before comparison).
`compat_test.go` adds `v0Module` and `v2Module` pairs that record their `Configure` calls and
carry `inject`-tagged fields of both injector types.

### Patterns per category

#### Bind-time rules

Table over `{name, configure, wantParts, check}`. Empty `wantParts` means
the binding must be accepted, and an accepted row also asserts that it resolves, so the table
teaches more than its rejections. Every rejection sits next to its nearest legal neighbour. The
key checks (B-01..B-04) are parameterized over `Bind`, `BindMulti`, `BindMap` and `Override`, with
the call name in `wantParts`.

```go
// TestBind_RejectsIllegalTypeParameters documents which T the four entry points accept.
// Covers B-01..B-04, K-08, K-09.
// v0: Bind[*Iface] was not expressible; Bind(new(*Foo)) silently made an unresolvable binding.
// Catches: a check that strips pointers before testing for interface kind.
func TestBind_RejectsIllegalTypeParameters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		configure func(*dingo.Injector)
		wantParts []string // empty: the binding must be accepted
		check     func(*testing.T, *dingo.Injector)
	}{{
		name:      "a pointer to a struct is accepted and keys on the struct",
		configure: func(i *dingo.Injector) { i.Bind[*counter]().ToInstance(&counter{n: 1}) },
		check: func(t *testing.T, injector *dingo.Injector) {
			t.Helper()

			assert.Equal(t, 1, get[counter](t, injector).n)
		},
	}, {
		name:      "a pointer to an interface is rejected, as it is at injection sites",
		configure: func(i *dingo.Injector) { i.Bind[*greeter]() },
		wantParts: []string{"Bind[", "flamingo.me/dingo/v2_test.greeter", "pointer to interface"},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := bindErr(tt.configure)
			if len(tt.wantParts) == 0 {
				require.NoError(t, err)
				tt.check(t, newInjector(t, dingo.ModuleFunc(tt.configure)))

				return
			}

			requireInvalidBinding(t, err, tt.wantParts...)
		})
	}
}
```

Expected substrings carry the full import path for test-package types and are substrings only,
never whole messages.

#### The panic path itself

`bind_errors_test.go` owns B-37 and B-38 as rules:
`TestBind_PanicsWithAnErrorWrappingErrInvalidBinding` uses `bindPanic` and `errors.Is` outside
`TryModule`; `TestTryModule_ConvertsBindPanicsToErrors` asserts the conversion;
`TestBindMessages_HaveTheDocumentedShape` walks one rejection per entry point and asserts the
whole `dingo: <call>: <reason>` shape. Without it a dropped prefix would surface as thirty
scattered failures instead of one.

#### Resolution

Table over `{name, configure, check}`, same loop as above without `wantParts`.
`T` cannot be table data, so the typed call lives in `check` and nothing else does. Rows follow
the spec's resolution table in order, so table and spec diff by eye. The `Catches` line of the
adaptation test names the aliasing bug: a value-typed `T` handing out the bound pointer's target,
so one caller mutates another's instance.

#### Scopes

Identity via `assert.Same` for pointers and a construction counter for values, never
`==` on `any`. The identity matrix is `{Singleton, ChildSingleton, custom registered, custom
unregistered}` × `{same injector twice, two children, two roots, same type different annotation}`.
Concurrency: `const concurrentResolvers = 100` goroutines, `sync.WaitGroup`, an `atomic.Int64`
construction count of 1. No sleeps and no `testing/synctest`: nothing here is clock-driven.

#### Child injectors

One matrix `{bindingKind, boundAt, observedFrom}`: kind covers single,
multi, map, interceptor, scope and the `*Injector` self-binding; `observedFrom` includes a sibling
child. C-04 is the deliberate exception: it calls `dingo.NewInjector` twice with no scope swap, is
not parallel, and says in a comment that the helper would hide the sharp edge under test.

#### Interceptors

Order is asserted as a recorded `[]string`, so a failure names which interceptor ran where. I-05
carries the wrapper-per-resolution rule: an intercepted singleton resolved twice yields two
wrappers, so the test asserts `assert.NotSame` on the wrappers and `assert.Same` on the value in
field 0, pinning the rebuild and the shared inner instance together.

#### Inspector

`spyInspector` records into typed slices.

#### Modules

Public surface in `module_test.go`; the `modGraph` and `qualifiedTypeName` tables stay in the root
module. `TryModule`'s two panic paths are two named cases, including the exact `%q` rendering of
`panic(42)`.

#### Interop

Every case in `compat_test.go` runs both directions where both exist: a v0 module on
a v2 injector and a v2 module on a v0 injector, as two subtests of one rule. Module identity is
asserted by counting `Configure` calls per module value, never by comparing adapter values.
Injection timing is asserted from inside `Configure`: the module records what its injected
fields held when `Configure` ran. Facade identity uses `assert.Same`.

`require` for preconditions, `assert` for outcomes. `errors.Is` for sentinels; `errors.AsType[E]`
once v2 has a typed error, which the v2 module can use at `go 1.27` and the root cannot at
`go 1.25.8`. Bare `assert.Error` and bare `assert.Panics` are banned.

### Hermeticity, parallelism, race

- Every test and subtest calls `t.Parallel()`, outer first, except C-04.
- No test calls `dingo.NewInjector` directly except C-04 and the tests asserting what
  `NewInjector` binds.
- No v2 test touches the tracing switches; they have no off switch and are pinned in the root
  module.
- `go test -shuffle=on -race ./... ./v2/...` locally and in CI.
- Counters are `atomic.Int64`, synchronization is `sync.WaitGroup`, never `time.Sleep`.

### Compile-fail corpus

A nested module replaces the `//go:build compilefail` idea from the first draft: a separate
`go.mod` under `testdata/` keeps the corpus out of the parent module's build, vet, lint and
dependency graph, and no tag can pull it in by accident. One package per case, so one deliberate
error cannot mask another. The driver locates `go` with `exec.LookPath`; when that fails it skips
locally and fails when `CI` is set. Per case it runs, with `exec.CommandContext(t.Context(), ...)`,
`cmd.Dir = testdata/compilefail` and `GOWORK=off` in the environment, because the corpus module is
not in `go.work` and its two `replace` directives must apply:

```
go build ./cases/<name>
```

and requires a non-zero exit plus the `// want:` substring and the file name in the output.
Substrings are short and stable (`cannot use`, `not enough type arguments`); a Go release that
rewords one is a one-line fix.

| Case | Documents |
|---|---|
| `to_instance_wrong_type` | `Bind[greeter]().ToInstance(42)`: v0's runtime panic is now a compile error (B-12) |
| `get_instance_wrong_assignment` | assigning the result of `GetInstance[greeter]()` to a `string` |
| `bind_missing_type_argument` | `Bind()` without `[T]` |
| `to_missing_type_argument` | `.To()` without `[U]` |
| `interceptor_one_type_argument` | `BindInterceptor[greeter]()` |
| `to_unrelated_type_compiles` | `Bind[A]().To[B]()` with unrelated `B` **builds**; asserts success and points at the bind-time test |

The last case documents the compile-time/bind-time boundary from both sides and proves that the
fixture module's two `replace` directives still resolve, which renovate cannot see.

### Examples and the README

Six whole-file Examples, one per README section, plus `example_migration_test.go` with
`ExampleInjector_Bind_migration` (the v0/v2 pair) and `ExampleInjector_Bind_config` (the Flamingo
config-loop shape), plus two in `compat/example_test.go`. A file with exactly one `Example`, no
`Test` functions and its own package-level declarations renders whole on pkg.go.dev; each file
owns its exported fixtures (`Greeter`, `PoliteGreeter`) with names disjoint across the files.
Every Example ends in `// Output:`, or `// Unordered output:` where a map is printed. The README
links to them (see Documentation).

```go
package dingo_test

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

### Coverage and gates

- Statement coverage of the v2 module: `go test -shuffle=on -race -covermode=atomic
  -coverprofile=coverage.txt` over `go list ./... | grep -Ev '/(example|miniexample)$'` inside
  `v2/`, then `go tool cover -func`. The root module keeps its ungated report.
- No threshold on day one; every number so far is a guess. Land the suite, read the real figure,
  commit it to `v2/coverage.min` in a follow-up. The facade is thin and every branch is a check
  with a test, so expect a figure well above 90%.
- Once `v2/coverage.min` exists, the coverage job fails below it and prints the delta. Lowering
  the file takes an explicit commit that says why.
- Branches left uncovered on purpose carry `// coverage: unreachable through the public API` at
  the production site.
- Behavior gate: `catalogue_test.go` reads `testdata/catalogue.txt`, scans `*_test.go` under the
  v2 module including `compat/` for `Covers` IDs, and fails on an ID with neither a reference nor
  a recorded reason. Excused today: R-26 (a cycle without tracing overflows the stack; a test
  binary cannot catch it), M-08's add-failure branch (dead code, `Depends` cannot fail), I-04
  (restated by B-33..B-36), and the five engine-internal IDs pinned in the root module.
- `v2/example` has no tests, is compiled by `go test ./...` (a package without tests still fails
  the run on a compile error) and stays out of the profile.

### Lint compliance

`linters.default: none` with `new: false`, so the whole tree must be clean.

Checked before writing the suite (2026-09-09): a spike with generic methods (`Bind[T]`, `To[U]`,
`GetInstance[T]`), type-argument-only calls, a bind-time panic wrapping a sentinel, a table of
`func(*Injector)` closures and a generic test helper `get[T any]`, in a library package plus an
external `_test` package, passes `go vet` and `go test` on Go 1.27.1 and reports 0 issues under
golangci-lint 2.13.2 with the repository's own `.golangci.yml`. The linters do read those files:
removing one `t.Helper()` and one `t.Parallel()` produced `thelper`, `paralleltest` and `wsl_v5`
findings on the generic test file.

- `paralleltest`, `tparallel`: one non-parallel island, C-04, with both names in `nolint` and a
  reason (`nolintlint` requires it).
- `testpackage`: `dingo_test` and `compat_test` everywhere; no white-box file in v2.
- `thelper`: `t.Helper()` first in every helper and every `check` closure taking `*testing.T`.
- `wsl_v5`: blank line before `if` or `for` after a statement, branches of at most 4 lines;
  branching lives in table data, which also keeps `cyclop`, `gocognit` and `nestif` quiet.
- `mnd`: named constants for goroutine and iteration counts. `errcheck`: every error checked.
- `usetesting`: `exec.CommandContext(t.Context(), ...)` in the compile-fail driver.
- `unconvert`: today's finding at the root's `module_test.go:206` is not reproduced.
- Excluded for `_test.go` by the repo config: `varnamelen`, `err113`, `forcetypeassert`,
  `goconst`, `wrapcheck`, `containedctx`. `dupl`, `funlen`, `exhaustruct` are not enabled.

### What the v2 suite re-derives from the root suite

The root files stay as they are. This table records what each one proves, so the v2 suite does
not lose it.

| Root file | v2 counterpart | Must not be lost |
|---|---|---|
| `binding_test.go` | `bind_target_test.go`; `TestBinding_equal` stays root (DUP-03) | The three mismatch panics become bind-time rules. |
| `circular_test.go` | none; R-25 excused to the root | Direct self-cycle panics with tracing on; a `…Provider`-typed self-reference injects fine and panics only when called. |
| `dingo_child_test.go` | `child_test.go` | Parent-bound type resolvable from the child. The inventory flags that it may prove provider laziness, not fallback: re-derive as two tests. |
| `dingo_setup_test.go` | `injection_test.go` | `Inject(...)` with a positional argument and an anonymous annotated-struct argument. |
| `dingo_test.go` | split across `resolution_test.go`, `injection_test.go`, `override_test.go`, `child_test.go` | `TestBoundToNothing`; `TestOptional`'s two tag spellings; `TestInjectStructRec` (`ErrInvalidInjectReceiver`); `TestInjectionOfInterfacePointer`; `TestInjector_InitModules` error propagation; the two-hop interface chain; the annotated provider; unbound `int` is `0`; nil-receiver `Child()`; `TestOverrides`' explicit `InitModules`. |
| `module_identity_test.go` | `module_test.go` with the fixture packages under `v2/testdata/moduleidentity` | All three same-name-different-package cases. |
| `module_test.go` | `module_test.go` (public views of M-03, M-05); the `modGraph` and `qualifiedTypeName` tables stay root | The cycle case with `ErrModuleCycle` and its path; `ModuleFunc` closure identity. |
| `multi_dingo_test.go` | `multibinding_test.go` | Registration order; parent-then-child merge order; per-entry scope for slices and maps; the mixed target set; `inject:"map:key"`. Six near-duplicates become two tables with an access-pattern axis. |
| `scope_test.go` | `scope_test.go` | Exactly-once construction under concurrency for both scopes; `ChildSingleton` stable within one child; the transitive `A→B→C` case. |

## Delivery

1. Root PR, dingo branch `feat/v2-bridge` from `master`: `internal/bridge`, `internal/typename`,
   the facade slot, `moduleKeyOf` and `InitModules` looking through adapters,
   `ErrPointerToInterface`, the wrapped `ErrModuleSort` cause, the tracing test, the
   `TestDingoCircula` rename, the `// coverage:` comments on `InitModules`' add-failure branch and
   on the sort fallback. Conventional commit `feat:`; semanticore releases it as `v0.5.0`. Draft
   PR against `master`, no reviewers assigned; body: what the bridge is for, the behavior-neutral
   claim and how it was checked.
2. v2 PR, dingo branch `feat/v2-generic-api` from `master` after `v0.5.0` is on the proxy: the
   `v2/` module requiring `v0.5.0`, `go.work`, the workflow changes, the semanticore guard, the
   root README notice, `v2/README.md`. Draft PR against `master`, no reviewers assigned; body:
   summary, the migration table, the compat lifetime, the manual `v2.0.0` tag step with
   `[skip release]` in the squash commit. The main commit carries a `BREAKING CHANGE:` footer
   line describing the API, for the changelog.
3. After the v2 merge: tag `v2.0.0` on the merge commit, `gh release create v2.0.0`, then commit
   `v2/coverage.min` with the measured figure. From that commit on the coverage job gates.
   Cut `release/v0.x` from the `v0.5.0` release commit in the same session, so the first root fix
   after v2 has a branch to be tagged on and nobody tags v0 on master by reflex.
4. This spec, its appendix and the implementation plan live in the workspace repo on branch
   `docs/dingo-v2-generic-api` under `docs/superpowers/`.

## Review decisions

1. 2026-09-09: Parent interceptors keep wrapping child-resolved values. Guice rule, needed by
   Flamingo's per-area child injectors; no call site in the workspace depends on the opposite.
   Stated in the README and pinned by tests that assert the order. The evidence for the current
   behavior being deliberate is thin: parent chaining arrived as a three-line diff in commit
   088ee7f of 2017-05-19, whose subject is "mov pug_template from framework to core", with no
   test and no note, and v0 has never had a test covering interceptors together with `Child()`.
   The decision therefore rests on Guice's documented rule and on Flamingo's need, not on a
   record of intent. v2 freezes it as a contract and says so in the README.
2. 2026-09-09: No README generator. Blocks are rewritten by hand, sections link to the Examples,
   and `readme_test.go` rejects v0 call shapes.
3. 2026-09-09: `ErrPointerToInterface` is exported with the message
   `pointer to interface is not allowed`.
4. 2026-09-09: Flamingo compatibility is a must, so v2 is a facade over the v0 engine in a `v2/`
   module of the same repository, with `compat` adapters both ways, and the path to a standalone
   v2 is documented above. Evidence: a prototype facade of 540 lines with 17 interop tests passing
   under `-race` against the unchanged engine plus one identity hook; Flamingo's 245 `Configure`
   implementations and eight OSS modules; semanticore's version detection, read from its source
   (see review decision 7); the internal-package rule verified across modules.
5. 2026-09-09: compat removal ships in a v2 minor, at the earliest two minors after the one that
   first carries the deprecation notice (notice in `v2.3.0`, removal in `v2.5.0`); compat is
   excluded from the compatibility promise from day one.
6. 2026-09-09: injection of an adapted module happens in the engine's `InitModules`, which looks
   through adapters with `bridge.Innermost`. Injection failures take the native error path; no
   injection hook in the bridge.
7. 2026-09-09: root v0.x releases move to a `release/v0.x` maintenance branch once `v2.0.0`
   exists. Semanticore bases its version on the first tag it meets walking back from HEAD, not on
   the highest tag, so a root tag on master would take the automatic stream back to v0. See
   "Versioning, toolchain, CI, release".
8. 2026-09-09: the v0.5.0 PR also wraps the sort error in `ErrModuleSort`
   (`module.go:150`), which today discards it. Second and last carve-out from "identical to
   v0.4.1", together with the pointer-to-interface message. `errors.Is` is unaffected either way;
   the message gains the cause.
9. 2026-09-09: the per-resolution interceptor wrapper is kept, stated under "Kept v0 behaviors
   worth knowing" and in the README, and pinned by I-05, which already existed and now also
   asserts that the two wrappers differ while the value in field 0 is the same instance.

## Out of scope, possible follow-ups

- `BindType(reflect.Type) *Binding[any]` as an escape hatch (additive, 2.x).
- Typed provider variants such as `ToProvider0[F ~func() T | ~func() (T, error)]` (additive, 2.x).
- A go/ast rewrite tool for the migration.
- Propagating provider `(T, error)` errors: `feat/to-provider-error-handling`.
- A per-injector `Singleton` scope instead of the process-wide object (C-04): a behavior change,
  its own decision.
- Caching interception: today every resolution of an intercepted type rebuilds the wrapper chain
  and walks to the root injector (I-05). Caching it would be a behavior change, since interceptor
  state would start surviving between resolutions, so it needs its own decision and a measurement
  first.
- Synchronizing the engine's interceptor map, which would make `BindInterceptor` after
  `InitModules` safe. Also a behavior change and its own decision.
- The Flamingo, flamingo-commerce, flamingo-om3 and application PRs that walk the adoption path.
- The standalone step itself: its own spec when the preconditions hold, seeded by the section
  above.
