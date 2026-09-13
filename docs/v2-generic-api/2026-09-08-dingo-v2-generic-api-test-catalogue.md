# Appendix: dingo v2 test-case catalogue

Companion to `2026-09-08-dingo-v2-generic-api-design.md`. Lists every planned test case of the
dingo v2 suite with the behavior IDs it pins, one section per test file in the order of the spec's
file layout. The Case column is the behavior statement; the IDs exist so the catalogue gate in
`catalogue_test.go` can prove that nothing on this list was dropped. `(decision)` marks a case that
pins surprising v0 behavior kept on purpose, or a deliberate v2 change. The suite lives in the `v2/`
module of the dingo repository; the `compat/` files are part of it. Engine-internal behavior stays
pinned by the root module's own suite, see "Pinned in the root module".

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
| `counter` | a struct with a mutable field, for copy-semantics assertions | — | supports R-02, R-18 |
| `greeterProvider`/`greetersProvider`/`greeterMapProvider` | `Provider`-suffixed func types, single/slice/map | — | supports R-28..R-30 |
| `plainGreeterFunc` | a `func() greeter` with no `Provider` suffix | — | supports R-08 |
| `countingScope` | a user `Scope` with a construction counter | — | supports S-03 |
| `recordingInterceptor1`/`recordingInterceptor2` | interceptors that append to a shared `[]string` | — | supports I-01 |
| `spyInspector` | an `Inspector` recording into typed slices | — | supports INS-01..INS-06; ranges over maps, sort before comparing |

## helpers_test.go

Six helpers (`newInjector`, `childOf`, `bindErr`, `bindPanic`, `requireInvalidBinding`,
`get[T]`), plus three tests pinning their own contract before anything depends on them.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestHelper_PrivateSingletonScopeIsolatesTwoRoots` | two `newInjector` roots do not share a singleton cache | — | pins `newInjector`'s scope swap |
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
| `TestBind_RejectsIllegalTypeParameters` | `Bind`/`BindMulti`/`BindMap`/`Override` each panic on `T` a pointer to an interface | B-01, K-08 | (decision) new bind-time check; injection-site rule K-08 is unchanged and separate |
| `TestBind_RejectsIllegalTypeParameters` | `Bind`/`BindMulti`/`BindMap`/`Override` each accept a plain interface `T` | B-02 | accepted counterpart to B-01 |
| `TestBind_RejectsIllegalTypeParameters` | `Bind`/`BindMulti`/`BindMap`/`Override` each panic on `T` a pointer to a pointer | B-03, K-09 | (decision) v0 silently made an unresolvable binding |
| `TestBind_RejectsIllegalTypeParameters` | `Bind`/`BindMulti`/`BindMap`/`Override` each accept `T` a pointer to a struct, keyed on the struct | B-04 | accepted counterpart to B-03 |
| `TestBinding_ZeroValuePanics` | any method on `dingo.Binding[T]{}` panics wrapping `ErrInvalidBinding`, naming "not created by an Injector" | B-39 | new rule, not in the behavior catalogue |
| `TestInjector_ZeroValuePanics` | any method on `dingo.Injector{}` panics wrapping `ErrInvalidBinding`, naming `NewInjector`, `Child` and `compat.Injector` | B-40 | new rule; the facade's engine field is nil |

## bind_target_test.go

`To`, `ToInstance`, `ToProvider`; exactly one target per binding.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestBindingTo_RequiresAssignability` | a value-receiver implementation is assignable, accepted | B-06 | |
| `TestBindingTo_RequiresAssignability` | a pointer-receiver-only implementation is assignable via `*U`, accepted | B-06 | |
| `TestBindingTo_RequiresAssignability` | an unrelated concrete type is not assignable, rejected | B-05 | |
| `TestBindingTo_RejectsSelfBinding` | `To[T]()` where `T` equals the key panics at bind time | B-07 | (decision) v0: circular error at resolution time; v2: panic at bind time |
| `TestBindingTo_RejectsPointerToInterfaceTarget` | `To[*Iface2]()` panics, same family as B-01 | B-08 | |
| `TestBindingTo_ChainsThroughAnotherInterfaceBinding` | binding one interface to another bound interface resolves through the chain | B-09 | |
| `TestBindingToInstance_RejectsUntypedNil` | `ToInstance(nil)` panics with "nil instance" | B-10 | (decision) v0 crashed with a nil-pointer dereference |
| `TestBindingToInstance_AcceptsTypedNilPointer` | a typed nil `*Impl` stays accepted and resolves to a nil-valued interface | B-11 | contrast with B-10 |
| `TestBindingToInstance_TypeCheckedAtCompileTime` | `ToInstance` takes typed `T`; a type mismatch is a compile error, not a runtime panic | B-12 | (decision) moved to compile time, see compilefail_test.go |
| `TestBindingToProvider_ValidatesShape` | a non-function value panics "must be a function" | B-13 | |
| `TestBindingToProvider_ValidatesShape` | 0 results and 3 results both panic on result count | B-14 | |
| `TestBindingToProvider_ValidatesShape` | 1 result assignable to the key, accepted | B-16 | |
| `TestBindingToProvider_ValidatesShape` | 1 result not assignable to the key, rejected | B-15 | |
| `TestBindingToProvider_ValidatesShape` | 1 result assignable only via `*U`, accepted | B-17 | |
| `TestBindingToProvider_ValidatesShape` | a 2nd result not implementing `error`, rejected | B-18 | |
| `TestBindingToProvider_ValidatesShape` | a 2nd result of type `error`, accepted | B-19 | |
| `TestBindingToProvider_ArgumentsAreInjectedFreely` | any argument count/type bind-time-validates; only a call-time failure is possible | B-20 | |
| `TestBinding_RejectsASecondTarget` | all 3x3 orderings of `To`/`ToInstance`/`ToProvider` (incl. same method twice) panic naming the first call | B-21 | (decision) v0: instance > provider > to silently won; v2: panic naming the first call; named top-level test |
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
| `TestBindMessages_HaveTheDocumentedShape` | one representative rejection per entry point (`Bind`, `To`, `ToInstance`, `ToProvider`, `AnnotatedWith`, `In`, `AsEagerSingleton`, `BindInterceptor`) matches `dingo: <call>: <reason>` with both qualified type names | B-38 | one rejection per entry point, not per check |

## resolution_test.go

`GetInstance[T]`/`GetAnnotatedInstance[T]`, adaptation, failures, the `"map:"` convention.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestGetInstance_AdaptsTheResultToT` | `GetInstance[*Service]()` returns the resolver's pointer or the bound instance | R-01, G-01 | |
| `TestGetInstance_AdaptsTheResultToT` | `GetInstance[Service]()` returns a copy, not an alias of the bound instance | R-02, G-01 | |
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
| `TestGetAnnotatedInstance_MapPrefixIsASpecialLookup` | `GetAnnotatedInstance[T]("map:k")` fetches one map-binding entry by key | G-03 | (decision) undocumented internal convention |
| `TestGetAnnotatedInstance_MapPrefixIsASpecialLookup` | a user annotation literally starting with `"map:"` is silently intercepted by the same lookup | G-03 | (decision) public-API footgun |
| `TestGetInstance_PointerToInterfaceReturnsWrappedError` | `GetInstance[*Iface]()` returns an error wrapping the exported `ErrPointerToInterface` | G-04 | (decision) new plumbing, not a port of v0 behavior |
| `TestBind_DirectSliceBindingWinsOverMultibinding` | a direct `Bind[[]X]()` wins over `BindMulti[X]()` at the same injection site | K-10 | (decision) precedence between two binding mechanisms |
| `TestBind_DirectMapBindingWinsOverBindMap` | a direct `Bind[map[string]X]()` wins over `BindMap[X](...)` at the same injection site | K-11 | (decision) precedence between two binding mechanisms |

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
| `TestInjection_PointerFieldReceivesPointerValueFieldReceivesDereferencedCopy` | a `*Concrete` field gets the pointer, a `Concrete` field gets `.Elem()`, same struct | R-17 | side-by-side pair, both from the same binding |
| `TestInjection_EachInjectionIsAnIndependentCopy` | mutating one injected value-typed field does not affect another injection | R-18 | related to R-02 |
| `TestInjection_AnnotatedFieldMatchesExactAnnotationString` | `inject:"a"` matches `.AnnotatedWith(a)` by exact string equality | R-19 | |
| `TestInjection_OptionalTagLeavesUnresolvableFieldAtZeroValue` | `inject:"tag,optional"` on an unresolvable field leaves it at zero value | R-20 | |
| `TestInjection_OptionalTagLeavesUnresolvableFieldAtZeroValue` | `inject:"tag, optional"` (spaced) behaves the same | R-20 | |
| `TestInjection_InjectMethodRunsBeforeFieldInjection` | a pointer-receiver `Inject(args...)` method runs, args resolved via the injector | R-21 | |
| `TestInjection_NonPointerReceiverInjectMethodErrors` | an `Inject` method reached with a non-pointer receiver kind returns `ErrInvalidInjectReceiver` | R-22 | |
| `TestRequestInjection_DelayedDuringConfigureRunsAfterAllModules` | `RequestInjection` called during `Configure` is queued and runs only after every module has configured | R-23 | delayed-queue timing |
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
| `TestBindMulti_PerElementScopeIsIndependent` | a scoped `BindMulti`/`BindMap` element resolves through the same scope machinery, per element | MB-13 | covers both slice and map bindings |
| `TestBindMulti_OverrideOfAMultibindingIsUnsupported` | `Override[T](a)` on a `BindMulti`-only `T` hits the "unknown binding" error | MB-14 | (decision) cross-ref B-32 |

## scope_test.go

Scope identity matrix, custom scopes, eager singletons, concurrency.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestScope_IdentityMatrix` | `Singleton` caches across repeated resolution in one injector | S-01 | one cell of the {scope}x{axis} matrix |
| `TestScope_IdentityMatrix` | `ChildSingleton` caches independently per child | S-02 | |
| `TestScope_IdentityMatrix` | a registered custom `Scope` resolves; an unregistered one errors "unknown scope" | S-03 | |
| `TestScope_IdentityMatrix` | the same type with different annotations does not collide in the singleton cache | S-08 | axis of the matrix, not a separate scope |
| `TestEagerSingleton_BuildsWithoutTouchingParent` | `BuildEagerSingletons(false)` builds only the current injector's eager bindings | S-04 | side effect observed before any `GetInstance` call |
| `TestEagerSingleton_WithParentBuildsChildBeforeParent` | `BuildEagerSingletons(true)` on a child builds the child's, then the parent's | S-05 | |
| `TestSetBuildEagerSingletons_DisablesAutomaticBuild` | `SetBuildEagerSingletons(false)` skips the automatic build; a manual call builds them later | S-06 | exported-API path |
| `TestSingleton_ConcurrentResolutionConstructsExactlyOnce` | 100 goroutines resolving the same singleton-scoped binding construct it exactly once | S-07 | `atomic.Int64` counter, `sync.WaitGroup`, run under `-race` |

## child_test.go

Parent/child/sibling matrix, inherited interceptors, the global `Singleton`.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestChild_FallsBackToParentBinding` | a child with no binding for `T` resolves it from the parent | C-01 | |
| `TestChild_LocalBindingInvisibleToParentAndSiblings` | a child-only binding is invisible to the parent and to sibling children | C-02 | re-derived, not a blind port |
| `TestChild_InterceptorsInheritedFromParentUnconditionally` | a parent-registered interceptor wraps a child-resolved instance outside the child's own interceptors; a resolution started at the parent never sees child interceptors | C-03 | (decision) Guice rule, kept 2026-09-09; order asserted as a recorded `[]string` |
| `TestChild_SingletonScopeIsSharedGloballyNotPerInjector` | two independently-bound `Singleton`-scoped bindings for the same `(type, annotation)` on parent and child collide in one shared cache | C-04 | (decision) sharp edge; uses `dingo.NewInjector` twice with no helper, not parallel |

## override_test.go

`Override[T]`, unknown and multibinding overrides, duplicate detection.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestOverride_ReplacesEveryMatchingAnnotationEntry` | `Override[T](a)` replaces every prior binding matching `(T, a)`, not just the first | OV-01 | |
| `TestOverride_UnknownBindingErrorsAtInitModules` | overriding a binding that was never made errors "cannot override unknown binding" at `InitModules` | B-31, OV-02 | |
| `TestOverride_OfAMultibindingErrorsAsUnknown` | `Override[T](a)` on a `BindMulti`-only `T` hits the same "unknown binding" error | B-32, OV-03 | (decision) overriding multibindings is unsupported; cross-ref MB-14 |
| `TestOverride_PresetsAnnotationWithoutDoubleCounting` | the override-created binding does not itself get overridden again or trip duplicate detection | OV-04 | override evaluation runs before duplicate-check |
| `TestDuplicateBinding_EqualBindingsAreTolerated` | two structurally-`equal()` bindings for the same `(type, annotation)` do not error | DUP-01 | |
| `TestDuplicateBinding_UnequalBindingsErrorAtInitModules` | two non-`equal()` bindings for the same `(type, annotation)` error, naming both | DUP-02 | zero existing v0.4.1 coverage despite being unchanged runtime behavior |

## interceptor_test.go

Chain order, field-0 write, injection into the interceptor, the wrapper per resolution,
validation.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestBindInterceptor_ValidatesShape` | `T` a concrete type panics "is not an interface" | B-33 | |
| `TestBindInterceptor_ValidatesShape` | `T` a pointer to an interface panics, same family | B-33 | |
| `TestBindInterceptor_ValidatesShape` | `I` not a struct, or a struct whose first field is not `T`, panics naming both types | B-34 | (decision) v0: raw reflect panic on first use; v2: bind-time panic |
| `TestBindInterceptor_ValidatesShape` | `*I` does not implement `T` panics | B-35 | |
| `TestBindInterceptor_ValidatesShape` | `T` an interface, `I` a struct with field 0 of type `T`, `*I` implementing `T` (incl. a value-receiver own method) is accepted | B-36 | accepted counterpart |
| `TestInterceptor_ChainAppliesInRegistrationOrder` | two interceptors registered for `T` apply in registration order, each wrapping the previous | I-01 | |
| `TestInterceptor_Field0ReceivesTheBaseValuePositionally` | a named, non-embedded field 0 still receives the base value | I-02 | embedding might obscure a positional bug |
| `TestInterceptor_InjectTaggedFieldsAlsoResolve` | an interceptor with both a field-0 target and a separate `inject`-tagged field gets both resolved | I-03 | |
| `TestInterceptor_ReWrapsOnEveryResolutionEvenWithASingletonBase` | a `Singleton`-scoped base is constructed once; the interceptor wrapper is rebuilt on every resolution, so two resolutions give two wrappers (`NotSame`) over one inner instance (`Same` on field 0) | I-05 | (decision) interception runs after the scope cache, so wrapper state does not survive a resolution |
| `TestInterceptor_AppliesAcrossChildBoundary` | a parent interceptor fires on a child-resolved instance, child inner, parent outer | I-06 | cross-ref C-03, no separate mechanism |
| — | I-04 restates B-33..B-36 from the interceptor angle | I-04 | cross-ref only, no separate case |

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

## inspect_test.go

The four `Inspect*` callbacks.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestInspect_BindingCallbackFiresPerSingularBinding` | `InspectBinding` fires once per bound type with its target, provider/instance and scope | INS-01 | zero existing coverage before this file |
| `TestInspect_MultiBindingCallbackFiresPerEntryWithSharedIndexSpace` | `InspectMultiBinding` fires per entry with its slice index, shared across annotations | INS-02 | (decision) index space is not filtered by annotation |
| `TestInspect_MapBindingCallbackFiresPerKey` | `InspectMapBinding` fires once per map-binding key | INS-03 | |
| `TestInspect_ParentCallbackFiresOnlyWhenAParentExists` | `InspectParent` fires only when `injector.parent != nil`, no auto-recursion | INS-04 | |
| `TestInspect_NilCallbacksAreSkipped` | an `Inspector` with only one callback set does not panic and does not invoke the others | INS-05 | |
| `TestInspect_ProviderAndInstanceArgsAreNilWhenAbsent` | `provider`/`instance` arguments are nil for a target-less or `To`-only binding | INS-06 | tests presence and absence for both |

## compilefail_test.go

Drives `v2/testdata/compilefail/cases` with `GOWORK=off`, so the corpus module's two `replace` directives apply.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestCompileFail_MatchesExpectedErrors` | `to_instance_wrong_type`: `ToInstance` with a value of the wrong type fails to build | B-12 | v0's runtime panic is now a compile error |
| `TestCompileFail_MatchesExpectedErrors` | `get_instance_wrong_assignment`: `GetInstance[greeter]()` assigned to `string` fails to build | — | |
| `TestCompileFail_MatchesExpectedErrors` | `bind_missing_type_argument`: `Bind()` without `[T]` fails to build | — | |
| `TestCompileFail_MatchesExpectedErrors` | `to_missing_type_argument`: `.To()` without `[U]` fails to build | — | |
| `TestCompileFail_MatchesExpectedErrors` | `interceptor_one_type_argument`: `BindInterceptor[greeter]()` fails to build | — | |
| `TestCompileFail_MatchesExpectedErrors` | `to_unrelated_type_compiles`: `Bind[A]().To[B]()` with unrelated `B` builds | B-05 | (decision) accepted counterpart; documents the compile-time/bind-time boundary, must build |

## readme_test.go

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestReadme_UsesV2CallShapes` | no fenced Go block in `v2/README.md` contains a v0 call shape (`Bind(`, `BindMulti(`, `BindMap(`, `Override(`, `BindInterceptor(`, `GetInstance(`, `GetAnnotatedInstance(`, `.To(` without a type argument) | — | drift guard; the migration table uses inline code, not fences |

## catalogue_test.go

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestCatalogue_EveryIDIsReferencedOrExcused` | every ID in `testdata/catalogue.txt` is referenced by a `*_test.go` file under the v2 module, `compat/` included, or has a recorded reason | — | the gate this appendix exists to satisfy |

## compat/compat_test.go (package compat_test)

Two APIs on one engine. Every rule that has two directions runs both as subtests: a v0 module on
a v2 injector and a v2 module on a v0 injector. Module identity is asserted by counting
`Configure` calls per module value; injection timing is asserted from inside `Configure`.

| Test function | Case | Covers | Note |
|---|---|---|---|
| `TestInjector_IsIdempotentPerEngine` | `compat.Injector(e)` twice returns the same facade; `compat.Engine(compat.Injector(e))` is `e` | X-01 | `assert.Same` |
| `TestInjector_IsDistinctPerChildEngine` | the facades of an engine and of its `Child()` differ and each wraps its own engine | X-02 | |
| `TestInjector_SharesBindingsWithTheV0API` | a v0 `Bind(new(I)).To(Impl{})` on the engine resolves through `GetInstance[I]` on the facade, and a v2 `Bind[I]().To[Impl]()` resolves through the engine's `GetInstance(new(I))` | X-03 | both directions |
| `TestMultibindingsAndMapBindings_ShareAcrossKinds` | a v0 `BindMulti(new(I)).To(A{})` entry and a v2 `BindMulti[I]().To[B]()` entry arrive together in a `[]I` field of a module of either kind and in `GetInstance[[]I]()`; the same for `BindMap` entries in `map[string]I` | X-17 | both directions |
| `TestToV0_RunsAV2ModuleOnAV0Injector` | `dingo.NewInjector(compat.ToV0(m))` (root) configures `m` once and its bindings resolve through the root API | X-04 | |
| `TestFromV0_RunsAV0ModuleOnAV2Injector` | `dingo.NewInjector(compat.FromV0(m))` (v2) configures `m` once and its bindings resolve through the v2 API | X-05 | |
| `TestAdaptedModules_KeepTheirIdentity` | the same module value passed direct and adapted, and `ToV0(FromV0(m))`, configure once; two distinct module types both configure; `ModuleFunc` closures of either package are distinct by value | X-06 | (decision) identity is the innermost module; named top-level test |
| `TestAdaptedModules_DependenciesResolveAcrossKinds` | a v2 `Depender` under a v0 injector and a v0 `Depender` under a v2 injector: every dependency configures once, before the dependent | X-07 | |
| `TestAdaptedModules_FieldsAreInjectedBeforeConfigure` | an adapted module of either kind reads its `inject`-tagged fields inside `Configure`, on either injector | X-08 | (decision) same timing as native modules; named top-level test; the prototype's known gap, closed by `InitModules` injecting the innermost module |
| `TestAdaptedModules_InjectionFailureReturnsFromInitModules` | an adapted module of either kind with an `inject` field that cannot be resolved: `InitModules` on either injector returns the engine's `initmodules: injection into %q failed` error naming the innermost module's type, `TryModule` of either package returns it, and `Configure` never runs | X-19 | (decision) same failure path as native modules |
| `TestSelfBinding_EachKindReceivesItsOwnInjector` | a `*dingo.Injector` (root) field receives the engine, a `*dingo.Injector` (v2) field receives its facade; inside a child, the child's | X-09 | |
| `TestScopes_AreSharedAcrossKinds` | a `Singleton`-scoped binding made through one API and resolved through the other is one instance; a scope registered with `BindScope` on the engine serves `In` on the facade | X-10 | `dingo.Singleton` is one object in both packages |
| `TestInterceptors_ApplyAcrossKinds` | a v2 `BindInterceptor[I, W]` wraps a v0-bound value; a v0 `BindInterceptor(new(I), W{})` wraps a v2-bound value | X-11 | |
| `TestBindTimeChecks_SurfaceThroughV0EntryPoints` | a v2 misuse inside a `ToV0` module: the root's `TryModule` returns an error wrapping `v2.ErrInvalidBinding`; the root's `NewInjector` panics with it | X-12 | |
| `TestEngineErrors_KeepV0WordingThroughTheFacade` | an unbound-interface resolution panic and a `DUP-02` duplicate-binding error reached through the v2 facade carry the engine's v0 wording, not the `dingo: <call>: <reason>` shape | X-18 | (decision) engine messages stay the engine's |
| `TestConfigLoop_V0RuntimeTypedBindingsReachV2Fields` | v0 `Bind(v).AnnotatedWith("config:k").ToInstance(v)` for `string`, `bool`, `float64` is read by a v2 module's `inject:"config:k"` fields and by `GetAnnotatedInstance[T]` | X-13 | the Flamingo `area.go` shape during steps 0 to 2 |
| `TestInspect_TranslatesTheParentToItsFacade` | v2 `Inspect` lists bindings of both origins and hands `InspectParent` the parent's facade, the same value `compat.Injector(parentEngine)` returns | X-14 | |
| `TestTryModule_AcceptsAdaptedModules` | v2 `TryModule(compat.FromV0(m))` and root `TryModule(compat.ToV0(m))` both skip eager singletons and return bind errors | X-15 | |
| `TestSentinels_AreSharedValues` | `errors.Is` holds across packages for `ErrPointerToInterface`, `ErrInitModules`, `ErrInvalidInjectReceiver`, `ErrModuleCycle`, `ErrModuleSort` | X-16 | |

## compat/example_test.go (package compat_test)

| Test function | Case | Covers | Note |
|---|---|---|---|
| `Example_v0Module` | a module with the v0 `Configure` signature binds through `compat.Injector(injector)` | — | README "Using v2 next to v0", step 0 of the Flamingo path |
| `ExampleToV0` | a v2 module listed among v0 modules with `compat.ToV0` | — | the application-list shape |

## Pinned in the root module

Engine-internal behavior with no observable effect through the v2 API, or needing package-level
state, stays in the root module's suite. These IDs are excused in `testdata/catalogue.txt` with
the root test as the reason.

| ID | Root test | Note |
|---|---|---|
| DUP-03 | `binding_test.go` `TestBinding_equal` | existing |
| R-25 | `circular_test.go` `TestDingoCircular` | existing, renamed by the root PR |
| R-27 | `tracing_test.go` `TestInjectionTracing_LogsFieldSetsAndResolutionsWhenEnabled` (package `dingo`) | added by the root PR; restores the switch in `t.Cleanup` |
| M-03, M-05 white-box halves | `module_test.go` `TestModGraph_Sorted`, `Test_resolveDependencies` | existing; the public views are covered in v2's `module_test.go` |

## Excused IDs

| ID | Reason | Comment location |
|---|---|---|
| R-26 | A tracing-disabled cycle through a `…Provider`-typed self-reference stack-overflows the process instead of returning a catchable panic; a test would crash the test binary, not fail it. | `testdata/catalogue.txt`, the reason on the ID's line |
| M-08 (add-failure branch) | `ErrInitModules`'s "failed adding modules" branch wraps an error from `mg.Add`, whose only error source is `Depends() []Module`, a signature that cannot itself return an error; the branch is unreachable through the public API. | root `module.go`, `// coverage: unreachable through the public API` above the add-failure branch in `InitModules`, added by the root PR |
| DUP-03, R-25, R-27 | Engine-internal; pinned by the root tests listed under "Pinned in the root module". | `testdata/catalogue.txt`, the reason on each ID's line |
| I-04 | Restates B-33..B-36 from the interceptor angle; no independent case. | `testdata/catalogue.txt`, the reason on the ID's line |

## ID count

159 IDs: B-01..B-40, K-01..K-11, R-01..R-31 with R-11a, MB-01..MB-14, S-01..S-08, C-01..C-04,
OV-01..OV-04, DUP-01..DUP-03, I-01..I-06, M-01..M-08, INS-01..INS-06, G-01..G-04, X-01..X-19.
B-39 (the zero `Binding[T]`), B-40 (the zero `Injector`), K-10 and K-11 (precedence of direct
slice and map bindings) and X-01..X-19 (interop) were added by the spec after the behavior list
was derived. Every ID appears in a Covers column above or under "Pinned in the root module";
R-26, the M-08 add-failure branch, DUP-03, R-25, R-27 and I-04 appear again under Excused IDs. `testdata/catalogue.txt` is generated from this file's Covers columns.
