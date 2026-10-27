package dingo

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The declaring module's behaviour is tested through Inspect in
// inspect_test.go. These tests cover what Inspect cannot show.

type (
	declaringIface     interface{ declared() }
	declaringImpl      struct{}
	declaringOtherImpl struct{}

	// Each module type carries its configuration as a field, so pointers to
	// two instances never compare equal.
	declaringModuleA struct{ configure func(*Injector) }
	declaringModuleB struct{ configure func(*Injector) }
)

func (*declaringImpl) declared()      {}
func (*declaringOtherImpl) declared() {}

func (m *declaringModuleA) Configure(injector *Injector) { m.configure(injector) }
func (m *declaringModuleB) Configure(injector *Injector) { m.configure(injector) }

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
			name:  "provider and no provider",
			left:  bind(new(string), a).ToProvider(provider),
			right: bind(new(string), b),
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
