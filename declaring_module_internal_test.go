package dingo

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	declaringIface     interface{ declared() }
	declaringImpl      struct{}
	declaringOtherImpl struct{}

	// Each module type carries its configuration as a field, so pointers to
	// two instances never compare equal and tests can define bindings inline.
	declaringModuleA struct{ configure func(*Injector) }
	declaringModuleB struct{ configure func(*Injector) }
)

func (*declaringImpl) declared()      {}
func (*declaringOtherImpl) declared() {}

func (m *declaringModuleA) Configure(injector *Injector) { m.configure(injector) }
func (m *declaringModuleB) Configure(injector *Injector) { m.configure(injector) }

func TestInitModules_DeclaringModule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// setup returns the modules to initialize and the expected declaring
		// module for every binding of the resulting injector.
		setup func() (modules []Module, want map[string]Module)
	}{
		{
			name: "single bindings record the module that created them",
			setup: func() ([]Module, map[string]Module) {
				a := &declaringModuleA{configure: func(injector *Injector) {
					injector.Bind(new(declaringIface)).To(declaringImpl{})
					injector.Bind(new(string)).ToInstance("a")
				}}
				b := &declaringModuleB{configure: func(injector *Injector) {
					injector.Bind(new(int)).ToProvider(func() int { return 1 })
				}}

				return []Module{a, b}, map[string]Module{
					"Bind dingo.Injector[0]":      nil,
					"Bind dingo.declaringIface[0]": a,
					"Bind string[0]":               a,
					"Bind int[0]":                  b,
				}
			},
		},
		{
			name: "multibindings and mapbindings record the module that created them",
			setup: func() ([]Module, map[string]Module) {
				a := &declaringModuleA{configure: func(injector *Injector) {
					injector.BindMulti(new(declaringIface)).To(declaringImpl{})
					injector.BindMap(new(string), "a").ToInstance("a")
				}}
				b := &declaringModuleB{configure: func(injector *Injector) {
					injector.BindMulti(new(declaringIface)).ToProvider(func() *declaringImpl { return nil })
					injector.BindMap(new(string), "b").ToInstance("b")
				}}

				return []Module{a, b}, map[string]Module{
					"Bind dingo.Injector[0]":           nil,
					"BindMulti dingo.declaringIface[0]": a,
					"BindMulti dingo.declaringIface[1]": b,
					`BindMap string["a"]`:               a,
					`BindMap string["b"]`:               b,
				}
			},
		},
		{
			name: "a ModuleFunc is recorded as the declaring module",
			setup: func() ([]Module, map[string]Module) {
				f := ModuleFunc(func(injector *Injector) {
					injector.Bind(new(string)).ToInstance("f")
				})
				g := ModuleFunc(func(injector *Injector) {
					injector.BindMulti(new(string)).ToInstance("g")
					injector.BindMap(new(string), "g").ToInstance("g")
				})

				return []Module{f, g}, map[string]Module{
					"Bind dingo.Injector[0]": nil,
					"Bind string[0]":         f,
					"BindMulti string[0]":    g,
					`BindMap string["g"]`:    g,
				}
			},
		},
		{
			name: "an override records the overriding module in every slot it replaces",
			setup: func() ([]Module, map[string]Module) {
				a := &declaringModuleA{configure: func(injector *Injector) {
					injector.Bind(new(string)).ToInstance("a")
				}}
				b := &declaringModuleB{configure: func(injector *Injector) {
					injector.Override(new(string), "").ToInstance("b")
				}}

				return []Module{a, b}, map[string]Module{
					"Bind dingo.Injector[0]": nil,
					"Bind string[0]":         b,
					"Bind string[1]":         b,
				}
			},
		},
		{
			name: "identical bindings from two modules are still allowed",
			setup: func() ([]Module, map[string]Module) {
				a := &declaringModuleA{configure: func(injector *Injector) {
					injector.Bind(new(declaringIface)).To(declaringImpl{})
				}}
				b := &declaringModuleB{configure: func(injector *Injector) {
					injector.Bind(new(declaringIface)).To(declaringImpl{})
				}}

				return []Module{a, b}, map[string]Module{
					"Bind dingo.Injector[0]":       nil,
					"Bind dingo.declaringIface[0]": a,
					"Bind dingo.declaringIface[1]": b,
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			modules, want := test.setup()

			injector, err := NewInjector(modules...)
			require.NoError(t, err)

			assertDeclaringModules(t, want, declaringModulesOf(injector))
		})
	}
}

func TestBinding_equalIgnoresTheDeclaringModule(t *testing.T) {
	t.Parallel()

	a := new(declaringModuleA)
	b := new(declaringModuleB)
	instance := new(declaringImpl)
	provider := func() string { return "provider" }

	bind := func(what any, module Module) *Binding {
		return &Binding{typeof: reflect.TypeOf(what).Elem(), declaredBy: module}
	}

	tests := []struct {
		name        string
		left, right *Binding
		equal       bool
	}{
		{
			name:  "same target",
			left:  bind(new(declaringIface), a).To(declaringImpl{}),
			right: bind(new(declaringIface), b).To(declaringImpl{}),
			equal: true,
		},
		{
			name:  "same instance",
			left:  bind(new(declaringIface), a).ToInstance(instance),
			right: bind(new(declaringIface), b).ToInstance(instance),
			equal: true,
		},
		{
			name:  "same provider",
			left:  bind(new(string), a).ToProvider(provider),
			right: bind(new(string), b).ToProvider(provider),
			equal: true,
		},
		{
			name:  "same annotation, scope and eagerness",
			left:  bind(new(string), a).AnnotatedWith("x").AsEagerSingleton(),
			right: bind(new(string), b).AnnotatedWith("x").AsEagerSingleton(),
			equal: true,
		},
		{
			name:  "different bound type",
			left:  bind(new(string), a),
			right: bind(new(int), b),
		},
		{
			name:  "different target",
			left:  bind(new(declaringIface), a).To(declaringImpl{}),
			right: bind(new(declaringIface), b).To(declaringOtherImpl{}),
		},
		{
			name:  "different instance",
			left:  bind(new(string), a).ToInstance("left"),
			right: bind(new(string), b).ToInstance("right"),
		},
		{
			name:  "different provider",
			left:  bind(new(string), a).ToProvider(provider),
			right: bind(new(string), b).ToProvider(func() string { return "other" }),
		},
		{
			name:  "different annotation",
			left:  bind(new(string), a).AnnotatedWith("x"),
			right: bind(new(string), b).AnnotatedWith("y"),
		},
		{
			name:  "different scope",
			left:  bind(new(string), a).In(Singleton),
			right: bind(new(string), b).In(ChildSingleton),
		},
		{
			name:  "different eagerness",
			left:  bind(new(string), a).In(Singleton),
			right: bind(new(string), b).AsEagerSingleton(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.equal, test.left.equal(test.right))
		})
	}
}

// declaringModulesOf returns the declaring module of every binding the
// injector holds, keyed by the bind method, the bound type and the position.
func declaringModulesOf(injector *Injector) map[string]Module {
	got := make(map[string]Module)

	for typ, bindings := range injector.bindings {
		for i, binding := range bindings {
			got[fmt.Sprintf("Bind %s[%d]", typ, i)] = binding.declaredBy
		}
	}

	for typ, bindings := range injector.multibindings {
		for i, binding := range bindings {
			got[fmt.Sprintf("BindMulti %s[%d]", typ, i)] = binding.declaredBy
		}
	}

	for typ, bindings := range injector.mapbindings {
		for key, binding := range bindings {
			got[fmt.Sprintf("BindMap %s[%q]", typ, key)] = binding.declaredBy
		}
	}

	return got
}

func assertDeclaringModules(t *testing.T, want, got map[string]Module) {
	t.Helper()

	assert.ElementsMatch(t, slices.Collect(maps.Keys(want)), slices.Collect(maps.Keys(got)), "bindings")

	for binding, module := range want {
		assertSameModule(t, module, got[binding], binding)
	}
}

// assertSameModule checks module identity. Equal cannot be used: it compares
// funcs as always different and distinct module pointers by their contents.
func assertSameModule(t *testing.T, want, got Module, msg string) {
	t.Helper()

	if want == nil {
		assert.Nil(t, got, msg)

		return
	}

	if !assert.NotNil(t, got, "%s: no declaring module", msg) {
		return
	}

	if _, ok := want.(ModuleFunc); ok {
		assert.True(t, reflect.ValueOf(want) == reflect.ValueOf(got), msg)

		return
	}

	assert.Same(t, want, got, msg)
}
