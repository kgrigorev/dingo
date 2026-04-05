package internal

import (
	"fmt"
	"reflect"
	"sort"

	"flamingo.me/dingo"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
	"gonum.org/v1/gonum/graph/traverse"
)

var (
	typeOfModuleFunc = reflect.TypeOf(dingo.ModuleFunc(nil))
)

type (
	ModGraph struct {
		*simple.DirectedGraph // edge direction is x (from) is a dependency of y (to)
		idMap                 map[int64]dingo.Module
		index                 map[fmt.Stringer]int64
	}
)

func NewModGraph(modules ...dingo.Module) (*ModGraph, error) {
	modGraph := &ModGraph{
		DirectedGraph: simple.NewDirectedGraph(),
		idMap:         make(map[int64]dingo.Module),
		index:         make(map[fmt.Stringer]int64),
	}

	for _, module := range modules {
		_, err := modGraph.addModule(module)
		if err != nil {
			return nil, err
		}
	}

	return modGraph, nil
}

func (modGraph *ModGraph) HasCycles() bool {
	return len(topo.DirectedCyclesIn(modGraph)) > 0
}

func (modGraph *ModGraph) TopologicallySorted() ([]dingo.Module, error) {
	sorted, err := topo.SortStabilized(modGraph, modGraph.orderByName)
	if err != nil {
		return nil, err
	}

	var modules []dingo.Module

	for _, node := range sorted {
		modules = append(modules, modGraph.idMap[node.ID()])
	}

	return modules, nil
}

func (modGraph *ModGraph) DependenciesOf(module dingo.Module) ([]dingo.Module, error) {
	ident := identity(module)
	id, ok := modGraph.index[ident]
	if !ok {
		return nil, fmt.Errorf("module not found in graph: %s", ident)
	}

	return collect(modGraph.reversed(), id, modGraph.idMap)
}

func collect(g graph.Directed, from int64, idMap map[int64]dingo.Module) ([]dingo.Module, error) {
	start := g.Node(from)
	if start == nil {
		return nil, fmt.Errorf("node not found in graph: %d", from)
	}

	dependencies := make([]dingo.Module, 0)

	bfs := traverse.BreadthFirst{}
	bfs.Walk(g, start, func(node graph.Node, depth int) bool {
		if start.ID() != node.ID() {
			dependency := idMap[node.ID()]
			dependencies = append(dependencies, dependency)
		}

		return false // false = keep walking
	})

	sort.SliceStable(dependencies, func(i, j int) bool {
		m1 := dependencies[i]
		m2 := dependencies[j]

		return modLess(m1, m2)
	})

	return dependencies, nil
}

func (modGraph *ModGraph) orderByName(nodes []graph.Node) {
	sort.SliceStable(nodes, func(i, j int) bool {
		m1 := modGraph.idMap[nodes[i].ID()]
		m2 := modGraph.idMap[nodes[j].ID()]

		return modLess(m1, m2)
	})
}

func (modGraph *ModGraph) addModule(module dingo.Module) (int64, error) {
	key := identity(module)

	processed, ok := modGraph.index[key]
	if ok {
		return processed, nil
	}

	newNode := modGraph.NewNode()
	modGraph.index[key] = newNode.ID()
	modGraph.idMap[newNode.ID()] = module
	modGraph.AddNode(newNode)

	if depender, ok := module.(dingo.Depender); ok {
		for _, dep := range depender.Depends() {
			depID, err := modGraph.addModule(dep)
			if err != nil {
				return 0, fmt.Errorf("could not add module: %w", err)
			}

			depNode := modGraph.Node(depID)
			if depNode == nil {
				return 0, fmt.Errorf("dep node not found: %v", dep)
			}

			isDependencyOf := modGraph.NewEdge(depNode, newNode) // dn is dependency of n

			modGraph.SetEdge(isDependencyOf)
		}
	}

	return newNode.ID(), nil
}

func (modGraph *ModGraph) reversed() graph.Directed {
	return &revGraph{modGraph}
}

func identity(module dingo.Module) fmt.Stringer {
	var key fmt.Stringer = reflect.TypeOf(module)
	if key == typeOfModuleFunc {
		key = reflect.ValueOf(module)
	}

	return key
}

type revGraph struct {
	graph.Directed
}

func (g revGraph) From(i int64) graph.Nodes {
	return g.Directed.To(i)
}

func (g revGraph) To(i int64) graph.Nodes {
	return g.Directed.From(i)
}

func (g revGraph) Edge(uid, vid int64) graph.Edge {
	return g.Directed.Edge(vid, uid)
}

func (g revGraph) HasEdgeFromTo(uid, vid int64) bool {
	return g.Directed.HasEdgeFromTo(vid, uid)
}

func modLess(m1, m2 dingo.Module) bool {
	n1 := identity(m1)
	n2 := identity(m2)

	return n1.String() < n2.String()
}
