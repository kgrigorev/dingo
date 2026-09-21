package bridge_test

import (
	"testing"

	"flamingo.me/dingo/internal/bridge"
	"github.com/stretchr/testify/assert"
)

type leaf struct{ name string }

// wrapper returns whatever inner holds; a nil inner models a broken adapter.
type wrapper struct{ inner any }

func (w *wrapper) DingoWrappedModule() any { return w.inner }

// selfWrapper returns itself: a fixed point.
type selfWrapper struct{}

func (s *selfWrapper) DingoWrappedModule() any { return s }

// funcWrapper is a wrapper of uncomparable dynamic type, the shape a v2 ModuleFunc adapter takes.
type funcWrapper struct{ inner any }

func (f funcWrapper) DingoWrappedModule() any { return f.inner }

// TestInnermost_StopsOnNilFixedPointAndDepth pins the unwrap contract the engine keys its module
// graph on.
// Catches: a nil-returning adapter collapsing every wrapped module onto one nil key so that only
// the first is configured; a self-returning adapter looping forever; two distinct wrapped modules
// becoming one; and a comparability panic when the wrapped value is a function.
func TestInnermost_StopsOnNilFixedPointAndDepth(t *testing.T) {
	t.Parallel()

	t.Run("an unwrapped value is its own innermost", func(t *testing.T) {
		t.Parallel()

		module := &leaf{name: "a"}
		assert.Same(t, module, bridge.Innermost(module))
	})

	t.Run("one layer unwraps to the inner value", func(t *testing.T) {
		t.Parallel()

		module := &leaf{name: "a"}
		assert.Same(t, module, bridge.Innermost(&wrapper{inner: module}))
	})

	t.Run("two layers unwrap to the inner value", func(t *testing.T) {
		t.Parallel()

		module := &leaf{name: "a"}
		assert.Same(t, module, bridge.Innermost(&wrapper{inner: &wrapper{inner: module}}))
	})

	t.Run("an adapter returning nil keeps its own identity", func(t *testing.T) {
		t.Parallel()

		broken := &wrapper{inner: nil}
		assert.Same(t, broken, bridge.Innermost(broken))
	})

	t.Run("an adapter returning itself stops", func(t *testing.T) {
		t.Parallel()

		fixed := &selfWrapper{}
		assert.Same(t, fixed, bridge.Innermost(fixed))
	})

	t.Run("two wrappers around distinct values stay distinct", func(t *testing.T) {
		t.Parallel()

		first, second := &leaf{name: "a"}, &leaf{name: "b"}
		assert.NotSame(t, bridge.Innermost(&wrapper{inner: first}), bridge.Innermost(&wrapper{inner: second}))
	})

	t.Run("depth is capped at MaxDepth layers", func(t *testing.T) {
		t.Parallel()

		const layers = 2 * bridge.MaxDepth

		chain := make([]*wrapper, layers)

		var inner any = &leaf{name: "deep"}

		for i := layers - 1; i >= 0; i-- {
			chain[i] = &wrapper{inner: inner}
			inner = chain[i]
		}

		assert.Same(t, chain[bridge.MaxDepth], bridge.Innermost(chain[0]))
	})

	t.Run("a wrapper holding a function unwraps without a comparability panic", func(t *testing.T) {
		t.Parallel()

		fn := func() {}

		assert.NotPanics(t, func() {
			assert.IsType(t, fn, bridge.Innermost(funcWrapper{inner: fn}))
		})
	})

	t.Run("nil is nil", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, bridge.Innermost(nil))
	})
}
