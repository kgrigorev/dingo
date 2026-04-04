package v2

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

	//g := &revGraph{modGraph}

	return collect(modGraph, id, modGraph.idMap)
}

func collect(g graph.Directed, from int64, idMap map[int64]dingo.Module) ([]dingo.Module, error) {
	if g.Node(from) == nil {
		return nil, fmt.Errorf("node not found in graph: %d", from)
	}

	fmt.Println("from", from, g.Node(from))

	dependencies := make([]dingo.Module, 0)

	iter := g.To(from)

	for iter.Next() {
		start := iter.Node()

		bfs := traverse.BreadthFirst{
			Visit: func(node graph.Node) {
				fmt.Println("visiting node", node)
			},

			Traverse: func(edge graph.Edge) bool {

				fmt.Println("traversing edge", edge)
				return true
			},
		}
		bfs.Walk(g, start, func(node graph.Node, depth int) bool {
			fmt.Println("start, node", start, node)
			if start.ID() != node.ID() {
				dependency := idMap[node.ID()]
				dependencies = append(dependencies, dependency)
			}

			return false // false = keep walking
		})
	}

	return dependencies, nil
}

func (modGraph *ModGraph) orderByName(nodes []graph.Node) {
	sort.SliceStable(nodes, func(i, j int) bool {
		m1 := modGraph.idMap[nodes[i].ID()]
		m2 := modGraph.idMap[nodes[j].ID()]
		n1 := identity(m1)
		n2 := identity(m2)

		return n1.String() < n2.String()
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
