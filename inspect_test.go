package dingo_test

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"testing"

	"flamingo.me/dingo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	inspectIface interface{ inspected() }
	inspectImpl  struct{}

	// Each module type carries its configuration as a field, so pointers to
	// two instances never compare equal and tests can define bindings inline.
	inspectModuleA struct{ configure func(*dingo.Injector) }
	inspectModuleB struct{ configure func(*dingo.Injector) }
)

func (*inspectImpl) inspected() {}

func (m *inspectModuleA) Configure(injector *dingo.Injector) { m.configure(injector) }
func (m *inspectModuleB) Configure(injector *dingo.Injector) { m.configure(injector) }

func TestInspect_DeclaringModule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// setup returns the injector to inspect and the declaring module
		// expected for every binding it reports.
		setup func(t *testing.T) (*dingo.Injector, map[string]dingo.Module)
	}{
		{
			name: "single bindings report the module that declared them",
			setup: func(t *testing.T) (*dingo.Injector, map[string]dingo.Module) {
				t.Helper()

				a := &inspectModuleA{configure: func(injector *dingo.Injector) {
					injector.Bind(new(inspectIface)).To(inspectImpl{})
					injector.Bind(new(string)).ToInstance("a")
				}}
				b := &inspectModuleB{configure: func(injector *dingo.Injector) {
					injector.Bind(new(int)).ToProvider(func() int { return 1 })
					injector.Bind(new(string)).AnnotatedWith("b").ToInstance("b")
				}}

				injector, err := dingo.NewInjector(a, b)
				require.NoError(t, err)

				return injector, map[string]dingo.Module{
					"Bind dingo.Injector[0]":              nil,
					"Bind dingo_test.inspectIface[0]":     a,
					"Bind string[0]":                      a,
					"Bind int[0]":                         b,
					"Bind string annotated with \"b\"[0]": b,
				}
			},
		},
		{
			name: "multibindings report the module that declared them",
			setup: func(t *testing.T) (*dingo.Injector, map[string]dingo.Module) {
				t.Helper()

				a := &inspectModuleA{configure: func(injector *dingo.Injector) {
					injector.BindMulti(new(inspectIface)).To(inspectImpl{})
				}}
				b := &inspectModuleB{configure: func(injector *dingo.Injector) {
					injector.BindMulti(new(inspectIface)).ToProvider(func() *inspectImpl { return nil })
					injector.BindMulti(new(inspectIface)).AnnotatedWith("b").ToInstance(new(inspectImpl))
				}}

				injector, err := dingo.NewInjector(a, b)
				require.NoError(t, err)

				return injector, map[string]dingo.Module{
					"Bind dingo.Injector[0]":                                    nil,
					"BindMulti dingo_test.inspectIface[0]":                      a,
					"BindMulti dingo_test.inspectIface[1]":                      b,
					"BindMulti dingo_test.inspectIface annotated with \"b\"[2]": b,
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			injector, want := test.setup(t)

			assertInspectedModules(t, want, inspectDeclaringModules(injector))
		})
	}
}

// inspectDeclaringModules returns the declaring module Inspect reports for
// every binding, keyed by the bind method, the bound type, the annotation and
// the position. Single bindings of one type are numbered in binding order.
func inspectDeclaringModules(injector *dingo.Injector) map[string]dingo.Module {
	got := make(map[string]dingo.Module)
	count := make(map[string]int)

	injector.Inspect(dingo.Inspector{
		InspectBinding: func(of reflect.Type, annotation string, _ reflect.Type, _, _ *reflect.Value, _ dingo.Scope, declaredBy dingo.Module) {
			binding := inspectKey("Bind", of, annotation)
			got[fmt.Sprintf("%s[%d]", binding, count[binding])] = declaredBy
			count[binding]++
		},
		InspectMultiBinding: func(of reflect.Type, index int, annotation string, _ reflect.Type, _, _ *reflect.Value, _ dingo.Scope, declaredBy dingo.Module) {
			got[fmt.Sprintf("%s[%d]", inspectKey("BindMulti", of, annotation), index)] = declaredBy
		},
	})

	return got
}

func inspectKey(method string, of reflect.Type, annotation string) string {
	if annotation == "" {
		return fmt.Sprintf("%s %s", method, of)
	}

	return fmt.Sprintf("%s %s annotated with %q", method, of, annotation)
}

func assertInspectedModules(t *testing.T, want, got map[string]dingo.Module) {
	t.Helper()

	assert.ElementsMatch(t, slices.Collect(maps.Keys(want)), slices.Collect(maps.Keys(got)), "bindings")

	for binding, module := range want {
		assertSameInspectedModule(t, module, got[binding], binding)
	}
}

// assertSameInspectedModule checks module identity. Equal cannot be used: it
// compares funcs as always different and distinct module pointers by their
// contents.
func assertSameInspectedModule(t *testing.T, want, got dingo.Module, msg string) {
	t.Helper()

	if want == nil {
		assert.Nil(t, got, msg)

		return
	}

	if !assert.NotNil(t, got, "%s: no declaring module", msg) {
		return
	}

	if _, ok := want.(dingo.ModuleFunc); ok {
		assert.True(t, reflect.ValueOf(want) == reflect.ValueOf(got), msg)

		return
	}

	assert.Same(t, want, got, msg)
}
