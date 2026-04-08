package dingo

import (
	"fmt"
	"reflect"
	"strings"

	"gonum.org/v1/gonum/graph/multi"
)

type (
	typeGraph struct {
		*multi.DirectedGraph
		idMap     map[int64]reflect.Type
		index     map[fmt.Stringer]int64
		lineAttrs map[int64]lineAttr
	}

	lineAttr struct {
		optional   bool
		annotation string
		module     int64
	}
)

func newTypeGraph() (*typeGraph, error) {
	return &typeGraph{
		DirectedGraph: multi.NewDirectedGraph(),
		index:         make(map[fmt.Stringer]int64),
		lineAttrs:     make(map[int64]lineAttr),
	}, nil
}

func (tg *typeGraph) ensure(value any) int64 {
	key := identity(value)
	id, ok := tg.index[key]
	if !ok {
		n := tg.NewNode()
		tg.AddNode(n)
		tg.index[key] = n.ID()

		return n.ID()
	}

	return id
}

type info struct {
	annotation string
	optional   bool
}

type infoOpt func(*info)

func withOptional(value bool) infoOpt {
	return func(i *info) {
		i.optional = value
	}
}

func withAnnotation(text string) infoOpt {
	return func(i *info) {
		i.annotation = text
	}
}

func applyOpts(opts ...infoOpt) info {
	var x info
	for _, opt := range opts {
		opt(&x)
	}

	return x
}

func (tg *typeGraph) Process(bindings []*Binding) error {
	for _, binding := range bindings {
		fmt.Println("module", reflect.TypeOf(binding.source),
			"added binding for", binding.typeof,
			"annotated with", binding.annotatedWith,
			"to instance", binding.instance,
			"to type", binding.to)

		tg.traverseDependencies(binding)
	}
	return nil
}

func (tg *typeGraph) traverseDependencies(binding *Binding) error {

	rt := binding.to

	if rt == nil {
		return nil
	}

	var (
		injectlist = []reflect.Type{rt}
		i          int
	)

	fmt.Println("traversing", rt)

	for {
		if i >= len(injectlist) {
			break
		}

		current := injectlist[i]

		i++

		method, methodFound := current.MethodByName("Inject")

		if current.Kind() != reflect.Ptr && methodFound {
			return fmt.Errorf("invalid inject receiver %s: %w", current, ErrInvalidInjectReceiver)
		}

		switch current.Kind() {
		// dereference pointer
		case reflect.Ptr:
			if methodFound {
				for it := range method.Type.Ins() {
					fmt.Println("depends on type", it, "as Inject parameter")
					err := tg.AddRequiresEdge(current, it, binding.source)
					if err != nil {
						return err
					}

					injectlist = append(injectlist, it)
				}
			}

			injectlist = append(injectlist, current.Elem())

		// inject into struct fields
		case reflect.Struct:
			for fieldIndex := 0; fieldIndex < current.NumField(); fieldIndex++ {
				if tag, ok := current.Field(fieldIndex).Tag.Lookup("inject"); ok {
					field := current.Field(fieldIndex)
					currentFieldName := current.Field(fieldIndex).Name
					if field.Type.Kind() == reflect.Struct {
						return fmt.Errorf("can not inject into struct %#v of %#v", field, current)
					}

					var optional bool
					for _, option := range strings.Split(tag, ",") {
						switch strings.TrimSpace(option) {
						case "optional":
							optional = true
						}
					}
					tag = strings.Split(tag, ",")[0]

					err := tg.AddRequiresEdge(current, field.Type, binding.source, withOptional(optional), withAnnotation(tag))
					if err != nil {
						return err
					}

					fmt.Println("depends on type", field.Type, "as injected field", currentFieldName, optional)
					injectlist = append(injectlist, field.Type)
				}
			}

		case reflect.Interface:
			fmt.Println("interface processing")
			//injectlist = append(injectlist, current)

		case reflect.Slice:

		default:
		}
	}

	return nil
}

func (tg *typeGraph) AddRequiresEdge(
	from reflect.Type,
	to reflect.Type,
	module Module,
	opts ...infoOpt,
) error {
	x := applyOpts(opts...)

	ida := tg.ensure(from)
	idb := tg.ensure(to)

	fmt.Printf("type %12v (%d)\trequires %12v (%d)\t(module: %v, info: %v)\n", from, ida, to, idb, module, x)

	na := tg.Node(ida)
	nb := tg.Node(idb)

	requires := tg.NewLine(na, nb)
	tg.SetLine(requires)

	attrs := lineAttr{
		optional:   x.optional,
		annotation: x.annotation,
	}

	if module != nil {
		idm := tg.ensure(module)
		attrs.module = idm
	}

	tg.lineAttrs[requires.ID()] = attrs

	return nil
}
