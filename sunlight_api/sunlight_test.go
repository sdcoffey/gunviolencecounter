package sunlight_api

import (
	"github.com/sdcoffey/gunviolencecounter/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"testing"
)

func TestGetReps(t *testing.T) {
	reps := GetReps("48430")
	assert.Len(t, 3, reps)
}
