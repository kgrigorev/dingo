// Package typename prints reflect types with their full import path.
//
// It is shared by the root injector (module graph diagnostics) and the typed API (bind-time
// messages), so that both name a type the same way. It is internal: Go's internal rule is
// import-path based, so flamingo.me/dingo/v2 may import it while nothing outside this repository
// can.
package typename

import (
	"fmt"
	"reflect"
)

// Qualified is like reflect.Type.String but uses the full import path instead of the short
// package name for named types. It handles common composite types recursively (pointer, slice,
// array, map, channel) so that any named element or key type inside them is also fully
// qualified. Anonymous composite types (struct, interface, func) fall back to
// reflect.Type.String, as does anything else not covered above.
func Qualified(typ reflect.Type) string {
	if typ.PkgPath() != "" {
		return typ.PkgPath() + "." + typ.Name()
	}

	//nolint:exhaustive // only kinds that can wrap a named type are qualified, everything else falls back to reflect.Type.String
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
