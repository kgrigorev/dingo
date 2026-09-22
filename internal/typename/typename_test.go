package typename_test

import (
	"reflect"
	"testing"

	"flamingo.me/dingo/internal/typename"
	"github.com/stretchr/testify/assert"
)

type named struct{}

type namedIface interface{ M() }

// TestQualified_PrintsFullImportPaths pins the printer shared by the root injector's module
// diagnostics and the typed API's bind-time messages.
// Catches: a printer that falls back to reflect.Type.String for named types, which would make two
// same-named types from different packages indistinguishable in an error message.
func TestQualified_PrintsFullImportPaths(t *testing.T) {
	t.Parallel()

	const pkg = "flamingo.me/dingo/internal/typename_test."

	tests := []struct {
		name string
		typ  reflect.Type
		want string
	}{
		{name: "named struct", typ: reflect.TypeFor[named](), want: pkg + "named"},
		{name: "named interface", typ: reflect.TypeFor[namedIface](), want: pkg + "namedIface"},
		{name: "pointer to named", typ: reflect.TypeFor[*named](), want: "*" + pkg + "named"},
		{name: "pointer to pointer", typ: reflect.TypeFor[**named](), want: "**" + pkg + "named"},
		{name: "slice of interface", typ: reflect.TypeFor[[]namedIface](), want: "[]" + pkg + "namedIface"},
		{name: "array", typ: reflect.TypeFor[[3]named](), want: "[3]" + pkg + "named"},
		{name: "map", typ: reflect.TypeFor[map[string]named](), want: "map[string]" + pkg + "named"},
		{name: "receive channel", typ: reflect.TypeFor[<-chan named](), want: "<-chan " + pkg + "named"},
		{name: "send channel", typ: reflect.TypeFor[chan<- named](), want: "chan<- " + pkg + "named"},
		{name: "bidirectional channel", typ: reflect.TypeFor[chan named](), want: "chan " + pkg + "named"},
		{name: "basic type", typ: reflect.TypeFor[string](), want: "string"},
		{name: "anonymous struct falls back", typ: reflect.TypeFor[struct{ A int }](), want: "struct { A int }"},
		{name: "func falls back", typ: reflect.TypeFor[func() int](), want: "func() int"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, typename.Qualified(tt.typ))
		})
	}
}
