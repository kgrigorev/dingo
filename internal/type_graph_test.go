package internal

import (
	"testing"

	"flamingo.me/dingo"
)

type Mod struct {
}

func (m *Mod) Configure(i *dingo.Injector) {
	i.Bind(new(string)).ToInstance("test")
}

func TestTypeGraph(t *testing.T) {
	//new(Mod).Configure(new(MyInj).Injector)
}

type MyInj struct {
	*dingo.Injector
}

func (m *MyInj) Bind(what interface{}) *dingo.Binding {
	return m.Injector.Bind(what)
}
