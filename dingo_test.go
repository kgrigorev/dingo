package dingo

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"flamingo.me/dingo/internal/hooks"
)

type (
	testInterface interface {
		Test() int
	}

	interfaceSub testInterface

	interfaceImpl1 struct {
		foo string
	}

	interfaceImpl2 struct{}

	testInterfaceProvider          func() testInterface
	testInterfaceWithErrorProvider func() (testInterface, error)

	depTest struct {
		Iface  testInterface `inject:""`
		Iface2 testInterface `inject:"test"`

		IfaceProvider          testInterfaceProvider          `inject:""`
		IfaceWithErrorProvider testInterfaceWithErrorProvider `inject:""`
		IfaceProvided          testInterface                  `inject:"provider"`
		IfaceImpl1Provided     testInterface                  `inject:"providerimpl1"`
		IfaceInstance          testInterface                  `inject:"instance"`
	}

	testSingleton struct{}

	testModule struct{}

	preTestModule struct{}
)

func interfaceProvider(str string) testInterface {
	return &interfaceImpl1{foo: str}
}

func interfaceImpl1Provider(str string) *interfaceImpl1 {
	return &interfaceImpl1{foo: str}
}

func (ptm *preTestModule) Configure(injector *Injector) {
	injector.Bind((*string)(nil)).ToInstance("Hello World")
}

func (tm *testModule) Configure(injector *Injector) {
	injector.Bind((*testInterface)(nil)).To((*interfaceSub)(nil))
	injector.Bind((*interfaceSub)(nil)).To(interfaceImpl1{})
	injector.Bind((*testInterface)(nil)).AnnotatedWith("test").To(interfaceImpl2{})

	injector.Bind((*testInterface)(nil)).AnnotatedWith("provider").ToProvider(interfaceProvider)
	injector.Bind((*testInterface)(nil)).AnnotatedWith("providerimpl1").ToProvider(interfaceImpl1Provider)
	injector.Bind((*testInterface)(nil)).AnnotatedWith("instance").ToInstance(new(interfaceImpl2))

	injector.Bind(testSingleton{}).AsEagerSingleton()
}

func (if1 *interfaceImpl1) Test() int {
	return 1
}

func (if2 *interfaceImpl2) Test() int {
	return 2
}

func TestDingoResolving(t *testing.T) {
	t.Run("Should resolve dependencies on request", func(t *testing.T) {
		injector, err := NewInjector(new(preTestModule), new(testModule))
		assert.NoError(t, err)

		i, err := injector.GetInstance(new(testInterface))
		assert.NoError(t, err)
		var iface testInterface
		iface = i.(testInterface)

		assert.Equal(t, 1, iface.Test())

		i, err = injector.GetInstance(new(depTest))
		assert.NoError(t, err)
		dt := *i.(*depTest)

		assert.Equal(t, 1, dt.Iface.Test())
		assert.Equal(t, 2, dt.Iface2.Test())

		var dt2 depTest
		assert.NoError(t, injector.requestInjection(&dt2, nil))

		assert.Equal(t, 1, dt2.Iface.Test())
		assert.Equal(t, 2, dt2.Iface2.Test())

		assert.Equal(t, 1, dt.IfaceProvided.Test())
		assert.Equal(t, 1, dt.IfaceImpl1Provided.Test())
		assert.Equal(t, 2, dt.IfaceInstance.Test())

		assert.Equal(t, 1, dt.IfaceProvider().Test())
		iface, err = dt.IfaceWithErrorProvider()
		assert.NoError(t, err)
		assert.Equal(t, 1, iface.Test())
		assert.Equal(t, "Hello World", dt.IfaceProvided.(*interfaceImpl1).foo)
		assert.Equal(t, "Hello World", dt.IfaceImpl1Provided.(*interfaceImpl1).foo)
	})

	t.Run("Should resolve scopes", func(t *testing.T) {
		injector, err := NewInjector(new(testModule))
		assert.NoError(t, err)

		i1, err := injector.GetInstance(testSingleton{})
		assert.NoError(t, err)
		i2, err := injector.GetInstance(testSingleton{})
		assert.NoError(t, err)
		assert.Equal(t, i1, i2)
	})

	t.Run("Error cases", func(t *testing.T) {
		var injector *Injector
		_, err := injector.Child()
		assert.Error(t, err)
	})
}

type testBoundNothingProvider func() *interfaceImpl1

func TestBoundToNothing(t *testing.T) {
	injector, err := NewInjector()
	assert.NoError(t, err)

	injector.Bind(new(interfaceImpl1)).AnnotatedWith("test")

	i, err := injector.GetInstance(new(testBoundNothingProvider))
	assert.NoError(t, err)
	ii, ok := i.(testBoundNothingProvider)
	assert.True(t, ok)
	assert.NotNil(t, ii)
	assert.NotNil(t, ii())
}

// interceptors
type (
	AopInterface interface {
		Test() string
	}

	AopImpl struct{}

	AopDep struct {
		A AopInterface `inject:""`
	}

	AopInterceptor1 struct {
		AopInterface
	}

	AopInterceptor2 struct {
		AopInterface
	}

	AopModule struct{}
)

func (m *AopModule) Configure(injector *Injector) {
	injector.Bind((*AopInterface)(nil)).To(AopImpl{})

	injector.BindInterceptor((*AopInterface)(nil), AopInterceptor1{})
	injector.BindInterceptor((*AopInterface)(nil), AopInterceptor2{})
}

func (a *AopImpl) Test() string {
	return "Test"
}

func (a *AopInterceptor1) Test() string {
	return a.AopInterface.Test() + " 1"
}

func (a *AopInterceptor2) Test() string {
	return a.AopInterface.Test() + " 2"
}

func TestInterceptors(t *testing.T) {
	injector, err := NewInjector(new(AopModule))
	assert.NoError(t, err)

	var dep AopDep
	assert.NoError(t, injector.requestInjection(&dep, nil))

	assert.Equal(t, "Test 1 2", dep.A.Test())
}

func TestOptional(t *testing.T) {
	type test struct {
		Must      string `inject:"must"`
		Optional  string `inject:"option,optional"`
		Optional2 string `inject:"option, optional"`
	}

	injector, err := NewInjector()
	assert.NoError(t, err)

	_, err = injector.GetInstance(new(test))
	assert.Error(t, err)

	injector.Bind(new(string)).AnnotatedWith("must").ToInstance("must")
	i, err := injector.GetInstance(new(test))
	assert.NoError(t, err)
	assert.Equal(t, i.(*test).Must, "must")
	assert.Equal(t, i.(*test).Optional, "")
	assert.Equal(t, i.(*test).Optional2, "")

	injector.Bind(new(string)).AnnotatedWith("option").ToInstance("option")
	i, err = injector.GetInstance(new(test))
	assert.NoError(t, err)
	assert.Equal(t, i.(*test).Must, "must")
	assert.Equal(t, i.(*test).Optional, "option")
	assert.Equal(t, i.(*test).Optional2, "option")
}

func TestOverrides(t *testing.T) {
	t.Run("not annotated", func(t *testing.T) {
		injector, err := NewInjector()
		assert.NoError(t, err)

		injector.Bind(new(string)).ToInstance("test")
		injector.Bind(new(string)).ToInstance("test-bla")
		injector.Override(new(string), "").ToInstance("test2")
		assert.NoError(t, injector.InitModules())

		i, err := injector.GetInstance(new(string))
		assert.NoError(t, err)

		s, ok := i.(string)
		assert.True(t, ok)
		assert.Equal(t, "test2", s)
	})

	t.Run("annotated", func(t *testing.T) {
		injector, err := NewInjector()
		assert.NoError(t, err)

		injector.Bind(new(string)).AnnotatedWith("test").ToInstance("test")
		injector.Bind(new(string)).AnnotatedWith("test").ToInstance("test-bla")
		injector.Override(new(string), "test").ToInstance("test2")
		assert.NoError(t, injector.InitModules())

		i, err := injector.GetAnnotatedInstance(new(string), "test")
		assert.NoError(t, err)

		s, ok := i.(string)
		assert.True(t, ok)
		assert.Equal(t, "test2", s)
	})
}

func TestProvider(t *testing.T) {
	t.Run("Provider", func(t *testing.T) {
		injector, err := NewInjector()
		assert.NoError(t, err)

		injector.Bind(new(string)).ToProvider(func(i int) string {
			return "test" + strconv.Itoa(i)
		})

		i, err := injector.GetInstance(new(string))
		assert.NoError(t, err)

		assert.Equal(t, "test0", i.(string))
	})

	t.Run("Slice Provider", func(t *testing.T) {
		injector, err := NewInjector()
		assert.NoError(t, err)

		injector.Bind(new([]string)).ToProvider(func() []string {
			return []string{"a", "b"}
		})

		i, err := injector.GetInstance(new([]string))
		assert.NoError(t, err)

		assert.Equal(t, []string{"a", "b"}, i.([]string))
	})

	t.Run("Invalid Provider", func(t *testing.T) {
		injector, err := NewInjector()
		assert.NoError(t, err)

		injector.Bind(new(string)).ToProvider(func(interface{}) string {
			return "test"
		})

		_, err = injector.GetInstance(new(string))
		assert.Error(t, err)
	})
}

type testInjectInvalid struct {
	A int `inject:"a"`
}

func (*testInjectInvalid) Configure(*Injector) {}

func TestInjector_InitModules(t *testing.T) {
	injector, err := NewInjector()
	assert.NoError(t, err)
	assert.Error(t, injector.InitModules(new(testInjectInvalid)))
}

type TestInjectStructRecInterface interface {
	TestXyz()
}

type testInjectStructRecStruct struct{}

func (t testInjectStructRecStruct) TestXyz() {}

func (t testInjectStructRecStruct) Inject() {}

func TestInjectStructRec(t *testing.T) {
	t.Parallel()

	injector, err := NewInjector()
	assert.NoError(t, err)

	injector.Bind(new(TestInjectStructRecInterface)).To(new(testInjectStructRecStruct))

	_, err = injector.GetInstance(new(TestInjectStructRecInterface))
	assert.Error(t, err)
}

type someStructWithInvalidInterfacePointer struct {
	A *testInterface `inject:""`
}

// TestInjectionOfInterfacePointer pins the pointer-to-interface rejection, the exported
// ErrPointerToInterface sentinel (matched via errors.Is by the typed API), and the message carve-out.
func TestInjectionOfInterfacePointer(t *testing.T) {
	t.Parallel()

	injector, err := NewInjector()
	require.NoError(t, err)

	injector.Bind((*testInterface)(nil)).To(interfaceImpl1{})

	_, err = injector.GetInstance(new(someStructWithInvalidInterfacePointer))
	require.ErrorIs(t, err, ErrPointerToInterface)
	assert.ErrorContains(t, err, "pointer to interface is not allowed")
}

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
		injector, ok := root.(*Injector)
		require.True(t, ok)

		created = append(created, injector)
		attached := &fakeAttached{root: injector}
		injector.Bind(fakeAttached{}).ToInstance(attached)

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
