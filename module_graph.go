package dingo

import (
	"fmt"
	"reflect"
	"sort"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
	"gonum.org/v1/gonum/graph/traverse"
)

type (
	modGraph struct {
		*simple.DirectedGraph // edge direction is: x (from) is a dependency of y (to)
		idMap                 map[int64]Module
		index                 map[fmt.Stringer]int64
	}
)

func newModuleGraph(modules ...Module) (*modGraph, error) {
	mg := &modGraph{
		DirectedGraph: simple.NewDirectedGraph(),
		idMap:         make(map[int64]Module),
		index:         make(map[fmt.Stringer]int64),
	}

	for _, module := range modules {
		_, err := mg.addModule(module)
		if err != nil {
			return nil, err
		}
	}

	return mg, nil
}

func (mg *modGraph) HasCycles() bool {
	return len(topo.DirectedCyclesIn(mg)) > 0
}

func (mg *modGraph) TopologicallySorted() ([]Module, error) {
	sorted, err := topo.SortStabilized(mg, mg.orderByName)
	if err != nil {
		return nil, err
	}

	var modules []Module

	for _, node := range sorted {
		modules = append(modules, mg.idMap[node.ID()])
	}

	return modules, nil
}

func (mg *modGraph) DependenciesOf(module Module) ([]Module, error) {
	ident := moduleIdentity(module)
	id, ok := mg.index[ident]
	if !ok {
		return nil, fmt.Errorf("module not found in graph: %s", ident)
	}

	return collect(mg.reversed(), id, mg.idMap)
}

func collect(g graph.Directed, from int64, idMap map[int64]Module) ([]Module, error) {
	start := g.Node(from)
	if start == nil {
		return nil, fmt.Errorf("node not found in graph: %d", from)
	}

	dependencies := make([]Module, 0)

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

func (mg *modGraph) orderByName(nodes []graph.Node) {
	sort.SliceStable(nodes, func(i, j int) bool {
		m1 := mg.idMap[nodes[i].ID()]
		m2 := mg.idMap[nodes[j].ID()]

		return modLess(m1, m2)
	})
}

func (mg *modGraph) addModule(module Module) (int64, error) {
	key := moduleIdentity(module)

	processed, ok := mg.index[key]
	if ok {
		return processed, nil
	}

	newNode := mg.NewNode()
	mg.index[key] = newNode.ID()
	mg.idMap[newNode.ID()] = module
	mg.AddNode(newNode)

	if depender, ok := module.(Depender); ok {
		for _, dep := range depender.Depends() {
			depID, err := mg.addModule(dep)
			if err != nil {
				return 0, fmt.Errorf("could not add module: %w", err)
			}

			depNode := mg.Node(depID)
			if depNode == nil {
				return 0, fmt.Errorf("dep node not found: %v", dep)
			}

			isDependencyOf := mg.NewEdge(depNode, newNode) // depNode is a dependency of newNode

			mg.SetEdge(isDependencyOf)
		}
	}

	return newNode.ID(), nil
}

func (mg *modGraph) reversed() graph.Directed {
	return &reversed{mg}
}

func moduleIdentity(module Module) fmt.Stringer {
	var key fmt.Stringer = reflect.TypeOf(module)
	if key == typeOfModuleFunc {
		key = reflect.ValueOf(module)
	}

	return key
}

func identity(value any) fmt.Stringer {
	if value == nil {
		return nil
	}

	switch item := value.(type) {
	case Module:
		return moduleIdentity(item)
	case reflect.Value:
		if item.Type() == typeOfModuleFunc {
			return item
		}

		return item.Type()
	case reflect.Type:
		return item

	default:
		return reflect.TypeOf(value)
	}
}

type reversed struct {
	graph.Directed
}

func (g *reversed) From(id int64) graph.Nodes {
	return g.Directed.To(id)
}

func (g *reversed) To(id int64) graph.Nodes {
	return g.Directed.From(id)
}

func (g *reversed) Edge(uid, vid int64) graph.Edge {
	return g.Directed.Edge(vid, uid)
}

func (g *reversed) HasEdgeFromTo(uid, vid int64) bool {
	return g.Directed.HasEdgeFromTo(vid, uid)
}

func modLess(m1, m2 Module) bool {
	n1 := moduleIdentity(m1)
	n2 := moduleIdentity(m2)

	return n1.String() < n2.String()
}
