package dingo

import (
	"flamingo.me/dingo/internal"
)

type (
	// Injector defines bindings and multibindings
	// it is possible to have a parent-injector, which can be asked if no resolution is available
	Injector = internal.Injector
	// Binding defines a type mapped to a more concrete type
	Binding = internal.Binding
	// Module is the default entry point for dingo Modules.
	// The Configure method is called once during initialization,
	// and lets the module set up Bindings for the provided Injector.
	Module = internal.Module
	// ModuleFunc wraps a func(injector *Injector) for dependency injection.
	// This allows using small functions as dingo Modules.
	// The same concept is http.HandlerFunc for http.Handler.
	ModuleFunc = internal.ModuleFunc
	// Depender returns a list of Modules via the Depends method.
	// This allows a module to specify dependencies, which will be loaded before the actual Module is loaded.
	Depender = internal.Depender
	// Scope defines a scope's behavior
	Scope = internal.Scope
	// Inspector defines callbacks called during injector inspection
	Inspector = internal.Inspector
	// Instance holds quick-references to type and value
	Instance = internal.Instance
	// SingletonScope is our Scope to handle Singletons
	SingletonScope = internal.SingletonScope
	// ChildSingletonScope manages child-specific singleton
	ChildSingletonScope = internal.ChildSingletonScope
)

func NewInjector(modules ...Module) (*Injector, error) {
	return internal.NewInjector(modules...)
}

// TryModule tests if modules are properly bound
func TryModule(modules ...Module) (resultingError error) {
	return internal.TryModule(modules...)
}

// EnableCircularTracing activates dingo's trace feature to find circular dependencies
// this is super expensive (memory wise), so it should only be used for debugging purposes
func EnableCircularTracing() {
	internal.EnableCircularTracing()
}

func EnableInjectionTracing() {
	internal.EnableInjectionTracing()
}

// NewSingletonScope creates a new singleton scope
func NewSingletonScope() *SingletonScope {
	return internal.NewSingletonScope()
}

// NewChildSingletonScope creates a new child singleton scope
func NewChildSingletonScope() *ChildSingletonScope {
	return internal.NewChildSingletonScope()
}
