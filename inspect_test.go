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
	inspectIface     interface{ inspected() }
	inspectImpl      struct{}
	inspectOtherImpl struct{}

	// Each module type carries its configuration as a field, so pointers to
	// two instances never compare equal and tests can define bindings inline.
	inspectModuleA struct{ configure func(*dingo.Injector) }
	inspectModuleB struct{ configure func(*dingo.Injector) }

	// inspectInjectedModule binds from its Inject method as well.
	inspectInjectedModule struct {
		inject, configure func(*dingo.Injector)
	}

	// inspectUnresolvableModule fails injection, so it is never configured.
	inspectUnresolvableModule struct {
		Missing inspectIface `inject:""`
	}

	// inspectDelayed binds while its delayed injection runs.
	inspectDelayed struct{ inject func(*dingo.Injector) }
)

func (*inspectImpl) inspected()      {}
func (*inspectOtherImpl) inspected() {}

func (m *inspectModuleA) Configure(injector *dingo.Injector) { m.configure(injector) }
func (m *inspectModuleB) Configure(injector *dingo.Injector) { m.configure(injector) }

func (m *inspectInjectedModule) Inject(injector *dingo.Injector)    { m.inject(injector) }
func (m *inspectInjectedModule) Configure(injector *dingo.Injector) { m.configure(injector) }

func (*inspectUnresolvableModule) Configure(*dingo.Injector) {}

func (d *inspectDelayed) Inject(injector *dingo.Injector) { d.inject(injector) }

// inspectFuncModuleA and inspectFuncModuleB are used as ModuleFunc modules.
func inspectFuncModuleA(injector *dingo.Injector) { injector.Bind(new(string)).ToInstance("a") }
func inspectFuncModuleB(injector *dingo.Injector) { injector.Bind(new(string)).ToInstance("b") }

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
		{
			name: "mapbindings report the module that declared them",
			setup: func(t *testing.T) (*dingo.Injector, map[string]dingo.Module) {
				t.Helper()

				a := &inspectModuleA{configure: func(injector *dingo.Injector) {
					injector.BindMap(new(string), "a").ToInstance("a")
				}}
				b := &inspectModuleB{configure: func(injector *dingo.Injector) {
					injector.BindMap(new(string), "b").ToProvider(func() string { return "b" })
					injector.BindMap(new(inspectIface), "b").AnnotatedWith("b").To(inspectImpl{})
				}}

				injector, err := dingo.NewInjector(a, b)
				require.NoError(t, err)

				return injector, map[string]dingo.Module{
					"Bind dingo.Injector[0]": nil,
					`BindMap string["a"]`:    a,
					`BindMap string["b"]`:    b,
					`BindMap dingo_test.inspectIface annotated with "b"["b"]`: b,
				}
			},
		},
		{
			name: "bindings of ModuleFuncs report their function",
			setup: func(t *testing.T) (*dingo.Injector, map[string]dingo.Module) {
				t.Helper()

				var f dingo.ModuleFunc = func(injector *dingo.Injector) {
					injector.Bind(new(string)).ToInstance("f")
					injector.BindMulti(new(string)).ToInstance("f")
					injector.BindMap(new(string), "f").ToInstance("f")
				}

				var g dingo.ModuleFunc = func(injector *dingo.Injector) {
					injector.BindMulti(new(string)).ToInstance("g")
					injector.BindMap(new(string), "g").ToInstance("g")
				}

				injector, err := dingo.NewInjector(f, g)
				require.NoError(t, err)

				return injector, map[string]dingo.Module{
					"Bind dingo.Injector[0]": nil,
					"Bind string[0]":         f,
					"BindMulti string[0]":    f,
					"BindMulti string[1]":    g,
					`BindMap string["f"]`:    f,
					`BindMap string["g"]`:    g,
				}
			},
		},
		{
			name: "an override reports the overriding module in every slot it replaces",
			setup: func(t *testing.T) (*dingo.Injector, map[string]dingo.Module) {
				t.Helper()

				a := &inspectModuleA{configure: func(injector *dingo.Injector) {
					injector.Bind(new(string)).ToInstance("a")
				}}
				b := &inspectModuleB{configure: func(injector *dingo.Injector) {
					injector.Override(new(string), "").ToInstance("b")
				}}

				injector, err := dingo.NewInjector(a, b)
				require.NoError(t, err)

				return injector, map[string]dingo.Module{
					"Bind dingo.Injector[0]": nil,
					"Bind string[0]":         b,
					"Bind string[1]":         b,
				}
			},
		},
		{
			name: "identical bindings from two modules are still allowed",
			setup: func(t *testing.T) (*dingo.Injector, map[string]dingo.Module) {
				t.Helper()

				a := &inspectModuleA{configure: func(injector *dingo.Injector) {
					injector.Bind(new(inspectIface)).To(inspectImpl{})
				}}
				b := &inspectModuleB{configure: func(injector *dingo.Injector) {
					injector.Bind(new(inspectIface)).To(inspectImpl{})
				}}

				injector, err := dingo.NewInjector(a, b)
				require.NoError(t, err)

				return injector, map[string]dingo.Module{
					"Bind dingo.Injector[0]":          nil,
					"Bind dingo_test.inspectIface[0]": a,
					"Bind dingo_test.inspectIface[1]": b,
				}
			},
		},
		{
			name: "bindings made while a module is injected report that module",
			setup: func(t *testing.T) (*dingo.Injector, map[string]dingo.Module) {
				t.Helper()

				m := &inspectInjectedModule{
					inject: func(injector *dingo.Injector) {
						injector.Bind(new(int)).ToInstance(1)
					},
					configure: func(injector *dingo.Injector) {
						injector.Bind(new(string)).ToInstance("configured")
					},
				}

				injector, err := dingo.NewInjector(m)
				require.NoError(t, err)

				return injector, map[string]dingo.Module{
					"Bind dingo.Injector[0]": nil,
					"Bind int[0]":            m,
					"Bind string[0]":         m,
				}
			},
		},
		{
			name: "a nested InitModules restores the outer module",
			setup: func(t *testing.T) (*dingo.Injector, map[string]dingo.Module) {
				t.Helper()

				inner := &inspectModuleB{configure: func(injector *dingo.Injector) {
					injector.Bind(new(int)).ToInstance(1)
				}}
				outer := &inspectModuleA{configure: func(injector *dingo.Injector) {
					if err := injector.InitModules(inner); err != nil {
						panic(err)
					}

					injector.Bind(new(string)).ToInstance("outer")
				}}

				injector, err := dingo.NewInjector(outer)
				require.NoError(t, err)

				return injector, map[string]dingo.Module{
					"Bind dingo.Injector[0]": nil,
					"Bind int[0]":            inner,
					"Bind string[0]":         outer,
				}
			},
		},
		{
			name: "a child reports its own modules and its parent keeps its modules",
			setup: func(t *testing.T) (*dingo.Injector, map[string]dingo.Module) {
				t.Helper()

				a := &inspectModuleA{configure: func(injector *dingo.Injector) {
					injector.Bind(new(string)).ToInstance("parent")
					injector.BindMulti(new(string)).ToInstance("parent")
				}}
				b := &inspectModuleB{configure: func(injector *dingo.Injector) {
					injector.Bind(new(string)).ToInstance("child")
					injector.BindMulti(new(string)).ToInstance("child")
					injector.BindMap(new(string), "child").ToInstance("child")
				}}

				parent, err := dingo.NewInjector(a)
				require.NoError(t, err)

				child, err := parent.Child()
				require.NoError(t, err)
				require.NoError(t, child.InitModules(b))

				return child, map[string]dingo.Module{
					"Bind dingo.Injector[0]":        nil,
					"Bind dingo.Injector[1]":        nil,
					"Bind string[0]":                b,
					"BindMulti string[0]":           b,
					`BindMap string["child"]`:       b,
					"parent Bind dingo.Injector[0]": nil,
					"parent Bind string[0]":         a,
					"parent BindMulti string[0]":    a,
				}
			},
		},
		{
			name: "a child created during a module's configuration reports none",
			setup: func(t *testing.T) (*dingo.Injector, map[string]dingo.Module) {
				t.Helper()

				var child *dingo.Injector

				a := &inspectModuleA{configure: func(injector *dingo.Injector) {
					var err error
					if child, err = injector.Child(); err != nil {
						panic(err)
					}

					injector.Bind(new(string)).ToInstance("parent")
				}}

				_, err := dingo.NewInjector(a)
				require.NoError(t, err)

				return child, map[string]dingo.Module{
					"Bind dingo.Injector[0]":        nil,
					"Bind dingo.Injector[1]":        nil,
					"parent Bind dingo.Injector[0]": nil,
					"parent Bind string[0]":         a,
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

func TestInspect_NoDeclaringModuleOutsideModuleInitialization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// setup returns an injector that bound something outside of module
		// initialization, and the declaring module expected for every
		// binding it reports.
		setup func(t *testing.T) (*dingo.Injector, map[string]dingo.Module)
	}{
		{
			name: "bound directly on the injector",
			setup: func(t *testing.T) (*dingo.Injector, map[string]dingo.Module) {
				t.Helper()

				a := &inspectModuleA{configure: func(injector *dingo.Injector) {
					injector.Bind(new(string)).ToInstance("a")
				}}

				injector, err := dingo.NewInjector(a)
				require.NoError(t, err)

				injector.Bind(new(int)).ToInstance(1)
				injector.BindMulti(new(int)).ToInstance(1)
				injector.BindMap(new(int), "direct").ToInstance(1)

				return injector, map[string]dingo.Module{
					"Bind dingo.Injector[0]": nil,
					"Bind string[0]":         a,
					"Bind int[0]":            nil,
					"BindMulti int[0]":       nil,
					`BindMap int["direct"]`:  nil,
				}
			},
		},
		{
			name: "resolved without a binding",
			setup: func(t *testing.T) (*dingo.Injector, map[string]dingo.Module) {
				t.Helper()

				injector, err := dingo.NewInjector()
				require.NoError(t, err)

				// Resolving an unbound type creates no binding to report.
				_, err = injector.GetInstance(new(inspectImpl))
				require.NoError(t, err)

				return injector, map[string]dingo.Module{
					"Bind dingo.Injector[0]": nil,
				}
			},
		},
		{
			name: "bound by a delayed injection",
			setup: func(t *testing.T) (*dingo.Injector, map[string]dingo.Module) {
				t.Helper()

				a := &inspectModuleA{configure: func(injector *dingo.Injector) {
					injector.Bind(new(int)).ToInstance(1)

					delayed := &inspectDelayed{inject: func(injector *dingo.Injector) {
						injector.Bind(new(string)).ToInstance("delayed")
					}}
					if err := injector.RequestInjection(delayed); err != nil {
						panic(err)
					}
				}}

				injector, err := dingo.NewInjector(a)
				require.NoError(t, err)

				return injector, map[string]dingo.Module{
					"Bind dingo.Injector[0]": nil,
					"Bind int[0]":            a,
					"Bind string[0]":         nil,
				}
			},
		},
		{
			name: "bound by a delayed injection that a nested InitModules runs",
			setup: func(t *testing.T) (*dingo.Injector, map[string]dingo.Module) {
				t.Helper()

				a := &inspectModuleA{configure: func(injector *dingo.Injector) {
					delayed := &inspectDelayed{inject: func(injector *dingo.Injector) {
						injector.Bind(new(string)).ToInstance("delayed")
					}}
					if err := injector.RequestInjection(delayed); err != nil {
						panic(err)
					}
				}}
				b := &inspectModuleB{configure: func(injector *dingo.Injector) {
					if err := injector.InitModules(dingo.ModuleFunc(func(*dingo.Injector) {})); err != nil {
						panic(err)
					}

					injector.Bind(new(int)).ToInstance(1)
				}}

				injector, err := dingo.NewInjector(a, b)
				require.NoError(t, err)

				return injector, map[string]dingo.Module{
					"Bind dingo.Injector[0]": nil,
					"Bind int[0]":            b,
					"Bind string[0]":         nil,
				}
			},
		},
		{
			name: "bound by the provider of an eager singleton",
			setup: func(t *testing.T) (*dingo.Injector, map[string]dingo.Module) {
				t.Helper()

				a := &inspectModuleA{configure: func(injector *dingo.Injector) {
					injector.Bind(new(int)).ToProvider(func(injector *dingo.Injector) int {
						injector.Bind(new(string)).ToInstance("eager")

						return 1
					}).AsEagerSingleton()
				}}

				injector, err := dingo.NewInjector()
				require.NoError(t, err)

				// The global Singleton scope would cache the instance across
				// test runs and skip the provider, so use a scope of our own.
				injector.BindScope(dingo.NewSingletonScope())
				require.NoError(t, injector.InitModules(a))

				return injector, map[string]dingo.Module{
					"Bind dingo.Injector[0]": nil,
					"Bind int[0]":            a,
					"Bind string[0]":         nil,
				}
			},
		},
		{
			name: "bound after a module failed injection",
			setup: func(t *testing.T) (*dingo.Injector, map[string]dingo.Module) {
				t.Helper()

				injector, err := dingo.NewInjector()
				require.NoError(t, err)
				require.Error(t, injector.InitModules(new(inspectUnresolvableModule)))

				injector.Bind(new(string)).ToInstance("after error")

				return injector, map[string]dingo.Module{
					"Bind dingo.Injector[0]": nil,
					"Bind string[0]":         nil,
				}
			},
		},
		{
			name: "bound after a module's configuration panicked",
			setup: func(t *testing.T) (*dingo.Injector, map[string]dingo.Module) {
				t.Helper()

				injector, err := dingo.NewInjector()
				require.NoError(t, err)

				a := &inspectModuleA{configure: func(injector *dingo.Injector) {
					injector.Bind(nil)
				}}

				require.Panics(t, func() { _ = injector.InitModules(a) })

				injector.Bind(new(string)).ToInstance("after panic")

				return injector, map[string]dingo.Module{
					"Bind dingo.Injector[0]": nil,
					"Bind string[0]":         nil,
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

func TestInitModules_DuplicateBindingErrorNamesTheDeclaringModules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		init func(t *testing.T) error
		want string
	}{
		{
			name: "both bindings declared by modules",
			init: func(t *testing.T) error {
				t.Helper()

				a := &inspectModuleA{configure: func(injector *dingo.Injector) {
					injector.Bind(new(inspectIface)).To(inspectImpl{})
				}}
				b := &inspectModuleB{configure: func(injector *dingo.Injector) {
					injector.Bind(new(inspectIface)).To(inspectOtherImpl{})
				}}

				_, err := dingo.NewInjector(a, b)

				return err
			},
			want: `already known binding for "dingo_test.inspectIface" with annotation "" | ` +
				`Known binding: "\"flamingo.me/dingo_test\"\"inspectImpl\"" Try "\"flamingo.me/dingo_test\"\"inspectOtherImpl\"" | ` +
				`Known binding declared by *flamingo.me/dingo_test.inspectModuleA, duplicate declared by *flamingo.me/dingo_test.inspectModuleB`,
		},
		{
			name: "a binding without a declaring module is named unknown",
			init: func(t *testing.T) error {
				t.Helper()

				injector, err := dingo.NewInjector()
				require.NoError(t, err)

				injector.Bind(new(string)).ToInstance("direct")

				b := &inspectModuleB{configure: func(injector *dingo.Injector) {
					injector.Bind(new(string)).ToInstance("b")
				}}

				return injector.InitModules(b)
			},
			want: `already known binding for "string" with annotation "" | Known binding: "" Try "" | ` +
				`Known binding declared by unknown, duplicate declared by *flamingo.me/dingo_test.inspectModuleB`,
		},
		{
			name: "ModuleFunc modules are named by their function",
			init: func(t *testing.T) error {
				t.Helper()

				_, err := dingo.NewInjector(dingo.ModuleFunc(inspectFuncModuleA), dingo.ModuleFunc(inspectFuncModuleB))

				return err
			},
			want: `already known binding for "string" with annotation "" | Known binding: "" Try "" | ` +
				`Known binding declared by flamingo.me/dingo_test.inspectFuncModuleA, duplicate declared by flamingo.me/dingo_test.inspectFuncModuleB`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assert.EqualError(t, test.init(t), test.want)
		})
	}
}

// inspectDeclaringModules returns the declaring module Inspect reports for
// every binding, keyed by the bind method, the bound type, the annotation and
// the position. Single bindings of one type and annotation are numbered in
// binding order.
// Keys of bindings reported by a parent injector start with "parent ".
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
		InspectMapBinding: func(of reflect.Type, key string, annotation string, _ reflect.Type, _, _ *reflect.Value, _ dingo.Scope, declaredBy dingo.Module) {
			got[fmt.Sprintf("%s[%q]", inspectKey("BindMap", of, annotation), key)] = declaredBy
		},
		InspectParent: func(parent *dingo.Injector) {
			for binding, module := range inspectDeclaringModules(parent) {
				got["parent "+binding] = module
			}
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
