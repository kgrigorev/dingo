package v2

import (
	"errors"
	"reflect"

	"flamingo.me/dingo"
)

var (
	ErrInvalidInjectReceiver = errors.New("usage of 'Inject' method with struct receiver is not allowed")
)

//const (
//	Singleton      Scope = "singleton"
//	ChildSingleton Scope = "childSingleton"
//)

type (
	Injector struct {
	}

	Binding struct{}

	Instance struct {
	}

	Provider struct {
	}

	unscopedFunc func(t reflect.Type, annotation string, optional bool) (reflect.Value, error)

	//Scope string
	Scope interface {
		ResolveType(t reflect.Type, annotation string, unscoped unscopedFunc) (reflect.Value, error)
	}

	// Module is the default entry point for dingo Modules.
	// The Configure method is called once during initialization
	// and lets the module set up Bindings for the provided Injector.
	Module interface {
		Configure(injector *Injector)
	}

	// ModuleFunc wraps a func(injector *Injector) for dependency injection.
	// This allows using small functions as dingo Modules.
	// The same concept is http.HandlerFunc for http.Handler.
	ModuleFunc func(injector *Injector)

	// Depender returns a list of Modules via the Depends method.
	// This allows a module to specify dependencies, which will be loaded before the actual Module is loaded.
	Depender interface {
		Depends() []Module
	}
)

// Configure calls the original ModuleFunc with the given *Injector.
func (f ModuleFunc) Configure(injector *Injector) {
	f(injector)
}

// EnableCircularTracing -
// Deprecated:
// no op
func EnableCircularTracing() {

}

// EnableInjectionTracing -
// Deprecated
// no op
func EnableInjectionTracing() {
}

func TryModule(modules ...Module) error {
	return nil
}

func NewInjector(modules ...Module) (*Injector, error) {
	return nil, nil
}

type Implementor interface {
	Child() (*dingo.Injector, error)
	InitModules(modules ...dingo.Module) error
	SetBuildEagerSingletons(build bool)
	BuildEagerSingletons(includeParent bool) error
	GetInstance(of interface{}) (interface{}, error)
	GetAnnotatedInstance(of interface{}, annotatedWith string) (interface{}, error)
	BindMulti(what interface{}) *dingo.Binding
	BindMap(what interface{}, key string) *dingo.Binding
	BindInterceptor(to, interceptor interface{})
	BindScope(s dingo.Scope)
	Bind(what interface{}) *dingo.Binding
	Override(what interface{}, annotatedWith string) *dingo.Binding
	RequestInjection(object interface{}) error
	Inspect(inspector dingo.Inspector)
}

// To binds a concrete type to a binding
func (b *Binding) To(what any) *Binding {
	return nil
}

// ToInstance binds to an instance
func (b *Binding) ToInstance(instance any) *Binding {
	return nil
}

// ToProvider binds a provider to an instance. The provider's arguments are automatically injected
func (b *Binding) ToProvider(p any) *Binding {
	return nil
}

// AnnotatedWith sets the binding's annotation
func (b *Binding) AnnotatedWith(annotation string) *Binding {
	return nil
}

// In sets the scope of the binding
func (b *Binding) In(scope Scope) *Binding {
	return nil
}

// AsEagerSingleton set's the binding to singleton and requests eager initialization
func (b *Binding) AsEagerSingleton() *Binding {
	return nil
}

// Create creates a new instance by the provider and requests injection, all provider arguments are automatically filled
func (p *Provider) Create(injector *Injector) (reflect.Value, error) {
	return reflect.Value{}, nil
}

// Inspector defines callbacks called during injector inspection
type Inspector struct {
	InspectBinding      func(of reflect.Type, annotation string, to reflect.Type, provider, instance *reflect.Value, in Scope)
	InspectMultiBinding func(of reflect.Type, index int, annotation string, to reflect.Type, provider, instance *reflect.Value, in Scope)
	InspectMapBinding   func(of reflect.Type, key string, annotation string, to reflect.Type, provider, instance *reflect.Value, in Scope)
	InspectParent       func(parent *Injector)
}

// Inspect the injector
func (injector *Injector) Inspect(inspector Inspector) {

}
