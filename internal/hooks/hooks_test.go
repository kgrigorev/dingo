package hooks_test

import (
	"testing"

	"flamingo.me/dingo/internal/hooks"
	"github.com/stretchr/testify/assert"
)

type leaf struct{ name string }

// wrapper returns its inner value; nil inner models a broken adapter.
type wrapper struct{ inner any }

func (w *wrapper) DingoUnwrap() any { return w.inner }

// selfWrapper returns itself (fixed point).
type selfWrapper struct{}

func (s *selfWrapper) DingoUnwrap() any { return s }

// funcWrapper wraps an uncomparable value, like a v2 ModuleFunc adapter.
type funcWrapper struct{ inner any }

func (f funcWrapper) DingoUnwrap() any { return f.inner }

// TestUnwrap_StopsOnNilFixedPointAndDepth pins the Unwrap rules the root module graph keys on.
// Catches: nil-returning adapters all mapping to one nil key; self-returning adapters looping;
// distinct wrapped modules becoming one; comparability panic when the wrapped value is a function.
func TestUnwrap_StopsOnNilFixedPointAndDepth(t *testing.T) {
	t.Parallel()

	t.Run("an unwrapped value is its own result", func(t *testing.T) {
		t.Parallel()

		module := &leaf{name: "a"}
		assert.Same(t, module, hooks.Unwrap(module))
	})

	t.Run("one layer unwraps to the inner value", func(t *testing.T) {
		t.Parallel()

		module := &leaf{name: "a"}
		assert.Same(t, module, hooks.Unwrap(&wrapper{inner: module}))
	})

	t.Run("two layers unwrap to the inner value", func(t *testing.T) {
		t.Parallel()

		module := &leaf{name: "a"}
		assert.Same(t, module, hooks.Unwrap(&wrapper{inner: &wrapper{inner: module}}))
	})

	t.Run("an adapter returning nil keeps its own identity", func(t *testing.T) {
		t.Parallel()

		broken := &wrapper{inner: nil}
		assert.Same(t, broken, hooks.Unwrap(broken))
	})

	t.Run("an adapter returning itself stops", func(t *testing.T) {
		t.Parallel()

		fixed := &selfWrapper{}
		assert.Same(t, fixed, hooks.Unwrap(fixed))
	})

	t.Run("two wrappers around distinct values stay distinct", func(t *testing.T) {
		t.Parallel()

		first, second := &leaf{name: "a"}, &leaf{name: "b"}
		assert.NotSame(t, hooks.Unwrap(&wrapper{inner: first}), hooks.Unwrap(&wrapper{inner: second}))
	})

	t.Run("depth is capped at MaxUnwrapDepth layers", func(t *testing.T) {
		t.Parallel()

		const layers = 2 * hooks.MaxUnwrapDepth

		chain := make([]*wrapper, layers)

		var inner any = &leaf{name: "deep"}

		for i := layers - 1; i >= 0; i-- {
			chain[i] = &wrapper{inner: inner}
			inner = chain[i]
		}

		assert.Same(t, chain[hooks.MaxUnwrapDepth], hooks.Unwrap(chain[0]))
	})

	t.Run("a wrapper holding a function unwraps without a comparability panic", func(t *testing.T) {
		t.Parallel()

		fn := func() {}

		assert.NotPanics(t, func() {
			assert.IsType(t, fn, hooks.Unwrap(funcWrapper{inner: fn}))
		})
	})

	t.Run("nil is nil", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, hooks.Unwrap(nil))
	})
}
