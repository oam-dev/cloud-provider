package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAppStackFailed(t *testing.T) {
	tests := []struct {
		input map[string]string
		exp   bool
	}{
		{
			input: map[string]string{
				AppStackStatus: Failed,
			},
			exp: true,
		},
		{
			input: map[string]string{
				AppStackStatus: Ready,
			},
			exp: false,
		},
	}
	for _, ti := range tests {
		got := IsAppStackFailed(ti.input)
		assert.Equal(t, ti.exp, got)
	}
}
