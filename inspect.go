package dingo

import "reflect"

// Inspector defines callbacks called during injector inspection.
// declaredBy is the module whose initialization created the binding.
// It is nil if that is unknown, because the binding was created outside of
// module initialization, like the injector's own binding.
type Inspector struct {
	InspectBinding      func(of reflect.Type, annotation string, to reflect.Type, provider, instance *reflect.Value, in Scope, declaredBy Module)
	InspectMultiBinding func(of reflect.Type, index int, annotation string, to reflect.Type, provider, instance *reflect.Value, in Scope, declaredBy Module)
	InspectMapBinding   func(of reflect.Type, key string, annotation string, to reflect.Type, provider, instance *reflect.Value, in Scope, declaredBy Module)
	InspectParent       func(parent *Injector)
}

// Inspect the injector
func (injector *Injector) Inspect(inspector Inspector) {
	if inspector.InspectBinding != nil {
		for t, bindings := range injector.bindings {
			for _, binding := range bindings {
				var pfnc *reflect.Value
				if binding.provider != nil {
					pfnc = &binding.provider.fnc
				}
				var ival *reflect.Value
				if binding.instance != nil {
					ival = &binding.instance.ivalue
				}

				inspector.InspectBinding(t, binding.annotatedWith, binding.to, pfnc, ival, binding.scope, binding.declaredBy)
			}
		}
	}

	if inspector.InspectMultiBinding != nil {
		for t, bindings := range injector.multibindings {
			for i, binding := range bindings {
				var pfnc *reflect.Value
				if binding.provider != nil {
					pfnc = &binding.provider.fnc
				}
				var ival *reflect.Value
				if binding.instance != nil {
					ival = &binding.instance.ivalue
				}

				inspector.InspectMultiBinding(t, i, binding.annotatedWith, binding.to, pfnc, ival, binding.scope, binding.declaredBy)
			}
		}
	}

	if inspector.InspectMapBinding != nil {
		for t, bindings := range injector.mapbindings {
			for key, binding := range bindings {
				var pfnc *reflect.Value
				if binding.provider != nil {
					pfnc = &binding.provider.fnc
				}
				var ival *reflect.Value
				if binding.instance != nil {
					ival = &binding.instance.ivalue
				}

				inspector.InspectMapBinding(t, key, binding.annotatedWith, binding.to, pfnc, ival, binding.scope, binding.declaredBy)
			}
		}
	}

	if inspector.InspectParent != nil && injector.parent != nil {
		inspector.InspectParent(injector.parent)
	}
}
