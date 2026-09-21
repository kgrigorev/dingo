// Package bridge is the private contract between the dingo engine (flamingo.me/dingo) and the v2
// facade (flamingo.me/dingo/v2). Go's internal-package rule is import-path based, so the v2
// module may import it while nothing outside this repository can.
//
// Values are typed any because neither package can import the other's types without a cycle; each
// side asserts its own types.
package bridge

import "reflect"

// WrappedModule is implemented by module adapters. The engine keys the module graph by the
// innermost module. The method name carries the Dingo prefix so that a third-party module with an
// unrelated WrappedModule method is not unwrapped by accident.
type WrappedModule interface {
	DingoWrappedModule() any
}

// MaxDepth bounds Innermost. Two adapter layers is the deepest legitimate nesting
// (ToV0(FromV0(m))); the cap exists so that a self-returning adapter cannot loop.
const MaxDepth = 8

// Innermost follows DingoWrappedModule until a value does not implement it, returns nil, returns
// itself, or MaxDepth is reached. A nil return stops at the last non-nil value, so a broken
// adapter keeps its own identity instead of collapsing onto the nil key.
func Innermost(module any) any {
	current := module

	for range MaxDepth {
		wrapped, ok := current.(WrappedModule)
		if !ok {
			return current
		}

		inner := wrapped.DingoWrappedModule()
		if inner == nil || same(inner, current) {
			return current
		}

		current = inner
	}

	return current
}

// same reports whether left and right are the same value. It uses reflect.Value.Comparable
// rather than reflect.Type.Comparable, because a struct type holding an interface field is a
// comparable *type* while a value of it holding a function is not a comparable *value*: a plain
// == on those two panics, and a ModuleFunc adapter is exactly that shape.
func same(left, right any) bool {
	value := reflect.ValueOf(left)
	if !value.IsValid() || !value.Comparable() || value.Type() != reflect.TypeOf(right) {
		return false
	}

	return left == right
}

// Facade returns the facade attached to an engine, or nil when none was attached.
// Installed by package flamingo.me/dingo in an init function.
var Facade func(engine any) any

// Installed by package flamingo.me/dingo/v2 in an init function. Nil until then, so a binary that
// does not link v2 attaches no facade.
var (
	// NewFacade creates the facade for an engine and binds it into the engine. The root's
	// NewInjector calls it, inside the engine's construction, before any module runs.
	NewFacade func(engine any) any
	// EngineOf returns the engine behind a facade.
	EngineOf func(facade any) any
	// ToEngineModule wraps a v2 module in the engine adapter.
	ToEngineModule func(module any) any
)
