package dingo_test

import (
	"maps"
	"reflect"
	"slices"
	"testing"

	"flamingo.me/dingo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func isInjectorBinding(of reflect.Type) bool {
	return of == reflect.TypeFor[dingo.Injector]()
}

type inspectedBinding struct {
	of         reflect.Type
	annotation string
	to         reflect.Type
	provider   *reflect.Value
	instance   *reflect.Value
	scope      dingo.Scope
}

type inspectSpeaker interface {
	Speak() string
}

type inspectSpeakerImpl struct {
	Word string
}

func (s *inspectSpeakerImpl) Speak() string { return s.Word }

type inspectHolder struct {
	Token string
}

func TestInspect_BindingCallbackFiresPerSingularBinding(t *testing.T) {
	t.Parallel()

	injector := newInjector(t)
	held := &inspectHolder{Token: "held"}

	injector.Bind((*inspectSpeaker)(nil)).To(new(inspectSpeakerImpl)).In(dingo.Singleton)
	injector.Bind(new(inspectHolder)).ToInstance(held)

	byType := map[reflect.Type]inspectedBinding{}

	injector.Inspect(dingo.Inspector{
		InspectBinding: func(of reflect.Type, annotation string, to reflect.Type, provider, instance *reflect.Value, scope dingo.Scope) {
			if isInjectorBinding(of) {
				return
			}

			byType[of] = inspectedBinding{
				of:         of,
				annotation: annotation,
				to:         to,
				provider:   provider,
				instance:   instance,
				scope:      scope,
			}
		},
	})

	want := []reflect.Type{
		reflect.TypeFor[inspectSpeaker](),
		reflect.TypeFor[inspectHolder](),
	}
	assert.Len(t, byType, len(want))

	speaker := byType[reflect.TypeFor[inspectSpeaker]()]
	assert.Equal(t, reflect.TypeFor[inspectSpeakerImpl](), speaker.to)
	assert.Nil(t, speaker.provider)
	assert.Nil(t, speaker.instance)
	assert.Same(t, dingo.Singleton, speaker.scope)
	assert.Empty(t, speaker.annotation)

	holder := byType[reflect.TypeFor[inspectHolder]()]
	assert.Nil(t, holder.to)
	assert.Nil(t, holder.provider)
	require.NotNil(t, holder.instance)
	assert.Same(t, held, holder.instance.Interface())
	assert.Nil(t, holder.scope)
}

type inspectMulti interface {
	inspectMulti()
}

type inspectMultiAlpha struct{}

func (*inspectMultiAlpha) inspectMulti() {}

type inspectMultiBeta struct{}

func (*inspectMultiBeta) inspectMulti() {}

type inspectedMulti struct {
	index      int
	annotation string
	to         reflect.Type
}

func TestInspect_MultiBindingCallbackFiresPerEntryWithSharedIndexSpace(t *testing.T) {
	t.Parallel()

	injector := newInjector(t)
	injector.BindMulti((*inspectMulti)(nil)).AnnotatedWith("alpha").To(new(inspectMultiAlpha))
	injector.BindMulti((*inspectMulti)(nil)).AnnotatedWith("beta").To(new(inspectMultiBeta))

	byAnnotation := map[string]inspectedMulti{}

	injector.Inspect(dingo.Inspector{
		InspectMultiBinding: func(of reflect.Type, index int, annotation string, to reflect.Type, provider, instance *reflect.Value, scope dingo.Scope) {
			byAnnotation[annotation] = inspectedMulti{index: index, annotation: annotation, to: to}

			assert.Equal(t, reflect.TypeFor[inspectMulti](), of)
			assert.Nil(t, provider)
			assert.Nil(t, instance)
			assert.Nil(t, scope)
		},
	})

	assert.Equal(t, []string{"alpha", "beta"}, slices.Sorted(maps.Keys(byAnnotation)))
	assert.Equal(t, 0, byAnnotation["alpha"].index)
	assert.Equal(t, 1, byAnnotation["beta"].index)
	assert.Equal(t, reflect.TypeFor[inspectMultiAlpha](), byAnnotation["alpha"].to)
	assert.Equal(t, reflect.TypeFor[inspectMultiBeta](), byAnnotation["beta"].to)
}

type inspectMapped interface {
	inspectMapped()
}

type inspectMappedHome struct{}

func (*inspectMappedHome) inspectMapped() {}

type inspectMappedWork struct{}

func (*inspectMappedWork) inspectMapped() {}

type inspectedMap struct {
	key        string
	annotation string
	to         reflect.Type
}

func TestInspect_MapBindingCallbackFiresPerKey(t *testing.T) {
	t.Parallel()

	injector := newInjector(t)
	injector.BindMap((*inspectMapped)(nil), "home").AnnotatedWith("area").To(new(inspectMappedHome))
	injector.BindMap((*inspectMapped)(nil), "work").AnnotatedWith("area").To(new(inspectMappedWork))

	byKey := map[string]inspectedMap{}

	injector.Inspect(dingo.Inspector{
		InspectMapBinding: func(of reflect.Type, key string, annotation string, to reflect.Type, provider, instance *reflect.Value, scope dingo.Scope) {
			byKey[key] = inspectedMap{key: key, annotation: annotation, to: to}

			assert.Equal(t, reflect.TypeFor[inspectMapped](), of)
			assert.Nil(t, provider)
			assert.Nil(t, instance)
			assert.Nil(t, scope)
		},
	})

	assert.Equal(t, []string{"home", "work"}, slices.Sorted(maps.Keys(byKey)))
	assert.Equal(t, "area", byKey["home"].annotation)
	assert.Equal(t, "area", byKey["work"].annotation)
	assert.Equal(t, reflect.TypeFor[inspectMappedHome](), byKey["home"].to)
	assert.Equal(t, reflect.TypeFor[inspectMappedWork](), byKey["work"].to)
}

// inspectLineageWidget is bound on a parent and on a middle injector to show that
// Inspect on the grandchild does not report either binding.
type inspectLineageWidget struct{}

func TestInspect_ParentCallbackFiresOnlyWhenAParentExists(t *testing.T) {
	t.Parallel()

	t.Run("child receives its parent once and not the grandparent bindings", func(t *testing.T) {
		t.Parallel()

		top := newInjector(t)
		top.Bind(new(inspectLineageWidget)).AnnotatedWith("on-top")

		middle := childOf(t, top)
		middle.Bind(new(inspectLineageWidget)).AnnotatedWith("on-middle")

		grand := childOf(t, middle)

		var parents []*dingo.Injector

		annotations := map[string]struct{}{}

		grand.Inspect(dingo.Inspector{
			InspectParent: func(parent *dingo.Injector) {
				parents = append(parents, parent)
			},
			InspectBinding: func(of reflect.Type, annotation string, to reflect.Type, provider, instance *reflect.Value, scope dingo.Scope) {
				if isInjectorBinding(of) {
					return
				}

				annotations[annotation] = struct{}{}

				assert.Nil(t, to)
				assert.Nil(t, provider)
				assert.Nil(t, instance)
				assert.Nil(t, scope)
			},
		})

		require.Len(t, parents, 1)
		assert.Same(t, middle, parents[0])
		assert.Empty(t, annotations)
	})

	t.Run("NewInjector does not invoke InspectParent", func(t *testing.T) {
		t.Parallel()

		injector, err := dingo.NewInjector()
		require.NoError(t, err)

		var calls int

		injector.Inspect(dingo.Inspector{
			InspectParent: func(*dingo.Injector) {
				calls++
			},
		})
		assert.Zero(t, calls)
	})
}

type inspectNilWidget struct{}

func TestInspect_NilCallbacksAreSkipped(t *testing.T) {
	t.Parallel()

	parent := newInjector(t)
	child := childOf(t, parent)
	child.Bind(new(inspectNilWidget)).AnnotatedWith("one")
	child.BindMulti(new(inspectNilWidget)).AnnotatedWith("one")
	child.BindMap(new(inspectNilWidget), "k").AnnotatedWith("one")

	// Each Inspect below sets one callback. The other three are nil.
	// Invoking a nil func would panic; this injector has data for every callback.
	sawParent := false

	child.Inspect(dingo.Inspector{
		InspectParent: func(got *dingo.Injector) {
			sawParent = true

			assert.Same(t, parent, got)
		},
	})
	assert.True(t, sawParent)

	sawBinding := false

	child.Inspect(dingo.Inspector{
		InspectBinding: func(of reflect.Type, annotation string, _ reflect.Type, _, _ *reflect.Value, _ dingo.Scope) {
			if isInjectorBinding(of) {
				return
			}

			sawBinding = true

			assert.Equal(t, "one", annotation)
		},
	})
	assert.True(t, sawBinding)

	sawMulti := false

	child.Inspect(dingo.Inspector{
		InspectMultiBinding: func(_ reflect.Type, _ int, annotation string, _ reflect.Type, _, _ *reflect.Value, _ dingo.Scope) {
			sawMulti = true

			assert.Equal(t, "one", annotation)
		},
	})
	assert.True(t, sawMulti)

	sawMap := false

	child.Inspect(dingo.Inspector{
		InspectMapBinding: func(_ reflect.Type, key string, annotation string, _ reflect.Type, _, _ *reflect.Value, _ dingo.Scope) {
			sawMap = true

			assert.Equal(t, "k", key)
			assert.Equal(t, "one", annotation)
		},
	})
	assert.True(t, sawMap)
}

type inspectBare struct{}

func inspectProvidedSpeaker() *inspectSpeakerImpl {
	return &inspectSpeakerImpl{Word: "spoken"}
}

func TestInspect_ProviderAndInstanceArgsAreNilWhenAbsent(t *testing.T) {
	t.Parallel()

	injector := newInjector(t)
	held := &inspectSpeakerImpl{Word: "ready"}

	injector.Bind(new(inspectBare)).AnnotatedWith("bare")
	injector.Bind((*inspectSpeaker)(nil)).AnnotatedWith("to").To(new(inspectSpeakerImpl))
	injector.Bind((*inspectSpeaker)(nil)).AnnotatedWith("instance").ToInstance(held)
	injector.Bind((*inspectSpeaker)(nil)).AnnotatedWith("provider").ToProvider(inspectProvidedSpeaker)

	byAnnotation := map[string]inspectedBinding{}

	injector.Inspect(dingo.Inspector{
		InspectBinding: func(of reflect.Type, annotation string, to reflect.Type, provider, instance *reflect.Value, scope dingo.Scope) {
			if isInjectorBinding(of) {
				return
			}

			byAnnotation[annotation] = inspectedBinding{
				of:         of,
				annotation: annotation,
				to:         to,
				provider:   provider,
				instance:   instance,
				scope:      scope,
			}
		},
	})

	assert.Equal(t, []string{"bare", "instance", "provider", "to"}, slices.Sorted(maps.Keys(byAnnotation)))

	bare := byAnnotation["bare"]
	assert.Equal(t, reflect.TypeFor[inspectBare](), bare.of)
	assert.Nil(t, bare.to)
	assert.Nil(t, bare.provider)
	assert.Nil(t, bare.instance)
	assert.Nil(t, bare.scope)

	toOnly := byAnnotation["to"]
	assert.Equal(t, reflect.TypeFor[inspectSpeaker](), toOnly.of)
	assert.Equal(t, reflect.TypeFor[inspectSpeakerImpl](), toOnly.to)
	assert.Nil(t, toOnly.provider)
	assert.Nil(t, toOnly.instance)

	instanceOnly := byAnnotation["instance"]
	assert.Nil(t, instanceOnly.to)
	assert.Nil(t, instanceOnly.provider)
	require.NotNil(t, instanceOnly.instance)
	assert.Same(t, held, instanceOnly.instance.Interface())

	providerOnly := byAnnotation["provider"]
	assert.Nil(t, providerOnly.to)
	assert.Nil(t, providerOnly.instance)
	require.NotNil(t, providerOnly.provider)
	assert.Equal(t, reflect.Func, providerOnly.provider.Kind())
	assert.Equal(t, reflect.ValueOf(inspectProvidedSpeaker).Pointer(), providerOnly.provider.Pointer())
}
