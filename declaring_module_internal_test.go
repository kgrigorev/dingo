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

	// declaringInjectedModule binds from its Inject method as well.
	declaringInjectedModule struct {
		inject, configure func(*Injector)
	}

	// declaringUnresolvableModule fails injection, so it is never configured.
	declaringUnresolvableModule struct {
		Missing declaringIface `inject:""`
	}

	// declaringDelayed binds while its delayed injection runs.
	declaringDelayed struct{ inject func(*Injector) }
)

func (*declaringImpl) declared()      {}
func (*declaringOtherImpl) declared() {}

func (m *declaringModuleA) Configure(injector *Injector) { m.configure(injector) }
func (m *declaringModuleB) Configure(injector *Injector) { m.configure(injector) }

func (m *declaringInjectedModule) Inject(injector *Injector)    { m.inject(injector) }
func (m *declaringInjectedModule) Configure(injector *Injector) { m.configure(injector) }

func (*declaringUnresolvableModule) Configure(*Injector) {}

func (d *declaringDelayed) Inject(injector *Injector) { d.inject(injector) }

// declaringFuncModuleA and declaringFuncModuleB are used as ModuleFunc modules.
func declaringFuncModuleA(injector *Injector) { injector.Bind(new(string)).ToInstance("a") }
func declaringFuncModuleB(injector *Injector) { injector.Bind(new(string)).ToInstance("b") }

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
					"Bind dingo.Injector[0]":       nil,
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
					"Bind dingo.Injector[0]":            nil,
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
		{
			name: "bindings made while a module is injected record that module",
			setup: func() ([]Module, map[string]Module) {
				m := &declaringInjectedModule{
					inject: func(injector *Injector) {
						injector.Bind(new(int)).ToInstance(1)
					},
					configure: func(injector *Injector) {
						injector.Bind(new(string)).ToInstance("configured")
					},
				}

				return []Module{m}, map[string]Module{
					"Bind dingo.Injector[0]": nil,
					"Bind int[0]":            m,
					"Bind string[0]":         m,
				}
			},
		},
		{
			name: "a nested InitModules restores the outer module",
			setup: func() ([]Module, map[string]Module) {
				inner := &declaringModuleB{configure: func(injector *Injector) {
					injector.Bind(new(int)).ToInstance(1)
				}}
				outer := &declaringModuleA{configure: func(injector *Injector) {
					if err := injector.InitModules(inner); err != nil {
						panic(err)
					}

					injector.Bind(new(string)).ToInstance("outer")
				}}

				return []Module{outer}, map[string]Module{
					"Bind dingo.Injector[0]": nil,
					"Bind int[0]":            inner,
					"Bind string[0]":         outer,
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

func TestChild_DeclaringModule(t *testing.T) {
	t.Parallel()

	t.Run("a child records its own modules and the parent keeps its own", func(t *testing.T) {
		t.Parallel()

		a := &declaringModuleA{configure: func(injector *Injector) {
			injector.Bind(new(string)).ToInstance("parent")
		}}
		b := &declaringModuleB{configure: func(injector *Injector) {
			injector.Bind(new(string)).ToInstance("child")
			injector.BindMulti(new(string)).ToInstance("child")
			injector.BindMap(new(string), "child").ToInstance("child")
		}}

		parent, err := NewInjector(a)
		require.NoError(t, err)

		child, err := parent.Child()
		require.NoError(t, err)
		require.NoError(t, child.InitModules(b))

		assertDeclaringModules(t, map[string]Module{
			"Bind dingo.Injector[0]": nil,
			"Bind string[0]":         a,
		}, declaringModulesOf(parent))
		assertDeclaringModules(t, map[string]Module{
			"Bind dingo.Injector[0]":  nil,
			"Bind dingo.Injector[1]":  nil,
			"Bind string[0]":          b,
			"BindMulti string[0]":     b,
			`BindMap string["child"]`: b,
		}, declaringModulesOf(child))
	})

	t.Run("a child created during a module's configuration records none", func(t *testing.T) {
		t.Parallel()

		var child *Injector

		a := &declaringModuleA{configure: func(injector *Injector) {
			var err error
			if child, err = injector.Child(); err != nil {
				panic(err)
			}

			injector.Bind(new(string)).ToInstance("parent")
		}}

		parent, err := NewInjector(a)
		require.NoError(t, err)

		assertDeclaringModules(t, map[string]Module{
			"Bind dingo.Injector[0]": nil,
			"Bind string[0]":         a,
		}, declaringModulesOf(parent))
		assertDeclaringModules(t, map[string]Module{
			"Bind dingo.Injector[0]": nil,
			"Bind dingo.Injector[1]": nil,
		}, declaringModulesOf(child))
	})
}

func TestInitModules_BindingsOutsideModuleConfigurationHaveNoDeclaringModule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// setup returns an injector that bound a string outside of any
		// module's configuration, and the expected declaring modules.
		setup func(t *testing.T) (*Injector, map[string]Module)
	}{
		{
			name: "bound directly on the injector",
			setup: func(t *testing.T) (*Injector, map[string]Module) {
				t.Helper()

				injector, err := NewInjector()
				require.NoError(t, err)

				injector.Bind(new(string)).ToInstance("direct")

				return injector, map[string]Module{
					"Bind dingo.Injector[0]": nil,
					"Bind string[0]":         nil,
				}
			},
		},
		{
			name: "bound by a delayed injection",
			setup: func(t *testing.T) (*Injector, map[string]Module) {
				t.Helper()

				a := &declaringModuleA{configure: func(injector *Injector) {
					injector.Bind(new(int)).ToInstance(1)

					delayed := &declaringDelayed{inject: func(injector *Injector) {
						injector.Bind(new(string)).ToInstance("delayed")
					}}
					if err := injector.RequestInjection(delayed); err != nil {
						panic(err)
					}
				}}

				injector, err := NewInjector(a)
				require.NoError(t, err)

				return injector, map[string]Module{
					"Bind dingo.Injector[0]": nil,
					"Bind int[0]":            a,
					"Bind string[0]":         nil,
				}
			},
		},
		{
			name: "bound by a delayed injection that a nested InitModules runs",
			setup: func(t *testing.T) (*Injector, map[string]Module) {
				t.Helper()

				a := &declaringModuleA{configure: func(injector *Injector) {
					delayed := &declaringDelayed{inject: func(injector *Injector) {
						injector.Bind(new(string)).ToInstance("delayed")
					}}
					if err := injector.RequestInjection(delayed); err != nil {
						panic(err)
					}
				}}
				b := &declaringModuleB{configure: func(injector *Injector) {
					if err := injector.InitModules(ModuleFunc(func(*Injector) {})); err != nil {
						panic(err)
					}

					injector.Bind(new(int)).ToInstance(1)
				}}

				injector, err := NewInjector(a, b)
				require.NoError(t, err)

				return injector, map[string]Module{
					"Bind dingo.Injector[0]": nil,
					"Bind int[0]":            b,
					"Bind string[0]":         nil,
				}
			},
		},
		{
			name: "bound by the provider of an eager singleton",
			setup: func(t *testing.T) (*Injector, map[string]Module) {
				t.Helper()

				a := &declaringModuleA{configure: func(injector *Injector) {
					injector.Bind(new(int)).ToProvider(func(injector *Injector) int {
						injector.Bind(new(string)).ToInstance("eager")

						return 1
					}).AsEagerSingleton()
				}}

				injector, err := NewInjector()
				require.NoError(t, err)

				// The global Singleton scope would cache the instance across
				// test runs and skip the provider, so use a scope of our own.
				injector.BindScope(NewSingletonScope())
				require.NoError(t, injector.InitModules(a))

				return injector, map[string]Module{
					"Bind dingo.Injector[0]": nil,
					"Bind int[0]":            a,
					"Bind string[0]":         nil,
				}
			},
		},
		{
			name: "bound after a module failed injection",
			setup: func(t *testing.T) (*Injector, map[string]Module) {
				t.Helper()

				injector, err := NewInjector()
				require.NoError(t, err)
				require.Error(t, injector.InitModules(new(declaringUnresolvableModule)))

				injector.Bind(new(string)).ToInstance("after error")

				return injector, map[string]Module{
					"Bind dingo.Injector[0]": nil,
					"Bind string[0]":         nil,
				}
			},
		},
		{
			name: "bound after a module's configuration panicked",
			setup: func(t *testing.T) (*Injector, map[string]Module) {
				t.Helper()

				injector, err := NewInjector()
				require.NoError(t, err)

				a := &declaringModuleA{configure: func(injector *Injector) {
					injector.Bind(nil)
				}}

				require.Panics(t, func() { _ = injector.InitModules(a) })

				injector.Bind(new(string)).ToInstance("after panic")

				return injector, map[string]Module{
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

			assertDeclaringModules(t, want, declaringModulesOf(injector))
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

				a := &declaringModuleA{configure: func(injector *Injector) {
					injector.Bind(new(declaringIface)).To(declaringImpl{})
				}}
				b := &declaringModuleB{configure: func(injector *Injector) {
					injector.Bind(new(declaringIface)).To(declaringOtherImpl{})
				}}

				_, err := NewInjector(a, b)

				return err
			},
			want: `already known binding for "dingo.declaringIface" with annotation "" | ` +
				`Known binding: "\"flamingo.me/dingo\"\"declaringImpl\"" Try "\"flamingo.me/dingo\"\"declaringOtherImpl\"" | ` +
				`Known binding declared by *flamingo.me/dingo.declaringModuleA, duplicate declared by *flamingo.me/dingo.declaringModuleB`,
		},
		{
			name: "a binding without a declaring module is named unknown",
			init: func(t *testing.T) error {
				t.Helper()

				injector, err := NewInjector()
				require.NoError(t, err)

				injector.Bind(new(string)).ToInstance("direct")

				b := &declaringModuleB{configure: func(injector *Injector) {
					injector.Bind(new(string)).ToInstance("b")
				}}

				return injector.InitModules(b)
			},
			want: `already known binding for "string" with annotation "" | Known binding: "" Try "" | ` +
				`Known binding declared by unknown, duplicate declared by *flamingo.me/dingo.declaringModuleB`,
		},
		{
			name: "ModuleFunc modules are named by their function",
			init: func(t *testing.T) error {
				t.Helper()

				_, err := NewInjector(ModuleFunc(declaringFuncModuleA), ModuleFunc(declaringFuncModuleB))

				return err
			},
			want: `already known binding for "string" with annotation "" | Known binding: "" Try "" | ` +
				`Known binding declared by flamingo.me/dingo.declaringFuncModuleA, duplicate declared by flamingo.me/dingo.declaringFuncModuleB`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assert.EqualError(t, test.init(t), test.want)
		})
	}
}

func TestInitModules_DuplicateBindingErrorWrapsErrDuplicateBinding(t *testing.T) {
	t.Parallel()

	injector, err := NewInjector()
	require.NoError(t, err)

	injector.Bind(new(string)).ToInstance("direct")

	err = injector.InitModules(ModuleFunc(func(injector *Injector) {
		injector.Bind(new(string)).ToInstance("module")
	}))

	assert.ErrorIs(t, err, errDuplicateBinding)
}

func TestErrUnbound_DoesNotPrintTheDeclaringModulePointer(t *testing.T) {
	t.Parallel()

	a := &declaringModuleA{configure: func(injector *Injector) {
		injector.Bind(new(string))
	}}

	injector, err := NewInjector(a)
	require.NoError(t, err)

	binding := injector.bindings[reflect.TypeFor[string]()][0]
	require.Same(t, a, binding.declaredBy)

	message := errUnbound{binding: binding, typ: binding.typeof}.Error()

	assert.Contains(t, message, "binding is not bound: ")
	assert.NotContains(t, message, fmt.Sprintf("%p", a))
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
