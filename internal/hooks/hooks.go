// Package hooks is the private contract between the dingo root injector (flamingo.me/dingo) and
// the typed API (flamingo.me/dingo/v2). Go's internal-package rule is import-path based, so the v2
// module may import it while nothing outside this repository can.
//
// Values are typed any because neither package can import the other's types without a cycle; each
// side asserts its own types.
package hooks

import "reflect"

// Unwrapper is implemented by module adapters. The engine keys the module graph by the
// unwrapped module. The method name carries the Dingo prefix so that a third-party module with an
// unrelated Unwrapper method is not unwrapped by accident.
type Unwrapper interface {
	DingoUnwrap() any
}

// MaxUnwrapDepth bounds Unwrap. Two adapter layers is the deepest legitimate nesting
// (ToRoot(FromRoot(m))); the cap exists so that a self-returning adapter cannot loop.
const MaxUnwrapDepth = 8

// Unwrap follows DingoUnwrap until a value does not implement Unwrapper, returns nil, returns
// itself, or MaxUnwrapDepth is reached. A nil return stops at the last non-nil value, so a broken
// adapter keeps its own identity instead of collapsing onto the nil key.
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

// Attached returns the typed injector attached to a root injector, or nil when none was attached.
// Installed by package flamingo.me/dingo in an init function.
var Attached func(root any) any

// Installed by package flamingo.me/dingo/v2 in an init function. Nil until then, so a binary that
// does not link v2 attaches nothing.
var (
	// Attach creates the typed injector for a root injector and binds it into the root. The root's
	// NewInjector calls it, inside the root injector's construction, before any module runs.
	Attach func(root any) any
	// RootOf returns the root injector behind a typed injector.
	RootOf func(attached any) any
	// AsModule wraps a v2 module in the root Module adapter.
	AsModule func(module any) any
)
