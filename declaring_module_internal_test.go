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

	module := ModuleFunc(func(*Injector) {})
	binding := &Binding{typeof: reflect.TypeFor[string](), declaredBy: module}

	message := errUnbound{binding: binding, typ: binding.typeof}.Error()

	assert.Contains(t, message, "binding is not bound: ")
	assert.NotContains(t, message, fmt.Sprintf("%p", module))
}

func TestBinding_equalIgnoresTheDeclaringModule(t *testing.T) {
	t.Parallel()

	// Non-nil funcs are never deeply equal, so comparing the modules would
	// make every equal case fail.
	a := ModuleFunc(func(*Injector) {})
	b := ModuleFunc(func(*Injector) {})
	instance := new(int)
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
			left:  bind(new(any), a).To(""),
			right: bind(new(any), b).To(""),
			equal: true,
		},
		{
			name:  "same instance",
			left:  bind(new(*int), a).ToInstance(instance),
			right: bind(new(*int), b).ToInstance(instance),
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
			left:  bind(new(any), a).To(""),
			right: bind(new(any), b).To(0),
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
			name:  "no provider and provider",
			left:  bind(new(string), a),
			right: bind(new(string), b).ToProvider(provider),
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
