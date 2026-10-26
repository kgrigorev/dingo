package dingo

import (
	"fmt"
	"reflect"
)

type (
	// Binding defines a type mapped to a more concrete type
	Binding struct {
		typeof reflect.Type

		to       reflect.Type
		instance *Instance
		provider *Provider

		eager         bool
		annotatedWith string
		scope         Scope

		declaredBy Module // module whose configuration created the binding, nil if none
	}

	// Instance holds quick-references to type and value
	Instance struct {
		itype  reflect.Type
		ivalue reflect.Value
	}

	// Provider holds the provider function
	Provider struct {
		fnctype reflect.Type
		fnc     reflect.Value
		binding *Binding
	}
)

// To binds a concrete type to a binding
func (b *Binding) To(what interface{}) *Binding {
	to := reflect.TypeOf(what)

	for to.Kind() == reflect.Ptr {
		to = to.Elem()
	}

	if !to.AssignableTo(b.typeof) && !reflect.PtrTo(to).AssignableTo(b.typeof) {
		panic(fmt.Sprintf("%s#%s not assignable to %s#%s", to.PkgPath(), to.Name(), b.typeof.PkgPath(), b.typeof.Name()))
	}

	b.to = to

	return b
}

// ToInstance binds an instance to a binding
func (b *Binding) ToInstance(instance interface{}) *Binding {
	b.instance = &Instance{
		itype:  reflect.TypeOf(instance),
		ivalue: reflect.ValueOf(instance),
	}
	if !b.instance.itype.AssignableTo(b.typeof) && !b.instance.itype.AssignableTo(reflect.PtrTo(b.typeof)) {
		panic(fmt.Sprintf("%s#%s not assignable to %s#%s", b.instance.itype.PkgPath(), b.instance.itype.Name(), b.typeof.PkgPath(), b.typeof.Name()))
	}
	return b
}

// ToProvider binds a provider to an instance. The provider's arguments are automatically injected
func (b *Binding) ToProvider(p interface{}) *Binding {
	provider := &Provider{
		fnc:     reflect.ValueOf(p),
		binding: b,
	}
	provider.fnctype = provider.fnc.Type().Out(0)
	if !provider.fnctype.AssignableTo(b.typeof) && !provider.fnctype.AssignableTo(reflect.PtrTo(b.typeof)) {
		panic(fmt.Sprintf("provider returns %q which is not assignable to %q", provider.fnctype, b.typeof))
	}
	b.provider = provider
	return b
}

// AnnotatedWith sets the binding's annotation
func (b *Binding) AnnotatedWith(annotation string) *Binding {
	b.annotatedWith = annotation
	return b
}

// In set's the bindings scope
func (b *Binding) In(scope Scope) *Binding {
	b.scope = scope
	return b
}

// AsEagerSingleton set's the binding to singleton and requests eager initialization
func (b *Binding) AsEagerSingleton() *Binding {
	b.In(Singleton)
	b.eager = true
	return b
}

// equal reports whether both bindings bind the same way. The declaring module
// is ignored, so identical bindings from different modules do not conflict.
func (b *Binding) equal(to *Binding) bool {
	return b.typeof == to.typeof &&
		b.to == to.to &&
		reflect.DeepEqual(b.instance, to.instance) &&
		b.provider.equal(to.provider) &&
		b.eager == to.eager &&
		b.annotatedWith == to.annotatedWith &&
		reflect.DeepEqual(b.scope, to.scope)
}

// equal reports whether both providers call the same function. The binding
// back-pointer is skipped because it points at the bindings being compared.
func (p *Provider) equal(to *Provider) bool {
	if p == nil || to == nil {
		return p == to
	}

	return p.fnctype == to.fnctype && reflect.DeepEqual(p.fnc, to.fnc)
}

// Create creates a new instance by the provider and requests injection, all provider arguments are automatically filled
func (p *Provider) Create(injector *Injector) (reflect.Value, error) {
	in := make([]reflect.Value, p.fnc.Type().NumIn())
	var err error
	for i := 0; i < p.fnc.Type().NumIn(); i++ {
		if in[i], err = injector.getInstance(p.fnc.Type().In(i), "", traceCircular); err != nil {
			return reflect.Value{}, err
		}
		for !in[i].Type().AssignableTo(p.fnc.Type().In(i)) && in[i].Kind() == reflect.Ptr {
			in[i] = in[i].Elem()
		}
	}
	res := p.fnc.Call(in)[0]
	return res, injector.requestInjection(res, traceCircular)
}
