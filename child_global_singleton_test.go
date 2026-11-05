package dingo_test

import (
	"testing"

	"flamingo.me/dingo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// globalSingletonProbe is resolved only by TestChild_SingletonScopeIsSharedGloballyNotPerInjector,
// through dingo.NewInjector, with no private scope. No other test may name this type.
type globalSingletonProbe struct {
	Token string
}

func TestChild_SingletonScopeIsSharedGloballyNotPerInjector(t *testing.T) { //nolint:paralleltest // writes the package-level Singleton cache for globalSingletonProbe and must stay sequential
	firstInjector, err := dingo.NewInjector()
	require.NoError(t, err)

	secondInjector, err := dingo.NewInjector()
	require.NoError(t, err)

	firstInjector.Bind(new(globalSingletonProbe)).In(dingo.Singleton)
	secondInjector.Bind(new(globalSingletonProbe)).In(dingo.Singleton)

	fromFirst, err := firstInjector.GetInstance(new(globalSingletonProbe))
	require.NoError(t, err)

	fromSecond, err := secondInjector.GetInstance(new(globalSingletonProbe))
	require.NoError(t, err)
	assert.Same(t, fromFirst, fromSecond)

	probe, ok := fromFirst.(*globalSingletonProbe)
	require.True(t, ok)

	probe.Token = "shared"

	seenFromSecond, ok := fromSecond.(*globalSingletonProbe)
	require.True(t, ok)
	assert.Equal(t, "shared", seenFromSecond.Token)
}
