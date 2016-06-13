package sunlight_api

import (
    "testing"
    "fmt"
)

func TestGetReps(t *testing.T) {
    reps := GetReps("48430")
    fmt.Println(reps)
}
