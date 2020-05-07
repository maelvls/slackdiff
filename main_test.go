package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_isInChans(t *testing.T) {
	tests := map[string]struct {
		all    []string
		sparse []string
		want   []string
	}{
		"should return a slice with true or false emojis": {
			all:    []string{"a", "b", "c", "d"},
			sparse: []string{"a", "c"},
			want:   []string{"✅", "❌", "✅", "❌"},
		},
		"if all slice is not a superset of sparse slice, should work anyway": {
			all:    []string{"a", "b"},
			sparse: []string{"a", "c"},
			want:   []string{"✅", "❌"},
		},
	}
	for ttname, tt := range tests {
		t.Run(ttname, func(t *testing.T) {
			assert.Equal(t, tt.want, isInChans(tt.all, tt.sparse, "✅", "❌"))
		})
	}
}
