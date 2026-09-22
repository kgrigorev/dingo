# Appendix: dingo v2 test-case catalogue

Companion to `2026-09-08-dingo-v2-generic-api-design.md`. Revised 2026-09-14 with the spec review
applied. Lists every planned test case of the dingo v2 suite with the behavior IDs it pins, one
section per test file in the order of the spec's file layout. The Case column is the behavior
statement; the IDs exist so the catalogue gate in `catalogue_test.go` can prove that nothing on
this list was dropped. Every ID has exactly one statement: the Case text of the first row that
lists it in its Covers column, or its row under "IDs without a case of their own". That statement
is copied into `testdata/catalogue.txt` as `ID<TAB>statement`, and the gate fails on an ID whose
statement is empty, so an ID is never defined by its prefix alone. `(decision)` marks a case that
pins surprising v0 behavior kept on purpose, or a deliberate v2 change. The suite lives in the
`v2/` module of the dingo repository; the `compat/` files are part of it. Engine-internal
behavior stays pinned by the root module's own suite, see "Pinned in the root module". The suite
runs in CI as described in the spec's "Versioning, toolchain, CI, release" section: workspace
mode with `-shuffle=on -race`, plus the as-published job for the `v2/` module.

ID legend:

| Prefix | Area |
|---|---|
| B | Bind-time checks |
| K | Key normalization (`T` shapes) |
| R | Resolution |
| MB | Multibindings and map bindings |
| S | Scopes |
| C | Child injectors |
| OV | Overrides |
| DUP | Duplicate binding detection |
| I | Interceptors |
| M | Modules |
| INS | Inspector callbacks |
| G | `GetInstance`/`GetAnnotatedInstance` (generic-signature-specific) |
| X | Interop: v0 and v2 modules on one engine, package `compat` |

## fixtures_test.go

No test functions; fixture declarations only, grouped by the rule each group exists to prove.

| Fixture group | Case | Covers | Note |
|---|---|---|---|
| `greeter`/`politeGreeter`/`loudGreeter`/`ptrOnlyGreeter` | value- and pointer-receiver implementations of one interface | — | supports B-06 |
| `counter` | a struct with a mutable field, for the `GetInstance[Service]()` copy assertion | — | supports R-02; never used as a value-typed inject field, which the engine rejects (R-32) |
| `label` | a named `string`, a non-struct value type for value-field injection | — | supports R-17, R-18 |
| `greeterProvider`/`greetersProvider`/`greeterMapProvider` | `Provider`-suffixed func types, single/slice/map | — | supports R-28..R-30, MB-15, MB-16 |
| `plainGreeterFunc` | a `func() greeter` with no `Provider` suffix | — | supports R-08 |
| `countingScope` | a user `Scope` with a construction counter | — | supports S-03 |
| `recordingInterceptor1`/`recordingInterceptor2` | interceptors that append to a shared `[]string`; field 0 is exported (`Wrapped greeter`) | — | supports I-01; an unexported field 0 is rejected at bind time (B-34a), so every interceptor fixture exports it |
| `uniqueSingletonFixture`, one per Example that scopes | types used under the global `Singleton`/`ChildSingleton` outside `newInjector` | — | supports C-04 and the Examples; unique so no parallel test reads their cached instance |
| `spyInspector` | an `Inspector` recording into typed slices | — | supports INS-01..INS-06; ranges over maps, sort before comparing |

## helpers_test.go

Six helpers (`newInjector`, `childOf`, `bindErr`, `bindPanic`, `requireInvalidBinding`,
`get[T]`), plus four tests pinning their own contract before anything depends on them.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestHelper_PrivateSingletonScopeIsolatesTwoRoots` | two `newInjector` roots do not share a singleton cache | — | pins `newInjector`'s `Singleton` swap |
| `TestHelper_PrivateChildSingletonScopeIsolatesTwoRoots` | two `newInjector` roots do not share a child-singleton cache for a root-level `In(ChildSingleton)` binding | — | pins `newInjector`'s `ChildSingleton` swap; `ChildSingleton` is a package-level object and only `Child()` replaces it, so without the swap S-02, MB-13 and B-26 are order-dependent under `-shuffle=on` |
| `TestHelper_ChildSharesParentsPrivateSingletonScope` | `childOf` re-registers the parent's private `Singleton`, not the global one | — | pins `childOf`'s scope swap |
| `TestHelper_RequireInvalidBindingAssertsSentinelAndPrefix` | `requireInvalidBinding` checks `errors.Is(err, ErrInvalidBinding)` and the `"dingo: "` prefix | — | pins the shared assertion helper |

## example_hello_test.go, example_provider_test.go, example_annotated_test.go, example_multibinding_test.go, example_interception_test.go, example_child_test.go

One whole-file `Example` each, one per README section. No catalogue IDs: these are documentation,
not regression tests, and duplicate no assertion made elsewhere.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `ExampleInjector_Bind` | bind an interface to an implementation, resolve it | — | README "Binding" section |
| `ExampleInjector_Bind_provider` | bind via `ToProvider`, injected arguments | — | README "Provider" section |
| `ExampleInjector_Bind_annotated` | two annotated bindings of the same interface | — | README "Annotations" section |
| `ExampleInjector_BindMulti` | three `BindMulti` calls, injected as a slice in order | — | README "MultiBindings" section |
| `ExampleInjector_BindInterceptor` | one interceptor wraps a bound implementation | — | README "Interception" section |
| `ExampleInjector_Child` | a child injector resolves a parent-bound type | — | README "Child injectors" section |

## example_migration_test.go

| Test function | Case | Covers | Note |
|---|---|---|---|
| `ExampleInjector_Bind_migration` | v0 form and v2 form side by side, same outcome | — | README "Migrating from v0.x" |
| `ExampleInjector_Bind_config` | the Flamingo config loop: `string`/`bool`/`float64`/`Map`/`Slice` bound under `"config:"`-prefixed annotations | — | README runtime-typed-values section |

## bind_key_test.go

What `T` means, key normalization, and the shapes all four entry points reject.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestBind_KeyIsTWithOnePointerLevelRemoved` | `T` an interface keys on the interface | K-01 | |
| `TestBind_KeyIsTWithOnePointerLevelRemoved` | `T` a struct value keys on the struct | K-02 | |
| `TestBind_KeyIsTWithOnePointerLevelRemoved` | `Bind[*Service]()` and `Bind[Service]()` key to the same slot | K-03 | (decision) same key, two spellings |
| `TestBind_KeyIsTWithOnePointerLevelRemoved` | `T` a basic type keys on the basic type | K-04 | |
| `TestBind_KeyIsTWithOnePointerLevelRemoved` | `T` a slice keys on the slice type, direct `Bind[[]X]()` | K-05 | |
| `TestBind_KeyIsTWithOnePointerLevelRemoved` | `T` a map keys on the map type, direct `Bind[map[string]X]()` | K-06 | |
| `TestBind_KeyIsTWithOnePointerLevelRemoved` | a `Provider`-suffixed func type needs no `Bind` of its own | K-07 | |
| `TestBind_RejectsIllegalTypeParameters` | `Bind`/`BindMulti`/`BindMap`/`Override` each panic on `T` a pointer to an interface | B-01 | (decision) new bind-time check |
| `TestBind_RejectsIllegalTypeParameters` | a `*Iface` key is unreachable from any injection site: the engine rejects a `*Iface` inject field before lookup, so B-01 forbids a binding nothing could ever consume | K-08 | the injection-site twin of B-01; the field-side error itself is R-24 |
| `TestBind_RejectsIllegalTypeParameters` | `Bind`/`BindMulti`/`BindMap`/`Override` each accept a plain interface `T` | B-02 | accepted counterpart to B-01 |
| `TestBind_RejectsIllegalTypeParameters` | `Bind`/`BindMulti`/`BindMap`/`Override` each panic on `T` a pointer to a pointer | B-03 | (decision) v0 silently made an unresolvable binding |
| `TestBind_RejectsIllegalTypeParameters` | a `*T` key is unreachable from any injection site: a `**T` field strips one level and looks up `*T`, whose entries live under `T` after the engine's own stripping, so v0's `Bind(new(*X))` was consumed by nothing; B-03 rejects the shape | K-09 | the key-side reason for B-03; asserted by binding `Bind[*counter]()` and `GetInstance[**counter]()` failing (G-05) |
| `TestBind_RejectsIllegalTypeParameters` | `Bind`/`BindMulti`/`BindMap`/`Override` each accept `T` a pointer to a struct, keyed on the struct | B-04 | accepted counterpart to B-03 |
| `TestBinding_ZeroValuePanics` | any method on a `var b dingo.Binding[T]` (or `new(dingo.Binding[T])`) panics wrapping `ErrInvalidBinding`, naming "not created by an Injector" | B-39 | new rule; `dingo.Binding[T]{}.Method()` does not compile (`cannot call pointer method`), so the test uses an addressable variable |
| `TestInjector_ZeroValuePanics` | every method except `Child` on a `var i dingo.Injector` (or `new(dingo.Injector)`, or a nil `*dingo.Injector`) panics wrapping `ErrInvalidBinding`, naming `NewInjector`, `Child` and `compat.Injector` | B-40 | new rule; the typed injector's engine field is nil |
| `TestInjector_ZeroValueChildReturnsAnError` | `Child()` on a zero or nil `*dingo.Injector` returns an error wrapping `ErrInvalidBinding` and does not panic | B-45 | (decision) mirrors the root's nil-receiver `Child()`, which returns an error and is on the must-not-be-lost list |

## bind_target_test.go

`To`, `ToType`, `ToInstance`, `ToProvider`; exactly one target per binding. B-12 (`ToInstance`
type-checked at compile time) is a compile-time-only rule and lives in `compilefail_test.go`
alone; a runtime test for it cannot exist.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestBindingTo_RequiresAssignability` | a value-receiver implementation is assignable, accepted | B-06 | |
| `TestBindingTo_RequiresAssignability` | a pointer-receiver-only implementation is assignable via `*U`, accepted | B-06 | |
| `TestBindingTo_RequiresAssignability` | an unrelated concrete type is not assignable, rejected | B-05 | the compile-fail corpus's `to_unrelated_type_compiles` case points here; it carries no ID of its own |
| `TestBindingTo_RejectsSelfBinding` | `To[T]()` where `T` equals the key panics at bind time | B-07 | (decision) v0: circular error at resolution time; v2: panic at bind time |
| `TestBindingTo_RejectsPointerToInterfaceTarget` | `To[*Iface2]()` panics, same family as B-01 | B-08 | |
| `TestBindingTo_ChainsThroughAnotherInterfaceBinding` | binding one interface to another bound interface resolves through the chain | B-09 | |
| `TestBindingToType_BindsARuntimeChosenTargetLikeTo` | `ToType(reflect.TypeOf(v))` with `v` a value whose type is chosen at run time resolves by constructing and injecting the target, exactly as `To[U]()` does; the resolved value is not `v` itself and has its `inject` fields set | B-41 | (decision) the v2 form of v0's `.To(value)`; asserts the difference from `ToInstance(v)`, which would return `v` uninjected; the Flamingo `BindRoutes`/`BindTemplateFunc` shape |
| `TestBindingToType_AppliesTheToRules` | `ToType(nil)` panics "nil type"; a non-assignable type, the key itself, and a pointer to an interface each panic with the `To` messages, naming the target from the `reflect.Type` | B-42 | table: nil, not assignable, self, pointer to interface, accepted `*U` |
| `TestBinding_RejectsASecondTarget` | `ToType` counts as a target: `ToType` after `To`/`ToInstance`/`ToProvider` and each of them after `ToType` panic naming the first call | B-43 | folded into the B-21 table as a fourth method, 4x4 orderings |
| `TestBindingToInstance_RejectsNilInterfaceValue` | `ToInstance(nil)` for an interface `T` panics with "nil instance" | B-10 | (decision) v0 crashed with a nil-pointer dereference; only an interface `T` can receive a nil interface value |
| `TestBindingToInstance_AcceptsTypedNilPointer` | a typed nil `*Impl` stays accepted; `GetInstance[Iface]()` returns a non-nil interface value holding a nil `*Impl` (`assert.NotNil` on the interface, `assert.Nil` on the asserted pointer) | B-11 | contrast with B-10; the two assertions are deliberate, an `assert.Nil` on the interface would fail |
| `TestBindingToProvider_ValidatesShape` | a non-function value panics "must be a function" | B-13 | |
| `TestBindingToProvider_ValidatesShape` | a nil function value panics "nil provider" | B-44 | (decision) listed tightening; v0 panicked inside reflect |
| `TestBindingToProvider_ValidatesShape` | 0 results and 3 results both panic on result count | B-14 | (decision) listed tightening; v0 accepted 3 results and read only the first, and panicked inside reflect on 0 |
| `TestBindingToProvider_ValidatesShape` | 1 result assignable to the key, accepted | B-16 | |
| `TestBindingToProvider_ValidatesShape` | 1 result not assignable to the key, rejected | B-15 | |
| `TestBindingToProvider_ValidatesShape` | 1 result assignable only via `*U`, accepted | B-17 | |
| `TestBindingToProvider_ValidatesShape` | a 2nd result not implementing `error`, rejected | B-18 | (decision) listed tightening; v0 accepted and ignored it |
| `TestBindingToProvider_ValidatesShape` | a 2nd result of type `error`, accepted | B-19 | |
| `TestBindingToProvider_ArgumentsAreInjectedFreely` | any argument count/type bind-time-validates; only a call-time failure is possible | B-20 | |
| `TestBinding_RejectsASecondTarget` | all 4x4 orderings of `To`/`ToType`/`ToInstance`/`ToProvider` (incl. same method twice) panic naming the first call | B-21 | (decision) v0: instance > provider > to silently won; v2: panic naming the first call; named top-level test |
| `TestBinding_TargetlessBindingStaysLegal` | `Bind[T]().In(scope)` with no target resolves via direct construction when `T` is concrete | B-22 | |

## bind_attribute_test.go

`AnnotatedWith`, `In`, `AsEagerSingleton`, free method order.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestBindingAnnotatedWith_SameValueTwiceIsANoOp` | repeating the same annotation value does not panic | B-23 | |
| `TestBindingAnnotatedWith_DifferentValuePanics` | a second, different annotation value panics | B-24 | accepted counterpart is B-23 |
| `TestBindingIn_RejectsNilScope` | `In(nil)` panics with "nil scope" | B-25 | |
| `TestBindingIn_SameScopeTwiceIsANoOpDifferentScopePanics` | `In(Singleton)` twice is a no-op | B-26 | |
| `TestBindingIn_SameScopeTwiceIsANoOpDifferentScopePanics` | `In(Singleton)` then `In(ChildSingleton)` panics | B-26 | |
| `TestBindingAsEagerSingleton_PanicsOnConflictingScope` | `AsEagerSingleton()` after a different scope is already set panics | B-27 | (decision) v0 silently overwrote the prior scope; named top-level test |
| `TestBindingAsEagerSingleton_CompatibleWithPrecedingSingletonScope` | `In(Singleton).AsEagerSingleton()` works | B-28 | accepted counterpart to B-27 |
| `TestBinding_MethodOrderIsFree` | `.In(s).To[U]()` and `.To[U]().In(s)` behave identically | B-29 | order permutations, table-driven |
| `TestBinding_MethodOrderIsFree` | `.AnnotatedWith(a).ToInstance(v)` and `.ToInstance(v).AnnotatedWith(a)` behave identically | B-29 | |
| `TestBindingOverride_PresetsAnnotation` | `Override[T](a)` then `.AnnotatedWith(a)` with the same value is a no-op | B-30 | |

## bind_errors_test.go

The panic itself, the sentinel, the message shape.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestBind_PanicsWithAnErrorWrappingErrInvalidBinding` | any bind-time misuse panics with an error satisfying `errors.Is(err, ErrInvalidBinding)`, via `bindPanic` outside `TryModule` | B-37 | raw panic path |
| `TestTryModule_ConvertsBindPanicsToErrors` | `TryModule` converts the same panic into a returned error | B-37 | conversion path |
| `TestBindMessages_HaveTheDocumentedShape` | one representative rejection per entry point (`Bind`, `BindMulti`, `BindMap`, `Override`, `To`, `ToType`, `ToInstance`, `ToProvider`, `AnnotatedWith`, `In`, `AsEagerSingleton`, `BindInterceptor`, `GetInstance`) matches `dingo: <call>: <reason>` with both qualified type names, and `<call>` carries the map key for `BindMap` and the annotation for `Override` and after `AnnotatedWith` | B-38 | one rejection per entry point, not per check; the key/annotation part of the shape was untested before |

## resolution_test.go

`GetInstance[T]`/`GetAnnotatedInstance[T]`, adaptation, failures, the `"map:"` convention.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestGetInstance_AdaptsTheResultToT` | `GetInstance[*Service]()` returns the resolver's pointer or the bound instance | R-01, G-01 | |
| `TestGetInstance_AdaptsTheResultToT` | `GetInstance[Service]()` returns a copy, not an alias of the bound `*Service`; the only place v2 hands out a struct copy, since a struct-kind inject field is an engine error (R-32) | R-02, G-01 | `counter` fixture |
| `TestGetInstance_AdaptsTheResultToT` | `GetInstance[Iface]()` returns the bound implementation, no manual assertion | R-03, G-01 | |
| `TestGetInstance_AdaptsTheResultToT` | `GetInstance[string]()` on a `ToInstance("x")` binding returns `"x"` | R-04, G-01 | |
| `TestGetInstance_AdaptsTheResultToT` | `GetInstance[[]X]()` returns the multibinding slice directly | R-05, G-01 | direct call, not field injection |
| `TestGetInstance_AdaptsTheResultToT` | `GetInstance[map[string]X]()` returns the map binding directly | R-06, G-01 | direct call, not field injection |
| `TestGetInstance_ResolvesInstanceBindingsDirectly` | an instance binding bypasses scope/provider/to logic entirely | R-09 | |
| `TestGetInstance_FollowsToChains` | `To` a concrete struct resolves via construction plus field injection | R-13 | |
| `TestGetInstance_FollowsToChains` | a 3+ hop interface-to-interface chain resolves to the end | R-14 | deeper than the existing 2-hop case |
| `TestGetInstance_TargetlessConcreteTypeConstructsDirectly` | a target-less concrete binding resolves by direct construction | R-15 | |
| `TestGetInstance_UnboundInterfaceErrors` | an interface with no binding at all errors "can not instantiate interface" | R-16 | isolated from the circular-dependency test |
| `TestGetAnnotatedInstance_ResolvesEachBindingShape` | annotated instance, provider, `To`, and target-less bindings all resolve | G-02 | |
| `TestGetAnnotatedInstance_ResolvesEachBindingShape` | an unbound annotated request without `optional` errors | G-02 | |
| `TestGetAnnotatedInstance_ResolvesEachBindingShape` | the Flamingo config loop: `string`/`bool`/`float64`/a map type/a slice type bound under `"config:"`-prefixed annotations, each resolved via `GetAnnotatedInstance[T]` | G-02 | config-loop shape from the spec |
| `TestGetAnnotatedInstance_MapPrefixIsASpecialLookup` | `GetAnnotatedInstance[T]("map:k")` with no singular binding annotated `"map:k"` fetches the map-binding entry `k` | G-03 | (decision) undocumented internal convention (`dingo.go:249-251`) |
| `TestGetAnnotatedInstance_MapPrefixIsASpecialLookup` | a singular binding annotated exactly `"map:k"` wins over the map-binding entry `k`: the exact-annotation match runs first (`dingo.go:240-247`), the `map:` lookup second | G-03 | (decision) the earlier catalogue stated this order backwards; reproduced 2026-09-14 |
| `TestGetAnnotatedInstance_MapPrefixIsASpecialLookup` | the annotation `"map:"` alone (four characters) is not a map lookup and resolves like any other unbound annotation | G-03 | (decision) `len(annotation) > 4` guard; public-API footgun documented in the README |
| `TestGetInstance_PointerToInterfaceReturnsWrappedError` | `GetInstance[*Iface]()` returns an error wrapping the exported `ErrPointerToInterface` | G-04 | (decision) new plumbing, not a port of v0 behavior |
| `TestGetInstance_PointerToPointerReturnsWrappedError` | `GetInstance[**T]()` and `GetAnnotatedInstance[**T](a)` return an error wrapping `ErrInvalidBinding`, no resolution | G-05 | (decision) the engine's `reflect.Type` path would strip one level and resolve `*T`; the request-side twin of B-03 |
| `TestBind_DirectSliceBindingWinsOverMultibinding` | a direct `Bind[[]X]()` wins over `BindMulti[X]()` at the same injection site | K-10 | (decision) precedence between two binding mechanisms |
| `TestBind_DirectMapBindingWinsOverBindMap` | a direct `Bind[map[string]X]()` wins over `BindMap[X](...)` at the same injection site | K-11 | (decision) precedence between two binding mechanisms |
| `TestBind_DirectProviderTypeBindingWinsOverGeneration` | a direct `Bind[XProvider]().ToInstance(fn)` wins over the auto-generated provider for `XProvider` at the same injection site | K-12 | (decision) bindings are consulted (`dingo.go:348`) before the `Provider`-suffix rule (`:361`) |

## provider_test.go

Providers, generated `…Provider` types, the dropped error result.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestGetInstance_ReturnsGeneratedProviderFunction` | `GetInstance[eventRouterProvider]()` on an unbound provider type returns the generated func | R-07 | direct call |
| `TestGetInstance_UnsuffixedFuncTypeErrors` | `GetInstance[func() X]()` without the `Provider` suffix errors | R-08 | |
| `TestProvider_ResolvesWithoutArguments` | a zero-argument provider func resolves and is called | R-10 | |
| `TestProvider_ResolvesArgumentsViaInjection` | each provider argument is resolved via the injector before the call | R-11 | |
| `TestProvider_UnboundConcreteArgumentZeroConstructs` | an unbound, concrete, non-interface argument resolves to its zero value instead of erroring | R-11a | (decision) central to the design, easy to assume is an error |
| `TestProvider_ErrorResultIsIgnoredAtResolution` | a `.ToProvider(fn)` binding whose `fn` returns `(T, error)` with a non-nil error, resolved via plain `GetInstance[T]()`, silently drops the error | R-12 | (decision) most spec-flagged surprising behavior; unchanged from v0.4.1 |
| `TestGeneratedProvider_NeedsOnlyTheUnderlyingTypeBound` | a `Provider`-suffixed field type auto-generates its implementation with no binding of its own | R-28 | |
| `TestGeneratedProvider_ForSliceRoutesToMultibinding` | a `func() []X`-shaped, `Provider`-suffixed field routes to multibinding resolution | R-29 | |
| `TestGeneratedProvider_ForMapRoutesToMapBinding` | a `func() map[string]X`-shaped, `Provider`-suffixed field routes to map-binding resolution | R-30 | |

## injection_test.go

Inject tags, optional, `Inject` methods, `RequestInjection`, `*Injector`.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestInjection_PointerFieldReceivesPointerValueFieldReceivesDereferencedValue` | a `*Concrete` field gets the pointer the engine constructed; a non-struct value field (`label`, a named `string`) bound `ToInstance` gets the value | R-17 | side-by-side pair; the value field is of non-struct kind on purpose, see R-32 |
| `TestInjection_StructKindFieldIsAnError` | an `inject`-tagged field of struct kind (`counter`, not `*counter`) fails injection with the engine's `can not inject into struct` error, binding present or not | R-32 | (decision) `dingo.go:752-754`; the earlier R-17 described a `.Elem()` copy for struct fields that does not exist |
| `TestInjection_EachInjectionIsAnIndependentCopy` | mutating one injected non-struct value-typed field does not affect another injection | R-18 | related to R-02; `label` fixture |
| `TestInjection_AnnotatedFieldMatchesExactAnnotationString` | `inject:"a"` matches `.AnnotatedWith(a)` by exact string equality | R-19 | |
| `TestInjection_OptionalTagLeavesUnresolvableFieldAtZeroValue` | `inject:"tag,optional"` on an unresolvable field leaves it at zero value | R-20 | |
| `TestInjection_OptionalTagLeavesUnresolvableFieldAtZeroValue` | `inject:"tag, optional"` (spaced) behaves the same | R-20 | |
| `TestInjection_InjectMethodRunsBeforeFieldInjection` | a pointer-receiver `Inject(args...)` method runs, args resolved via the injector | R-21 | |
| `TestInjection_AnonymousAnnotatedStructIsResolvedAsAnArgument` | an anonymous `*struct{ V string `inject:"config:k,optional"` }` arrives populated as an `Inject` parameter and as a `ToProvider` argument, with the optional field zero when the annotation is unbound | R-21a | Flamingo's dominant config idiom (`servemodule.Inject` in `app.go`); both entry paths in one test |
| `TestInjection_NonPointerReceiverInjectMethodErrors` | an `Inject` method reached with a non-pointer receiver kind returns `ErrInvalidInjectReceiver` | R-22 | |
| `TestRequestInjection_DelayedDuringConfigureRunsAfterAllModules` | `RequestInjection` called during `Configure` is queued and runs only after every module has configured | R-23 | delayed-queue timing |
| `TestRequestInjection_AfterInitModulesInjectsImmediately` | `RequestInjection` called after `InitModules` returned injects the object before it returns | R-33 | the immediate path had no ID |
| `TestRequestInjection_QueuedFailureReturnsFromInitModulesUnwrapped` | a queued `RequestInjection` whose object has an unresolvable field makes `InitModules` return the bare injection error; `errors.Is(err, ErrInitModules)` is false for it | R-34 | (decision) `dingo.go:169-173`; kept so that a caller matching `ErrInitModules` is not misled by the spec |
| `TestInjection_PointerToInterfaceFieldErrors` | a `*Iface`-typed inject field errors, binding present or not | R-24 | unchanged from v0 |
| `TestInjection_InjectorSelfBindingReturnsCurrentInjector` | a struct resolved in a child gets the child `*Injector`, not the root | R-31 | |

## multibinding_test.go

`BindMulti`, `BindMap`.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestBindMulti_RegistersInOrder` | N `BindMulti[Handler]()` calls inject as `[]Handler` in registration order | MB-01 | |
| `TestBindMulti_DuplicateImplementationsAreAllKept` | two `BindMulti` calls for the same implementation type are both kept, no error | MB-02 | (decision) contrasts with singular-binding duplicate detection |
| `TestBindMulti_AnnotationsProduceDisjointSlices` | different annotations on `BindMulti` produce disjoint slices | MB-03 | |
| `TestBindMulti_EmptyResolvesToNonNilEmptySlice` | no `BindMulti` calls for a type resolves to a non-nil, empty slice | MB-04 | |
| `TestBindMulti_ChildMergesAfterParent` | a child's multibinding slice has parent entries first, then the child's own | MB-05 | |
| `TestBindMap_LastWriteWinsPerKey` | `BindMap[Iface]("k")` injects as `map[string]Iface` | MB-06 | |
| `TestBindMap_DuplicateKeySilentlyOverwrites` | a second `BindMap` call for the same key silently wins, no error | MB-07 | (decision) contrasts with singular-binding duplicate detection |
| `TestBindMap_AnnotationsProduceDisjointMaps` | different annotations on `BindMap` produce disjoint maps | MB-08 | |
| `TestBindMap_EmptyResolvesToNonNilEmptyMap` | no `BindMap` calls for a type resolves to a non-nil, empty map | MB-09 | |
| `TestBindMap_ChildOverridesParentKey` | a child's `BindMap` entry for the same key overrides the parent's, other keys still inherited | MB-10 | |
| `TestBindMulti_ProviderSuffixedElementsResolveIndependently` | `[]XProvider` gives each element its own generated provider bound to a specific implementation | MB-11 | |
| `TestBindMap_ProviderSuffixedValuesResolveIndependently` | `map[string]XProvider` gives each key its own generated provider | MB-12 | |
| `TestBindMulti_PerElementScopeIsIndependent` | a scoped `BindMulti`/`BindMap` element resolves through the same scope machinery, per element | MB-13 | covers both slice and map bindings; uses `ChildSingleton`, isolated by `newInjector`'s second swap |
| `TestBindMulti_OverrideOfAMultibindingLeavesTheSliceAlone` | `Override[T](a)` on a `BindMulti`-only `T` adds a singular `(T, a)` binding and does not touch the multibinding slice, which still resolves with all its entries | MB-14 | (decision) the earlier catalogue expected an "unknown binding" error the engine never raises; cross-ref B-32 |
| `TestBindMulti_RejectsProviderSuffixedElementType` | `BindMulti[greeterProvider]()` panics at bind time naming `BindMulti[greeter]` as the form to use | MB-15 | (decision) listed tightening; a `[]greeterProvider` site looks up `greeter`'s multibinding (`dingo.go:528-534`), so v0 resolved it to an empty slice with no error |
| `TestBindMap_RejectsProviderSuffixedValueType` | `BindMap[greeterProvider]("k")` panics at bind time naming `BindMap[greeter]` as the form to use | MB-16 | (decision) listed tightening; same mechanism for maps (`dingo.go:580-586`) |
| `TestBindMap_FieldWithMapPrefixReceivesOneEntry` | an `inject:"map:k"` field of type `greeter` receives the map-binding entry `k`; with no such entry and no `optional` it errors | MB-17 | the field form of G-03; `inject:"map:key"` from the root's `multi_dingo_test.go` |
| `TestBindMulti_TargetlessDirectSliceBindingScopesTheMultibinding` | `Bind[[]greeter]().In(dingo.Singleton)` with no target falls through to the `BindMulti` entries and caches the assembled slice in the scope: two resolutions return the same slice | MB-18 | (decision) how a multibinding slice gets a scope; verified real 2026-09-14; contrast with K-10, where a target makes the direct binding win |

## scope_test.go

Scope identity matrix, custom scopes, eager singletons, concurrency.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestScope_IdentityMatrix` | `Singleton` caches across repeated resolution in one injector | S-01 | one cell of the {scope}x{axis} matrix |
| `TestScope_IdentityMatrix` | `ChildSingleton` caches independently per child | S-02 | root-level `In(ChildSingleton)` cells rely on `newInjector`'s `ChildSingleton` swap |
| `TestScope_IdentityMatrix` | a registered custom `Scope` resolves; an unregistered one errors "unknown scope" | S-03 | |
| `TestScope_IdentityMatrix` | the same type with different annotations does not collide in the singleton cache | S-08 | axis of the matrix, not a separate scope |
| `TestBindScope_FreshSingletonScopeReplacesTheCache` | `BindScope(dingo.NewSingletonScope())` on an injector replaces the `*SingletonScope` registration: a `Singleton`-scoped binding resolved before and after the swap yields two instances, and the same for `NewChildSingletonScope()` | S-09 | (decision) scopes are looked up by Go type; the helper contract every other test relies on |
| `TestEagerSingleton_BuildsWithoutTouchingParent` | `BuildEagerSingletons(false)` builds only the current injector's eager bindings | S-04 | side effect observed before any `GetInstance` call |
| `TestEagerSingleton_WithParentBuildsChildBeforeParent` | `BuildEagerSingletons(true)` on a child builds the child's, then the parent's | S-05 | the parent is created with `SetBuildEagerSingletons(false)`; otherwise `NewInjector` builds the parent's eager singletons before the child exists and the order is unobservable |
| `TestEagerSingleton_ConstructionFailureReturnsFromNewInjector` | an `AsEagerSingleton()` binding whose construction fails (an unresolvable `inject` field) makes `NewInjector` return the error, while `TryModule` on the same module returns nil | S-10 | (decision) `TryModule` proves bind-time acceptance only; both halves in one test |
| `TestSetBuildEagerSingletons_DisablesAutomaticBuild` | `SetBuildEagerSingletons(false)` skips the automatic build; a manual call builds them later | S-06 | exported-API path |
| `TestSingleton_ConcurrentResolutionConstructsExactlyOnce` | 100 goroutines resolving the same singleton-scoped binding construct it exactly once | S-07 | `atomic.Int64` counter, `sync.WaitGroup`, run under `-race`; the type has no interceptor, because the interceptor map is unguarded (I-07) and would add an unrelated race |

## child_test.go

Parent/child/sibling matrix, inherited interceptors, the global `Singleton`.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestChild_FallsBackToParentBinding` | a child with no binding for `T` resolves it from the parent | C-01 | |
| `TestChild_LocalBindingInvisibleToParentAndSiblings` | a child-only binding is invisible to the parent and to sibling children | C-02 | re-derived, not a blind port |
| `TestChild_LocalBindingShadowsParentBinding` | a child binding for a key the parent also binds wins for resolutions started at the child, for singular bindings, map-binding keys (MB-10) and the `*Injector` self-binding; the parent and a sibling still see the parent's | C-05 | the `boundAt = both` column of the matrix; what every Flamingo per-area override (`framework/config/area.go:305-325`) relies on |
| `TestChild_InterceptorsInheritedFromParentUnconditionally` | a parent-registered interceptor wraps a child-resolved instance outside the child's own interceptors; a resolution started at the parent never sees child interceptors | C-03 | (decision) Guice rule, kept 2026-09-09; order asserted as a recorded `[]string` |
| `TestChild_SingletonScopeIsSharedGloballyNotPerInjector` | two independently-bound `Singleton`-scoped bindings for the same `(type, annotation)` on two root injectors collide in one shared cache | C-04 | (decision) sharp edge; uses `dingo.NewInjector` twice with no helper, not parallel, on a fixture type unique to this test so no parallel test reads the cached instance |

## override_test.go

`Override[T]`, unknown and multibinding overrides, late overrides, duplicate detection. B-31 and
B-32 keep their numbers from the earlier catalogue but pin the opposite of what it said: the
engine never raises "unknown binding" for a key v2 lets through, because `Override` calls `Bind`
first (`dingo.go:681`), so the key always has at least the override's own binding when the
override loop runs.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestOverride_ReplacesEveryMatchingAnnotationEntry` | `Override[T](a)` replaces every prior binding matching `(T, a)`, not just the first | OV-01 | |
| `TestOverride_UnknownBindingSilentlyBecomesAPlainBinding` | `Override[T](a)` for a `(T, a)` nothing bound returns no error from `InitModules` and resolves as a plain `(T, a)` binding | B-31, OV-02 | (decision) v0 quirk kept; the `cannot override unknown binding` branch is unreachable through v2 (it needs a pointer-typed key, which B-03 rejects); reproduced 2026-09-14; v0's `TestOverrides` never covered it |
| `TestOverride_OfAMultibindingAddsASingularBindingAndLeavesTheSliceAlone` | `Override[T](a)` on a `BindMulti`-only `T` returns no error, adds a singular `(T, a)` binding, and `GetInstance[[]T]()` still returns every multibinding entry | B-32, OV-03 | (decision) overriding multibindings is not supported and not detected; cross-ref MB-14 |
| `TestOverride_PresetsAnnotationWithoutDoubleCounting` | the override-created binding does not itself get overridden again or trip duplicate detection | OV-04 | override evaluation runs before duplicate-check |
| `TestOverride_AfterInitModulesIsNeverEvaluated` | `Override[T](a)` called after `InitModules` returned does not replace the existing `(T, a)` binding; the earlier binding still resolves, and no error is reported | OV-05 | (decision) the override loop runs only inside `InitModules` (`dingo.go:132`); a late override is a dangling plain binding appended after the duplicate check |
| `TestDuplicateBinding_EqualBindingsAreTolerated` | two structurally-`equal()` bindings for the same `(type, annotation)` do not error | DUP-01 | |
| `TestDuplicateBinding_UnequalBindingsErrorAtInitModules` | two `To`-based bindings with different targets for the same `(type, annotation)` error at `InitModules`, and the message names both target types | DUP-02 | zero existing v0.4.1 coverage despite being unchanged runtime behavior; `To`-based on purpose: the message interpolates only `binding.to` (`dingo.go:154-160`), so for `ToInstance`/`ToProvider` bindings it names empty strings and "naming both" is unsatisfiable |
| `TestDuplicateBinding_UnequalBindingsErrorAtInitModules` | two `ToInstance` bindings with different instances for the same `(type, annotation)` error at `InitModules`; only the key and annotation are asserted in the message | DUP-02 | the same rule for a binding kind the message cannot name |

## interceptor_test.go

Chain order, field-0 write, injection into the interceptor, the wrapper per resolution,
validation.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestBindInterceptor_ValidatesShape` | `T` a concrete type panics "is not an interface" | B-33 | |
| `TestBindInterceptor_ValidatesShape` | `T` a pointer to an interface panics, same family | B-33 | |
| `TestBindInterceptor_ValidatesShape` | `I` not a struct, or a struct whose first field is not `T`, panics naming both types | B-34 | (decision) v0: raw reflect panic on first use; v2: bind-time panic |
| `TestBindInterceptor_ValidatesShape` | `I` a struct whose field 0 has type `T` but is unexported panics at bind time naming both types | B-34a | (decision) the engine's `Field(0).Set` panics with `reflect.Value.Set using value obtained using unexported field` at first resolution (`dingo.go:310`); the suite's lowercase-fixture rule would produce exactly this shape, so interceptor fixtures export field 0 |
| `TestBindInterceptor_ValidatesShape` | `*I` does not implement `T` panics | B-35 | |
| `TestBindInterceptor_ValidatesShape` | `T` an interface, `I` a struct with an exported field 0 of type `T`, `*I` implementing `T` (incl. a value-receiver own method) is accepted | B-36 | accepted counterpart |
| `TestBindInterceptor_RejectedInterceptorRegistersNothing` | after a rejected `BindInterceptor[T, I]()` (panic recovered with `bindPanic`), resolving `T` on the same injector returns the bare bound implementation, unwrapped | I-04 | v2's checks run before the engine is touched; previously a cross-reference with no case of its own |
| `TestInterceptor_ChainAppliesInRegistrationOrder` | two interceptors registered for `T` apply in registration order, each wrapping the previous | I-01 | |
| `TestInterceptor_Field0ReceivesTheBaseValuePositionally` | a named, non-embedded field 0 still receives the base value | I-02 | embedding might obscure a positional bug |
| `TestInterceptor_InjectTaggedFieldsAlsoResolve` | an interceptor with both a field-0 target and a separate `inject`-tagged field gets both resolved | I-03 | |
| `TestInterceptor_ReWrapsOnEveryResolutionEvenWithASingletonBase` | a `Singleton`-scoped base is constructed once; the interceptor wrapper is rebuilt on every resolution, so two resolutions give two wrappers (`NotSame`) over one inner instance (`Same` on field 0) | I-05 | (decision) interception runs after the scope cache, so wrapper state does not survive a resolution |
| `TestInterceptor_AppliesAcrossChildBoundary` | a parent interceptor fires on a child-resolved instance, child inner, parent outer | I-06 | cross-ref C-03, no separate mechanism |
| — | `BindInterceptor` on an injector that is already serving concurrent resolutions is a data race: the engine's interceptor map is written by `BindInterceptor` and read by resolution without synchronization | I-07 | excused, see "Excused IDs"; stated under "Kept v0 behaviors worth knowing" so S-07 knows to avoid it |

## module_test.go

`Module`, `ModuleFunc`, `Depender`, dedup, identity, `InitModules`, `TryModule`, through the
public API, including the moduleidentity fixture packages.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestModule_ConfigureCalledOncePerModule` | `Configure` runs once per registered module | M-01 | |
| `TestModuleFunc_DistinctClosuresAreDistinctModules` | distinct closures from the same func literal are distinct modules | M-02 | |
| `TestDepender_DependenciesConfiguredBeforeDependents` | a `Depends()` module's dependencies configure first, registration order preserved among independents | M-03 | public-API view; the white-box table stays in the root module's `module_test.go` |
| `TestModule_DuplicateModulesConfiguredOnce` | the same module type or `ModuleFunc` value, direct or transitive, configures once | M-04 | |
| `TestModule_DuplicateModulesConfiguredOnce` | same-named module types from different packages are treated as distinct | M-04 | moduleidentity fixture packages |
| `TestModule_CycleErrorNamesThePath` | a dependency cycle returns `ErrModuleCycle` with an `"A → B → C"` name chain | M-05 | public-API view; the white-box table stays in the root module's `module_test.go` |
| `TestTryModule_PassesThroughErrorPanicsUnchanged` | `panic(someErr)` inside `Configure` returns that same error from `TryModule` | M-06 | |
| `TestTryModule_WrapsNonErrorPanicsWithQFormatting` | `panic(42)` returns `dingo.TryModule panic: '*'`, pinned exactly (`%q` renders an int as a character literal) | M-07 | |
| `TestInitModules_WrapsSortFailureInErrInitModules` | a cycle-caused sort failure is wrapped in `ErrInitModules` | M-08 | the add-failure branch is excused, see below |
| `TestTryModule_DoesNotBuildEagerSingletons` | v2 `TryModule` on a native v2 module with an `AsEagerSingleton()` binding whose construction would fail returns nil: eager singletons are not built | M-09 | the v2-native twin of X-15, which covers adapted modules only; pairs with S-10 |

## tracing_test.go

The one v2 test that touches a package-level tracing switch. Not parallel; the switch has no off
switch, so the root suite stays the primary pin (R-27 under "Pinned in the root module").

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestEnableInjectionTracing_ReachesTheEngine` | after v2 `EnableInjectionTracing()`, one field injection emits a `slog` record containing `SETTING FIELD` through a handler swapped in with `slog.SetDefault` and restored in `t.Cleanup` | R-27 | v2 view of R-27: proves the wrapper calls the root function; second `paralleltest`/`tparallel` `nolint` island after C-04 |

## inspect_test.go

The four `Inspect*` callbacks.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestInspect_BindingCallbackFiresPerSingularBinding` | `InspectBinding` fires once per bound type with its target, provider/instance and scope | INS-01 | zero existing coverage before this file |
| `TestInspect_MultiBindingCallbackFiresPerEntryWithSharedIndexSpace` | `InspectMultiBinding` fires per entry with its slice index, shared across annotations | INS-02 | (decision) index space is not filtered by annotation |
| `TestInspect_MapBindingCallbackFiresPerKey` | `InspectMapBinding` fires once per map-binding key | INS-03 | |
| `TestInspect_ParentCallbackFiresOnlyWhenAParentExists` | `InspectParent` fires once with the parent for an injector created by `Child()` and never for one created by `NewInjector`; it does not recurse into the grandparent on its own | INS-04 | stated through the public API: `Child()` is the only way an injector gets a parent |
| `TestInspect_NilCallbacksAreSkipped` | an `Inspector` with only one callback set does not panic and does not invoke the others | INS-05 | requires v2's `Inspect` to keep each callback's nil-ness when it builds the root `Inspector`; a typed injector that installed no-op callbacks would pass INS-01..INS-04 and fail here |
| `TestInspect_ProviderAndInstanceArgsAreNilWhenAbsent` | `provider`/`instance` arguments are nil for a target-less or `To`-only binding | INS-06 | tests presence and absence for both |

## compilefail_test.go

Drives `v2/testdata/compilefail/cases` with `GOWORK=off`, so the corpus module's two `replace`
directives and its committed `go.sum` apply. Expected substrings are `cannot use` for type
mismatches and `cannot infer` for missing type arguments (Go 1.27 reports `in call to i.Bind,
cannot infer T`; the substring `not enough type arguments` never appears).

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestCompileFail_MatchesExpectedErrors` | `to_instance_wrong_type`: `ToInstance` takes typed `T`, so a value of the wrong type fails to build with `cannot use` | B-12 | (decision) v0's runtime panic is now a compile error; the only pin for B-12 |
| `TestCompileFail_MatchesExpectedErrors` | `get_instance_wrong_assignment`: `GetInstance[greeter]()` assigned to `string` fails to build | — | |
| `TestCompileFail_MatchesExpectedErrors` | `bind_missing_type_argument`: `Bind()` without `[T]` fails to build with `cannot infer` | — | |
| `TestCompileFail_MatchesExpectedErrors` | `to_missing_type_argument`: `.To()` without `[U]` fails to build with `cannot infer` | — | |
| `TestCompileFail_MatchesExpectedErrors` | `interceptor_one_type_argument`: `BindInterceptor[greeter]()` fails to build | — | |
| `TestCompileFail_MatchesExpectedErrors` | `to_unrelated_type_compiles`: `Bind[A]().To[B]()` with unrelated `B` builds | — | (decision) documents the compile-time/bind-time boundary, must build; the rejection itself is B-05 in `bind_target_test.go`, and this row carries no ID so that one ID never stands for two opposite assertions |

## readme_test.go

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestReadme_UsesV2CallShapes` | no fenced Go block in `v2/README.md` contains a v0 call shape (`Bind(`, `BindMulti(`, `BindMap(`, `Override(`, `BindInterceptor(`, `GetInstance(`, `GetAnnotatedInstance(`, `.To(` without a type argument) | — | drift guard; the migration table uses inline code, not fences |

## catalogue_test.go

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestCatalogue_EveryIDHasAStatement` | every line of `testdata/catalogue.txt` is `ID<TAB>statement` with a non-empty statement; an excused line adds `<TAB>excused: <reason>` | — | an ID without a statement is undefined and nobody can write its test |
| `TestCatalogue_EveryIDIsReferencedOrExcused` | every ID in `testdata/catalogue.txt` is referenced by a `*_test.go` file under the v2 module, `compat/` included, or has a recorded reason | — | the gate this appendix exists to satisfy |

## compat/compat_test.go (package compat_test)

Two APIs on one engine. Every rule that has two directions runs both as subtests: a v0 module on
a v2 injector and a v2 module on a v0 injector. Module identity is asserted by counting
`Configure` calls per module value; injection timing is asserted from inside `Configure`.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestInjector_IsIdempotentPerRoot` | `compat.Injector(e)` twice returns the same typed injector; `compat.Root(compat.Injector(e))` is `e`; the typed injector already exists before the first call, so `Inspect` on `e` lists the `v2.Injector` binding whether or not `compat.Injector` was ever called | X-01 | `assert.Same`; the last clause pins eager attachment |
| `TestInjector_IsDistinctPerChildRoot` | child created on the v0 side: `compat.Injector(e.Child())` differs from `compat.Injector(e)` and wraps the child engine | X-02 | subtest `child_created_through_the_engine`, Flamingo's `area.Injector = parent.Child()` path |
| `TestInjector_IsDistinctPerChildRoot` | child created on the v2 side: `compat.Injector(e).Child()` differs from the parent typed injector, and `compat.Root` of it is a child of `e` (its `Inspect` reports `e`'s typed injector as parent) | X-02 | subtest `child_created_through_the_typed_api` |
| `TestInjector_SharesBindingsWithTheV0API` | a v0 `Bind(new(I)).To(Impl{})` on the engine resolves through `GetInstance[I]` on the typed injector, and a v2 `Bind[I]().To[Impl]()` resolves through the engine's `GetInstance(new(I))` | X-03 | both directions |
| `TestMultibindingsAndMapBindings_ShareAcrossKinds` | a v0 `BindMulti(new(I)).To(A{})` entry and a v2 `BindMulti[I]().To[B]()` entry arrive together in a `[]I` field of a module of either kind and in `GetInstance[[]I]()`; the same for `BindMap` entries in `map[string]I` | X-17 | both directions |
| `TestToRoot_RunsAV2ModuleOnAV0Injector` | `dingo.NewInjector(compat.ToRoot(m))` (root) configures `m` once and its bindings resolve through the root API | X-04 | |
| `TestFromRoot_RunsAV0ModuleOnAV2Injector` | `dingo.NewInjector(compat.FromRoot(m))` (v2) configures `m` once and its bindings resolve through the v2 API | X-05 | |
| `TestAdaptedModules_KeepTheirIdentity` | the same module value passed direct and adapted, and `ToRoot(FromRoot(m))`, configure once; two distinct module types both configure; `ModuleFunc` closures of either package are distinct by value | X-06 | (decision) identity is the unwrapped module; named top-level test |
| `TestAdaptedModules_DependenciesResolveAcrossKinds` | a v2 `Depender` under a v0 injector and a v0 `Depender` under a v2 injector: every dependency configures once, before the dependent | X-07 | |
| `TestAdaptedModules_FieldsAreInjectedBeforeConfigure` | an adapted module of either kind reads its `inject`-tagged fields inside `Configure`, on either injector | X-08 | (decision) same timing as native modules; named top-level test; the prototype's known gap, closed by `InitModules` injecting the unwrapped module |
| `TestAdaptedModules_FieldsAreInjectedBeforeConfigure` | an adapted module of either kind with an `Inject(cfg *struct{ Level string `inject:"config:level,optional"` })` method has that method called, with the config visible, before `Configure` runs | X-08 | the `core/zap` shape (`core/zap/module.go:60-73`); the `Inject` method is a different engine path (`dingo.go:735-743`) from tagged fields (`:748-791`), so both shapes run |
| `TestAdaptedModules_InjectionFailureReturnsFromInitModules` | an adapted module of either kind with an `inject` field that cannot be resolved: `InitModules` on either injector returns the engine's `initmodules: injection into %q failed` error naming the unwrapped module's type, `TryModule` of either package returns it, and `Configure` never runs | X-19 | (decision) same failure path as native modules; runs once with a pointer-typed inner module and once with a value-typed one (`MyModule{}`), the latter depending on the root PR's pointer guard on `dingo.go:125` |
| `TestSelfBinding_EachKindReceivesItsOwnInjector` | a `*dingo.Injector` (root) field receives the engine, a `*dingo.Injector` (v2) field receives its typed injector; inside a child, the child's | X-09 | |
| `TestScopes_AreSharedAcrossKinds` | a `Singleton`-scoped binding made through one API and resolved through the other is one instance; a scope registered with `BindScope` on the engine serves `In` on the typed injector | X-10 | `dingo.Singleton` is one object in both packages |
| `TestInterceptors_ApplyAcrossKinds` | a v2 `BindInterceptor[I, W]` wraps a v0-bound value; a v0 `BindInterceptor(new(I), W{})` wraps a v2-bound value | X-11 | |
| `TestBindTimeChecks_SurfaceThroughRootEntryPoints` | a v2 misuse inside a `ToRoot` module: the root's `TryModule` returns an error wrapping `v2.ErrInvalidBinding`; the root's `NewInjector` panics with it | X-12 | |
| `TestEngineErrors_KeepRootWordingThroughTheTypedAPI` | an unbound-interface resolution error (`can not instantiate interface`, returned, not panicked, `dingo.go:383-385`) and a `DUP-02` duplicate-binding error reached through the typed API carry the engine's v0 wording, not the `dingo: <call>: <reason>` shape | X-18 | (decision) engine messages stay the engine's |
| `TestConfigLoop_V0RuntimeTypedBindingsReachV2Fields` | v0 `Bind(v).AnnotatedWith("config:k").ToInstance(v)` for `string`, `bool`, `float64` is read by a v2 module's `inject:"config:k"` fields and by `GetAnnotatedInstance[T]` | X-13 | the Flamingo `area.go` shape during steps 0 to 2 |
| `TestInspect_TranslatesTheParentToItsAttached` | v2 `Inspect` lists bindings of both origins and hands `InspectParent` the parent's typed injector, the same value `compat.Injector(parentEngine)` returns | X-14 | |
| `TestTryModule_AcceptsAdaptedModules` | v2 `TryModule(compat.FromRoot(m))` and root `TryModule(compat.ToRoot(m))` both skip eager singletons and return bind errors | X-15 | |
| `TestSentinels_AreSharedValues` | `errors.Is` holds across packages for `ErrPointerToInterface`, `ErrInitModules`, `ErrInvalidInjectReceiver`, `ErrModuleCycle`, `ErrModuleSort` | X-16 | |
| `TestFlamingoShape_MixedModuleTreeOnOneRoot` | mirroring `framework/config/area.go:305-340`: a v0 engine with `SetBuildEagerSingletons(false)`, `Bind(area{}).ToInstance(&area{})` and config values bound under `config:` annotations; one v0 module using `compat.Injector(...)` inside `Configure` and declaring `Depends()`; one `compat.ToRoot(v2Module)` with an `Inject(cfg *struct{...})` method reading config inside `Configure`; both contributing `BindMulti`/`BindMap` entries, one via a `ToProvider` taking an annotated anonymous struct. Asserts: each `Configure` once, in dependency order; `Inject` before `Configure` with config visible; `GetInstance[[]command]()` merges both origins in parent order; `compat.Root(compat.Injector(engine))` is `engine`. Then `engine.Child()` on the v0 side: `compat.Injector(child)` differs from the parent typed injector; a v2 module in the child receives both the typed API and the v0 child; a child-bound binding shadows the parent's; `Inspect` lists both origins and `InspectParent` receives exactly the parent typed injector | X-20 | integration test, not a rule: the one place the Flamingo bootstrap shape is exercised end to end before Flamingo adopts v2 |

## compat/example_test.go (package compat_test)

| Test function | Case | Covers | Note |
|---|---|---|---|
| `Example_rootModule` | a module with the v0 `Configure` signature binds through `compat.Injector(injector)` | — | README "Using v2 next to v0", step 0 of the Flamingo path |
| `ExampleToRoot` | a v2 module listed among v0 modules with `compat.ToRoot` | — | the application-list shape |

## Pinned in the root module

Engine-internal behavior with no observable effect through the v2 API, or needing package-level
state, stays in the root module's suite. These IDs are excused in `testdata/catalogue.txt` with
the root test as the reason. Two root tests added by the root PR carry no catalogue ID because
their subject does not exist in the v2 API: `hooks_test.go` `TestUnwrap_StopsOnNilFixedPointAndDepth`
(the unwrap contract, review decision 15) and `dingo_test.go`
`TestInitModules_NamesAValueTypedInnerModuleWithoutPanicking` (the pointer guard, review
decision 12). The standalone step brings both into v2.

| ID | Root test | Note |
|---|---|---|
| DUP-03 | `binding_test.go` `TestBinding_equal` | existing |
| R-25 | `circular_test.go` `TestDingoCircular` | existing, renamed by the root PR |
| R-27 | `tracing_test.go` `TestInjectionTracing_LogsFieldSetsAndResolutionsWhenEnabled` (package `dingo`) | added by the root PR; restores the switch in `t.Cleanup`; also covered from the v2 side by `tracing_test.go`, so it is pinned, not excused |
| M-03, M-05 white-box halves | `module_test.go` `TestModGraph_Sorted`, `Test_resolveDependencies` | existing; the public views are covered in v2's `module_test.go` |

## IDs without a case of their own

Statements for the IDs the root suite pins or that are excused, so that every ID has one.

| ID | Statement |
|---|---|
| R-25 | A direct self-cycle (`A` injects `*A`) with `EnableCircularTracing` on panics with the cycle path; a `…Provider`-typed self-reference injects fine and panics only when the provider is called. |
| R-26 | A cycle with circular tracing disabled overflows the stack instead of returning a catchable panic. |
| R-27 | `EnableInjectionTracing` makes the engine emit one `slog` line per field set (`SETTING FIELD`) and per resolution. |
| DUP-03 | `binding.equal` compares two bindings field by field: `to`, instance type and value, provider function, scope, eagerness and annotation; a difference in any one makes them unequal. |
| I-07 | `BindInterceptor` on an injector that is already serving concurrent resolutions is a data race on the engine's interceptor map. |
| M-03 (white-box half) | `modGraph.Sort` orders modules topologically with ties broken by insertion order. |
| M-05 (white-box half) | `resolveDependencies` reports a cycle as `ErrModuleCycle` with the module names joined by ` → `. |

## Excused IDs

| ID | Reason | Comment location |
|---|---|---|
| R-26 | A tracing-disabled cycle through a `…Provider`-typed self-reference stack-overflows the process instead of returning a catchable panic; a test would crash the test binary, not fail it. | `testdata/catalogue.txt`, the reason on the ID's line |
| M-08 (add-failure branch) | `ErrInitModules`'s "failed adding modules" branch wraps an error from `mg.Add`, whose only error source is `Depends() []Module`, a signature that cannot itself return an error; the branch is unreachable through the public API. | root `module.go`, `// coverage: unreachable through the public API` above the add-failure branch in `InitModules`, added by the root PR |
| DUP-03, R-25 | Engine-internal; pinned by the root tests listed under "Pinned in the root module". | `testdata/catalogue.txt`, the reason on each ID's line |
| I-07 | A data race cannot be asserted: under `-race` it fails the run, without `-race` it is invisible. Stated in the spec and the README; synchronizing the map is a listed follow-up. | `testdata/catalogue.txt`, the reason on the ID's line |

## ID count

182 IDs: B-01..B-45 with B-34a, K-01..K-12, R-01..R-34 with R-11a and R-21a, MB-01..MB-18,
S-01..S-10, C-01..C-05, OV-01..OV-05, DUP-01..DUP-03, I-01..I-07, M-01..M-09, INS-01..INS-06,
G-01..G-05, X-01..X-20. B-39 (the zero `Binding[T]`), B-40 (the zero `Injector`), K-10 and K-11
(precedence of direct slice and map bindings) and X-01..X-19 (interop) were added by the spec
after the behavior list was derived. The 2026-09-14 review added 23: B-34a, B-41..B-45, K-12,
R-21a, R-32..R-34, MB-15..MB-18, S-09, S-10, C-05, OV-05, I-07, M-09, G-05, X-20; it kept the
numbers of B-31, B-32, OV-02, OV-03 and MB-14 with corrected statements, turned I-04 into a case
and R-27 into a doubly pinned ID, and moved B-12 to the compile-fail corpus alone. Every ID
appears in a Covers column above, under "Pinned in the root module" or under "IDs without a case
of their own"; R-26, the M-08 add-failure branch, DUP-03, R-25 and I-07 appear again under Excused
IDs. `testdata/catalogue.txt` is generated from this file: the ID, a tab, the statement of the
first row that lists it, and for excused IDs a tab and `excused: ` followed by the reason.
