// Package hooks is a private link between flamingo.me/dingo (the root injector)
// and flamingo.me/dingo/v2 (the typed API). Only this repository can import it.
//
// Values use any so the two packages do not import each other.
// Each side casts to its own types.
package hooks

import "reflect"

// Unwrapper is for module adapters.
// The root injector keys the module graph by the inner module, not the adapter.
// The method is named DingoUnwrap so unrelated Unwrap methods are ignored.
type Unwrapper interface {
	DingoUnwrap() any
}

// MaxUnwrapDepth limits how far Unwrap walks.
// Real use needs at most two layers: ToRoot(FromRoot(m)).
// The higher number stops a broken adapter that returns itself from looping forever.
const MaxUnwrapDepth = 8

// Unwrap keeps calling DingoUnwrap to find the inner module.
// It stops when the value is not an Unwrapper, when DingoUnwrap returns nil,
// when it returns itself, or when MaxUnwrapDepth is reached.
//
// A nil return keeps the last non-nil value.
// That way broken adapters do not all collapse onto one nil key.
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

// same is true when left and right are the same value.
// We use reflect.Value.Comparable, not Type.Comparable.
// A struct type with an interface field can look comparable as a type,
// but a value that holds a function is not comparable — == would panic.
// ModuleFunc adapters look like that.
func same(left, right any) bool {
	value := reflect.ValueOf(left)
	if !value.IsValid() || !value.Comparable() || value.Type() != reflect.TypeOf(right) {
		return false
	}

	return left == right
}

// Attached returns the typed injector stored on a root injector.
// It returns nil if nothing was attached.
// flamingo.me/dingo sets this in init.
var Attached func(root any) any

// flamingo.me/dingo/v2 sets these in init.
// They stay nil until v2 is linked, so Attach does nothing in a root-only binary.
var (
	// Attach creates the typed injector and binds it into the root.
	// NewInjector calls it while building the injector, before any module runs.
	Attach func(root any) any
	// RootOf returns the root injector for a typed injector.
	RootOf func(attached any) any
	// AsModule wraps a v2 module so the root injector can load it.
	AsModule func(module any) any
)
