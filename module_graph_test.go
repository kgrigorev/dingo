package dingo_test

import (
	"testing"

	"flamingo.me/dingo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	_ dingo.Module = new(A)
	_ dingo.Module = new(B)
	_ dingo.Module = new(C)
	_ dingo.Module = new(D)

	_ dingo.Depender = new(A)
	//_ dingo.Depender = new(B)
	_ dingo.Depender = new(C)
	//_ dingo.Depender = new(D)
	_ dingo.Depender = new(E)
)

type (
	A struct {
		withCycle   bool
		SampleText1 string `inject:"test1"`
	}
	B struct{}
	C struct{ withCycle bool }
	D struct{}
	E struct {
		SampleText2 string `inject:"test2"`
	}
)

func (a *A) Configure(i *dingo.Injector) {
	i.Bind(new(string)).AnnotatedWith("test2").ToInstance("test2")
}

func (b *B) Configure(i *dingo.Injector) {
	i.Bind(new(string)).AnnotatedWith("test1").ToInstance("test1")
}

func (c *C) Configure(_ *dingo.Injector) {
}

func (d *D) Configure(_ *dingo.Injector) {
}

func (e *E) Configure(_ *dingo.Injector) {
}

func (a *A) Depends() []dingo.Module {
	return []dingo.Module{new(B), &C{withCycle: a.withCycle}}
}

func (c *C) Depends() []dingo.Module {
	deps := []dingo.Module{
		new(B),
		new(D),
	}

	if c.withCycle {
		deps = append(deps, new(E))
	}

	return deps
}

func (e *E) Depends() []dingo.Module {
	return []dingo.Module{new(A)}
}

func TestModGraph_HasCycles(t *testing.T) {
	tests := []struct {
		name    string
		modules []dingo.Module
		want    bool
	}{
		{
			modules: []dingo.Module{&A{withCycle: false}, new(B), &C{withCycle: false}, new(D), new(E)},
			want:    false,
		},
		{
			modules: []dingo.Module{&A{withCycle: true}, new(B), &C{withCycle: true}, new(D), new(E)},
			want:    true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			modGraph, err := dingo.NewModGraph(test.modules...)
			if assert.NoError(t, err) {
				assert.Equalf(t, test.want, modGraph.HasCycles(), "HasCycles()")
			}
		})
	}
}

func TestModGraph_TopologicallySorted(t *testing.T) {
	tests := []struct {
		name      string
		modules   []dingo.Module
		sorted    []dingo.Module
		want      bool
		assertErr assert.ErrorAssertionFunc
	}{
		{
			modules:   []dingo.Module{&A{withCycle: false}, new(B), &C{withCycle: false}, new(D), new(E)},
			sorted:    []dingo.Module{new(B), new(D), &C{withCycle: false}, &A{withCycle: false}, new(E)},
			assertErr: assert.NoError,
		},
		{
			modules:   []dingo.Module{&A{withCycle: true}, new(B), &C{withCycle: true}, new(D), new(E)},
			assertErr: assert.Error,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			modGraph, err := dingo.NewModGraph(test.modules...)
			require.NoError(t, err)

			sorted, err := modGraph.TopologicallySorted()
			if test.assertErr(t, err) {
				assert.Equalf(t, test.sorted, sorted, "TopologicallySorted()")
			}
		})
	}
}

func TestModGraph_DependenciesOf(t *testing.T) {
	allModules := func() []dingo.Module {
		return []dingo.Module{new(A), new(B), new(C), new(D), new(E)}
	}

	tests := []struct {
		name    string
		modules []dingo.Module
		start   dingo.Module
		want    []dingo.Module
	}{
		{
			name:    "module B has no dependencies",
			modules: allModules(),
			start:   new(B),
			want:    []dingo.Module{},
		},
		{
			name:    "module D has no dependencies",
			modules: allModules(),
			start:   new(D),
			want:    []dingo.Module{},
		},
		{
			name:    "module C depends on B and D",
			modules: allModules(),
			start:   new(C),
			want:    []dingo.Module{new(B), new(D)},
		},
		{
			name:    "module A depends on B, C and D",
			modules: allModules(),
			start:   new(A),
			want:    []dingo.Module{new(B), new(C), new(D)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			modGraph, err := dingo.NewModGraph(test.modules...)
			require.NoError(t, err)

			dependencies, err := modGraph.DependenciesOf(test.start)
			require.NoError(t, err)

			assert.Equalf(t, test.want, dependencies, "DependenciesOf()")
		})
	}
}

func TestWitInjector(t *testing.T) {
	injector, err := dingo.NewInjector()
	require.NoError(t, err)

	//modules := []dingo.Module{new(E), new(D), new(C), new(B), new(A)}
	modules := []dingo.Module{new(A), new(B), new(C), new(D), new(E)}

	err = injector.InitModules(modules...)
	assert.NoError(t, err)
}
