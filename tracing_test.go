//nolint:testpackage // white-box: needs package-level EnableInjectionTracing
package dingo

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tracedDependency struct{}

type tracedTarget struct {
	Dependency *tracedDependency `inject:""`
}

// TestInjectionTracing_LogsFieldSetsAndResolutionsWhenEnabled checks R-27.
// With the switch on, each field set and each resolution logs one slog line.
// If the switch is only read and never used, debugging stays silent.
//
//nolint:paralleltest // swaps the process-wide slog default and the package-level tracing switch
func TestInjectionTracing_LogsFieldSetsAndResolutionsWhenEnabled(t *testing.T) {
	var buffer bytes.Buffer

	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buffer, nil)))

	t.Cleanup(func() {
		slog.SetDefault(previous)

		injectionTracing = false
	})

	EnableInjectionTracing()

	injector, err := NewInjector()
	require.NoError(t, err)

	_, err = injector.GetInstance(new(tracedTarget))
	require.NoError(t, err)

	assert.Contains(t, buffer.String(), "SETTING FIELD: Dependency")
	assert.Contains(t, buffer.String(), "INJECTING: flamingo.me/dingo#tracedTarget")
	assert.Contains(t, buffer.String(), "INJECTING: flamingo.me/dingo#tracedDependency")
}
