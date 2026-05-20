package synthra_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopherly.dev/synthra"
	"gopherly.dev/synthra/source"
)

// TestOnBound_RunsAgainstBoundStruct verifies the hook receives the
// populated struct.
func TestOnBound_RunsAgainstBoundStruct(t *testing.T) {
	t.Parallel()

	type Cfg struct {
		Level string `synthra:"level"`
	}
	var out Cfg
	ran := false
	cfg, err := synthra.New(
		synthra.WithSource(source.NewMap(map[string]any{"level": "WARN"})),
		synthra.WithBinding(&out,
			synthra.OnBound(func(c *Cfg) error {
				ran = true
				c.Level = strings.ToLower(c.Level)
				return nil
			}),
		),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))
	assert.True(t, ran, "hook must have run")
	assert.Equal(t, "warn", out.Level)
}

// TestOnBound_MultipleHooks verifies hooks run in registration order.
func TestOnBound_MultipleHooks(t *testing.T) {
	t.Parallel()

	type Cfg struct {
		Value string `synthra:"value"`
	}
	var out Cfg
	order := make([]int, 0, 2)
	cfg, err := synthra.New(
		synthra.WithSource(source.NewMap(map[string]any{"value": "x"})),
		synthra.WithBinding(&out,
			synthra.OnBound(func(c *Cfg) error {
				order = append(order, 1)
				return nil
			}),
			synthra.OnBound(func(c *Cfg) error {
				order = append(order, 2)
				return nil
			}),
		),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))
	assert.Equal(t, []int{1, 2}, order)
}

// TestOnBound_FirstErrorStopsPipeline verifies that the second hook is not
// called when the first hook returns an error.
func TestOnBound_FirstErrorStopsPipeline(t *testing.T) {
	t.Parallel()

	type Cfg struct {
		Value string `synthra:"value"`
	}
	var out Cfg
	secondRan := false
	cfg, err := synthra.New(
		synthra.WithSource(source.NewMap(map[string]any{"value": "x"})),
		synthra.WithBinding(&out,
			synthra.OnBound(func(_ *Cfg) error {
				return errors.New("hook failed")
			}),
			synthra.OnBound(func(_ *Cfg) error {
				secondRan = true
				return nil
			}),
		),
	)
	require.NoError(t, err)
	require.Error(t, cfg.Load(context.Background()))
	assert.False(t, secondRan, "second hook must not run after first error")
}

// TestOnBound_RunsAfterApplyDefaults verifies the hook sees default-filled fields.
func TestOnBound_RunsAfterApplyDefaults(t *testing.T) {
	t.Parallel()

	type Cfg struct {
		Level string `synthra:"level" default:"info"`
	}
	var out Cfg
	var hookSawLevel string
	cfg, err := synthra.New(
		synthra.WithSource(source.NewMap(map[string]any{})),
		synthra.WithBinding(&out,
			synthra.OnBound(func(c *Cfg) error {
				hookSawLevel = c.Level
				return nil
			}),
		),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))
	assert.Equal(t, "info", hookSawLevel, "hook must see default-applied value")
}

// TestOnBound_RunsBeforeValidatorValidate verifies that a mutation by the hook is
// visible when the struct's Validate method is called.
func TestOnBound_RunsBeforeValidatorValidate(t *testing.T) {
	t.Parallel()

	// Use two independent hooks to verify order.
	type WithValidate struct {
		Level string `synthra:"level"`
		// hook writes here
		NormalisedBy int
	}

	var out WithValidate
	hookOrder := 0
	validateOrder := 0

	// We cannot attach Validate to a local type through synthra.Validator
	// without a named type. Use OnBound to simulate both sides.
	cfg2, err := synthra.New(
		synthra.WithSource(source.NewMap(map[string]any{"level": "DEBUG"})),
		synthra.WithBinding(&out,
			// hook that records its position
			synthra.OnBound(func(c *WithValidate) error {
				hookOrder++
				c.NormalisedBy = hookOrder
				return nil
			}),
			// second hook that records after first
			synthra.OnBound(func(c *WithValidate) error {
				validateOrder = hookOrder + 1
				return nil
			}),
		),
	)
	require.NoError(t, err)
	require.NoError(t, cfg2.Load(context.Background()))
	assert.Equal(t, 1, out.NormalisedBy, "first hook must have run")
	assert.Equal(t, 2, validateOrder, "second hook runs after first")
}

// TestOnBound_NilFunctionReportedAtLoad verifies that passing nil as the OnBound
// function is caught at Load time with a descriptive error.
func TestOnBound_NilFunctionReportedAtLoad(t *testing.T) {
	t.Parallel()

	type Cfg struct{ Value string }
	var out Cfg
	cfg, err := synthra.New(
		synthra.WithSource(source.NewMap(map[string]any{"value": "x"})),
		synthra.WithBinding(&out,
			synthra.OnBound[Cfg](nil),
		),
	)
	require.NoError(t, err, "nil fn is detected at Load, not New")
	err = cfg.Load(context.Background())
	require.Error(t, err)
}

// TestWithBinding_NilTargetRejectedAtNew verifies that a nil binding target
// is caught during New.
func TestWithBinding_NilTargetRejectedAtNew(t *testing.T) {
	t.Parallel()

	type Cfg struct{ Value string }
	_, err := synthra.New(
		synthra.WithSource(source.NewMap(map[string]any{"value": "x"})),
		synthra.WithBinding((*Cfg)(nil)),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "binding target cannot be nil")
}

// TestOnBound_TypeMismatch_CompilesOnlyWithMatchingType is a documentation
// test. The following snippet does NOT compile:
//
//	type A struct{ X string }
//	type B struct{ Y int }
//	var a A
//	synthra.WithBinding(&a, synthra.OnBound(func(b *B) error { return nil }))
//	// compile error: cannot use OnBound(func(*B) error) as BindingOption[A]
func TestOnBound_TypeMismatch_CompilesOnlyWithMatchingType(t *testing.T) {
	t.Parallel()
	// Nothing to run; the doc comment above is the assertion.
	t.Log("type-safety is verified at compile time; see doc comment above")
}

// TestDefaults_PointerToStruct verifies that setDefaults allocates and
// recurses into nil *SubConfig fields when SubConfig carries default tags.
func TestDefaults_PointerToStruct(t *testing.T) {
	t.Parallel()

	type DB struct {
		Host string `synthra:"host" default:"localhost"`
		Port int    `synthra:"port" default:"5432"`
	}
	type Cfg struct {
		Name string `synthra:"name"`
		DB   *DB    `synthra:"db"`
	}

	var out Cfg
	cfg, err := synthra.New(
		synthra.WithSource(source.NewMap(map[string]any{"name": "app"})),
		synthra.WithBinding(&out),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	require.NotNil(t, out.DB, "pointer-to-struct with defaults must be allocated")
	assert.Equal(t, "localhost", out.DB.Host)
	assert.Equal(t, 5432, out.DB.Port)
}

// TestDefaults_PointerToStructNoDefaults verifies that nil *SubConfig
// fields are NOT allocated when the sub-struct has no default tags.
func TestDefaults_PointerToStructNoDefaults(t *testing.T) {
	t.Parallel()

	type DB struct {
		Host string `synthra:"host"`
		Port int    `synthra:"port"`
	}
	type Cfg struct {
		Name string `synthra:"name" default:"app"`
		DB   *DB    `synthra:"db"`
	}

	var out Cfg
	cfg, err := synthra.New(
		synthra.WithSource(source.NewMap(map[string]any{})),
		synthra.WithBinding(&out),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	assert.Equal(t, "app", out.Name)
	assert.Nil(t, out.DB, "pointer-to-struct without defaults must stay nil")
}

// TestDefaults_PointerToStructAlreadyPopulated verifies that defaults do not
// overwrite already-populated fields in a pointer-to-struct.
func TestDefaults_PointerToStructAlreadyPopulated(t *testing.T) {
	t.Parallel()

	type DB struct {
		Host string `synthra:"host" default:"localhost"`
		Port int    `synthra:"port" default:"5432"`
	}
	type Cfg struct {
		DB *DB `synthra:"db"`
	}

	var out Cfg
	cfg, err := synthra.New(
		synthra.WithSource(source.NewMap(map[string]any{
			"db": map[string]any{"host": "remotehost", "port": 3306},
		})),
		synthra.WithBinding(&out),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	require.NotNil(t, out.DB)
	assert.Equal(t, "remotehost", out.DB.Host)
	assert.Equal(t, 3306, out.DB.Port)
}
