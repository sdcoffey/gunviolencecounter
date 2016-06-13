package sunlight_api

import (
	"fmt"
	"testing"
)

func TestGetReps(t *testing.T) {
	reps := GetReps("48430")
	fmt.Println(reps)
}
