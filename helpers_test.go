package dingo_test

import (
	"sync"
	"testing"

	"flamingo.me/dingo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// privateSingletons maps an injector to the *SingletonScope newInjector registered on it.
// childOf loads the parent entry and BindScope's that same pointer onto the child.
// Child() does not copy it: NewInjector registers the package-level Singleton instead.
var privateSingletons sync.Map

// newInjector returns an injector with private Singleton and ChildSingleton caches.
//
// NewInjector is called with no modules so eager construction cannot land in the
// package-level scopes. The private scopes are registered next, then modules initialize.
func newInjector(t *testing.T) *dingo.Injector {
	t.Helper()

	injector, err := dingo.NewInjector()
	require.NoError(t, err)

	singleton := dingo.NewSingletonScope()
	injector.BindScope(singleton)
	injector.BindScope(dingo.NewChildSingletonScope())
	privateSingletons.Store(injector, singleton)

	err = injector.InitModules()
	require.NoError(t, err)

	return injector
}

// childOf returns a child that resolves In(Singleton) through the parent's private
// SingletonScope. Child() has already installed a fresh ChildSingletonScope; leave that in place.
func childOf(t *testing.T, parent *dingo.Injector) *dingo.Injector {
	t.Helper()

	child, err := parent.Child()
	require.NoError(t, err)

	loaded, ok := privateSingletons.Load(parent)
	require.True(t, ok, "childOf parent must come from newInjector or childOf")

	singleton, ok := loaded.(*dingo.SingletonScope)
	require.True(t, ok, "private singleton has type %T", loaded)

	child.BindScope(singleton)
	privateSingletons.Store(child, singleton)

	return child
}

func mustInstance(t *testing.T, injector *dingo.Injector, of any) any {
	t.Helper()

	instance, err := injector.GetInstance(of)
	require.NoError(t, err)

	return instance
}

// harnessSingletonWidget is resolved only through newInjector, In(Singleton).
type harnessSingletonWidget struct {
	Token string
}

// harnessChildSingletonWidget is resolved only through newInjector, In(ChildSingleton),
// on a top-level injector. It is not the per-child fixture in scope_semantics_test.go.
type harnessChildSingletonWidget struct {
	Token string
}

// harnessChildSharedWidget is the only fixture TestHelper_ChildSharesParentsPrivateSingleton
// resolves, including once through dingo.NewInjector. No other test may mention it.
type harnessChildSharedWidget struct {
	Token string
}

type harnessUnboundDep interface {
	harnessUnbound()
}

// harnessFailedSingleton fails construction because Dep is an unbound interface.
// Dep is exported so package dingo can see it; construction fails before the field is set.
type harnessFailedSingleton struct {
	Token string
	Dep   harnessUnboundDep `inject:""`
}

func TestHelper_PrivateScopesIsolateTwoInjectors(t *testing.T) {
	t.Parallel()

	left := newInjector(t)
	right := newInjector(t)

	left.Bind(new(harnessSingletonWidget)).In(dingo.Singleton)
	right.Bind(new(harnessSingletonWidget)).In(dingo.Singleton)

	leftSingleton := mustInstance(t, left, new(harnessSingletonWidget))
	leftSingletonAgain := mustInstance(t, left, new(harnessSingletonWidget))
	rightSingleton := mustInstance(t, right, new(harnessSingletonWidget))

	assert.Same(t, leftSingleton, leftSingletonAgain)
	assert.NotSame(t, leftSingleton, rightSingleton)

	leftWidget, ok := leftSingleton.(*harnessSingletonWidget)
	require.True(t, ok)

	leftWidget.Token = "left"

	leftAgain, ok := leftSingletonAgain.(*harnessSingletonWidget)
	require.True(t, ok)
	assert.Equal(t, "left", leftAgain.Token)

	rightWidget, ok := rightSingleton.(*harnessSingletonWidget)
	require.True(t, ok)
	assert.Empty(t, rightWidget.Token)

	left.Bind(new(harnessChildSingletonWidget)).In(dingo.ChildSingleton)
	right.Bind(new(harnessChildSingletonWidget)).In(dingo.ChildSingleton)

	leftChildScoped := mustInstance(t, left, new(harnessChildSingletonWidget))
	leftChildScopedAgain := mustInstance(t, left, new(harnessChildSingletonWidget))
	rightChildScoped := mustInstance(t, right, new(harnessChildSingletonWidget))

	assert.Same(t, leftChildScoped, leftChildScopedAgain)
	assert.NotSame(t, leftChildScoped, rightChildScoped)

	leftChildWidget, ok := leftChildScoped.(*harnessChildSingletonWidget)
	require.True(t, ok)

	leftChildWidget.Token = "child-scoped"

	leftChildAgain, ok := leftChildScopedAgain.(*harnessChildSingletonWidget)
	require.True(t, ok)
	assert.Equal(t, "child-scoped", leftChildAgain.Token)

	rightChildWidget, ok := rightChildScoped.(*harnessChildSingletonWidget)
	require.True(t, ok)
	assert.Empty(t, rightChildWidget.Token)
}

func TestHelper_ChildSharesParentsPrivateSingleton(t *testing.T) {
	t.Parallel()

	parent := newInjector(t)
	parent.Bind(new(harnessChildSharedWidget)).In(dingo.Singleton)

	child := childOf(t, parent)

	fromParent := mustInstance(t, parent, new(harnessChildSharedWidget))
	fromChild := mustInstance(t, child, new(harnessChildSharedWidget))
	assert.Same(t, fromParent, fromChild)

	shared, ok := fromParent.(*harnessChildSharedWidget)
	require.True(t, ok)

	shared.Token = "shared"

	fromChildAgain, ok := fromChild.(*harnessChildSharedWidget)
	require.True(t, ok)
	assert.Equal(t, "shared", fromChildAgain.Token)

	// This NewInjector is the negative control: it must not see the private cache.
	// Do not resolve harnessChildSharedWidget from any other test.
	globalInjector, err := dingo.NewInjector()
	require.NoError(t, err)

	globalInjector.Bind(new(harnessChildSharedWidget)).In(dingo.Singleton)

	fromGlobal := mustInstance(t, globalInjector, new(harnessChildSharedWidget))
	assert.NotSame(t, fromParent, fromGlobal)

	globalWidget, ok := fromGlobal.(*harnessChildSharedWidget)
	require.True(t, ok)
	assert.Empty(t, globalWidget.Token)
}

func TestHelper_FailedSingletonConstructionDoesNotLeak(t *testing.T) {
	t.Parallel()

	left := newInjector(t)
	right := newInjector(t)

	left.Bind(new(harnessFailedSingleton)).In(dingo.Singleton)
	right.Bind(new(harnessFailedSingleton)).In(dingo.Singleton)

	// The first call returns the construction error and a nil value. The scope has
	// already stored the allocated struct, so the next call on this injector returns
	// that struct with a nil error. The other injector must still fail.
	first, err := left.GetInstance(new(harnessFailedSingleton))
	require.Error(t, err)
	assert.Nil(t, first)

	second, err := left.GetInstance(new(harnessFailedSingleton))
	require.NoError(t, err)
	require.NotNil(t, second)

	leftHost, ok := second.(*harnessFailedSingleton)
	require.True(t, ok)
	assert.Nil(t, leftHost.Dep)

	leftHost.Token = "half"

	third, err := left.GetInstance(new(harnessFailedSingleton))
	require.NoError(t, err)
	assert.Same(t, second, third)

	seenAgain, ok := third.(*harnessFailedSingleton)
	require.True(t, ok)
	assert.Equal(t, "half", seenAgain.Token)

	otherFirst, err := right.GetInstance(new(harnessFailedSingleton))
	require.Error(t, err)
	assert.Nil(t, otherFirst)

	otherSecond, err := right.GetInstance(new(harnessFailedSingleton))
	require.NoError(t, err)
	assert.NotSame(t, second, otherSecond)

	rightHost, ok := otherSecond.(*harnessFailedSingleton)
	require.True(t, ok)
	assert.Empty(t, rightHost.Token)
	assert.Nil(t, rightHost.Dep)
}
