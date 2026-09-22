// Package typename prints a reflect type with its full import path.
//
// Root module errors and typed-API bind errors both use this, so names match.
// Only this repository can import it (internal/).
package typename

import (
	"fmt"
	"reflect"
)

// Qualified is like reflect.Type.String, but named types show the full import path.
// It also walks pointers, slices, arrays, maps, and channels.
// Struct, interface, and func types without a name fall back to Type.String.
func Qualified(typ reflect.Type) string {
	if typ.PkgPath() != "" {
		return typ.PkgPath() + "." + typ.Name()
	}

	//nolint:exhaustive // only kinds that wrap a named type get a full path; others use Type.String
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
