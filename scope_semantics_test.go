package dingo_test

import (
	"errors"
	"reflect"
	"testing"

	"flamingo.me/dingo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type singletonCachedWidget struct {
	Token string
}

func TestScope_SingletonCachesWithinOneInjector(t *testing.T) {
	t.Parallel()

	injector := newInjector(t)
	injector.Bind(new(singletonCachedWidget)).In(dingo.Singleton)

	first := mustInstance(t, injector, new(singletonCachedWidget))
	second := mustInstance(t, injector, new(singletonCachedWidget))
	assert.Same(t, first, second)

	widget, ok := first.(*singletonCachedWidget)
	require.True(t, ok)

	widget.Token = "cached"

	again, ok := second.(*singletonCachedWidget)
	require.True(t, ok)
	assert.Equal(t, "cached", again.Token)
}

type childSingletonCachedWidget struct {
	Token string
}

func TestScope_ChildSingletonCachesPerChild(t *testing.T) {
	t.Parallel()

	parent := newInjector(t)
	parent.Bind(new(childSingletonCachedWidget)).In(dingo.ChildSingleton)

	left := childOf(t, parent)
	right := childOf(t, parent)

	leftFirst := mustInstance(t, left, new(childSingletonCachedWidget))
	leftSecond := mustInstance(t, left, new(childSingletonCachedWidget))
	assert.Same(t, leftFirst, leftSecond)

	leftWidget, ok := leftFirst.(*childSingletonCachedWidget)
	require.True(t, ok)

	leftWidget.Token = "left"

	leftAgain, ok := leftSecond.(*childSingletonCachedWidget)
	require.True(t, ok)
	assert.Equal(t, "left", leftAgain.Token)

	rightFirst := mustInstance(t, right, new(childSingletonCachedWidget))
	assert.NotSame(t, leftFirst, rightFirst)

	rightWidget, ok := rightFirst.(*childSingletonCachedWidget)
	require.True(t, ok)
	assert.Empty(t, rightWidget.Token)
}

type unknownScopeWidget struct{}

type unregisteredScope struct{}

var errUnregisteredScopeInvoked = errors.New("unregistered scope was invoked")

func (unregisteredScope) ResolveType(
	_ reflect.Type,
	_ string,
	_ func(reflect.Type, string, bool) (reflect.Value, error),
) (reflect.Value, error) {
	return reflect.Value{}, errUnregisteredScopeInvoked
}

var _ dingo.Scope = unregisteredScope{}

func TestScope_UnknownScopeErrors(t *testing.T) {
	t.Parallel()

	injector := newInjector(t)
	injector.Bind(new(unknownScopeWidget)).In(unregisteredScope{})

	instance, err := injector.GetInstance(new(unknownScopeWidget))
	require.ErrorContains(t, err, "unknown scope")
	require.ErrorContains(t, err, "unregisteredScope")
	require.ErrorContains(t, err, "unknownScopeWidget")
	assert.Nil(t, instance)
}

type customScopeWidget struct {
	Token string
}

type countingScope struct {
	resolutions int
}

func (s *countingScope) ResolveType(
	bound reflect.Type,
	annotation string,
	unscoped func(bound reflect.Type, annotation string, optional bool) (reflect.Value, error),
) (reflect.Value, error) {
	s.resolutions++

	return unscoped(bound, annotation, false)
}

var _ dingo.Scope = (*countingScope)(nil)

func TestScope_CustomScopeResolves(t *testing.T) {
	t.Parallel()

	injector := newInjector(t)
	scope := &countingScope{}
	injector.BindScope(scope)
	injector.Bind(new(customScopeWidget)).In(scope)

	instance := mustInstance(t, injector, new(customScopeWidget))

	widget, ok := instance.(*customScopeWidget)
	require.True(t, ok)
	assert.Empty(t, widget.Token)
	assert.Equal(t, 1, scope.resolutions)
}

type replacedSingletonWidget struct {
	Token string
}

type replacedChildSingletonWidget struct {
	Token string
}

func TestBindScope_FreshScopeReplacesTheCache(t *testing.T) {
	t.Parallel()

	t.Run("singleton", func(t *testing.T) {
		t.Parallel()

		injector := newInjector(t)
		injector.Bind(new(replacedSingletonWidget)).In(dingo.Singleton)

		first := mustInstance(t, injector, new(replacedSingletonWidget))
		injector.BindScope(dingo.NewSingletonScope())

		second := mustInstance(t, injector, new(replacedSingletonWidget))
		assert.NotSame(t, first, second)

		firstWidget, ok := first.(*replacedSingletonWidget)
		require.True(t, ok)

		firstWidget.Token = "before-swap"

		secondWidget, ok := second.(*replacedSingletonWidget)
		require.True(t, ok)
		assert.Empty(t, secondWidget.Token)
		assert.Equal(t, "before-swap", firstWidget.Token)
	})

	t.Run("child singleton", func(t *testing.T) {
		t.Parallel()

		injector := newInjector(t)
		injector.Bind(new(replacedChildSingletonWidget)).In(dingo.ChildSingleton)

		first := mustInstance(t, injector, new(replacedChildSingletonWidget))
		injector.BindScope(dingo.NewChildSingletonScope())

		second := mustInstance(t, injector, new(replacedChildSingletonWidget))
		assert.NotSame(t, first, second)

		firstWidget, ok := first.(*replacedChildSingletonWidget)
		require.True(t, ok)

		firstWidget.Token = "before-swap"

		secondWidget, ok := second.(*replacedChildSingletonWidget)
		require.True(t, ok)
		assert.Empty(t, secondWidget.Token)
		assert.Equal(t, "before-swap", firstWidget.Token)
	})
}
