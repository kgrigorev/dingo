// Package hooks is the private contract between the root injector (flamingo.me/dingo) and the
// typed API (flamingo.me/dingo/v2). It lives under internal/, so only this repository can import
// it (including the v2 module).
//
// Values are typed as any to avoid an import cycle; each side asserts its own types.
package hooks

import "reflect"

// Unwrapper marks a module adapter. The root injector builds the module graph from the inner
// (unwrapped) module. The Dingo-prefixed method name avoids colliding with unrelated Unwrap methods.
type Unwrapper interface {
	DingoUnwrap() any
}

// MaxUnwrapDepth is the maximum Unwrap recursion. Real nesting stops at two layers
// (ToRoot(FromRoot(m))); the higher cap stops a self-returning adapter from looping forever.
const MaxUnwrapDepth = 8

// Unwrap walks DingoUnwrap until the value is not an Unwrapper, returns nil, returns itself, or
// MaxUnwrapDepth is hit. On nil, Unwrap keeps the last non-nil value so a broken adapter does not
// share one nil key with every other broken adapter.
func Unwrap(module any) any {
	current := module

	for range MaxUnwrapDepth {
		wrapped, ok := current.(Unwrapper)
		if !ok {
			return current
		}

		inner := wrapped.DingoUnwrap()
		if inner == nil || same(inner, current) {
			return current
		}

		current = inner
	}

	return current
}

// same reports whether left and right are the same value. It checks reflect.Value.Comparable, not
// Type.Comparable: a struct type with an interface field can be a comparable type, yet a value that
// holds a function is not a comparable value — plain == would panic. ModuleFunc adapters have that shape.
func same(left, right any) bool {
	value := reflect.ValueOf(left)
	if !value.IsValid() || !value.Comparable() || value.Type() != reflect.TypeOf(right) {
		return false
	}

	return left == right
}

// Attached returns the typed injector on a root injector, or nil if none.
// Set by flamingo.me/dingo in an init function.
var Attached func(root any) any

// Set by flamingo.me/dingo/v2 in an init function. Nil when v2 is not linked, so Attach does nothing.
var (
	// Attach builds the typed injector for a root injector and binds it there. NewInjector calls
	// it during construction, before any module runs.
	Attach func(root any) any
	// RootOf returns the root injector behind a typed injector.
	RootOf func(attached any) any
	// AsModule wraps a v2 module as a root Module.
	AsModule func(module any) any
)
