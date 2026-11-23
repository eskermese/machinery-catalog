package entity

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPointer(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		v    interface{}
		exp  interface{}
	}{
		{
			name: "test",
			v:    "test",
			exp:  "test",
		},
	}

	for _, s := range cases {
		s := s

		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			actual := Pointer(s.v)

			require.Equal(t, s.exp, *actual)
		})
	}
}
