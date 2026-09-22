// Package typename formats reflect types with their full import path.
//
// The root injector (module graph errors) and the typed API (bind-time messages) share it so both
// print the same type name. It is under internal/, so only this repository can import it.
package typename

import (
	"fmt"
	"reflect"
)

// Qualified is like reflect.Type.String, but named types use the full import path instead of the
// short package name. Pointer, slice, array, map, and channel types are walked so named parts
// inside them are fully qualified too. Anonymous structs, interfaces, funcs, and anything else
// fall back to reflect.Type.String.
func Qualified(typ reflect.Type) string {
	if typ.PkgPath() != "" {
		return typ.PkgPath() + "." + typ.Name()
	}

	//nolint:exhaustive // only kinds that can wrap a named type are qualified; others use reflect.Type.String
	switch typ.Kind() {
	case reflect.Pointer:
		return "*" + Qualified(typ.Elem())
	case reflect.Slice:
		return "[]" + Qualified(typ.Elem())
	case reflect.Array:
		return fmt.Sprintf("[%d]%s", typ.Len(), Qualified(typ.Elem()))
	case reflect.Map:
		return "map[" + Qualified(typ.Key()) + "]" + Qualified(typ.Elem())
	case reflect.Chan:
		var prefix string

		switch typ.ChanDir() {
		case reflect.RecvDir:
			prefix = "<-chan "
		case reflect.SendDir:
			prefix = "chan<- "
		case reflect.BothDir:
			prefix = "chan "
		}

		return prefix + Qualified(typ.Elem())
	}

	return typ.String()
}
