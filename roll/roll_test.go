package roll

import (
	"log"
	"math/rand"
	"testing"
)

type fakeRolls struct {
	nums    []int
	counter int
}

func (fs fakeRolls) Seed(int64) {}
func (fs *fakeRolls) Int63() int64 {
	if fs.counter >= len(fs.nums) {
		log.Printf("fakeRolls called too many times. Got fake nums: %v, being called %v-th time", fs.nums, fs.counter+1)
		return 0
	}
	num := fs.nums[fs.counter]
	fs.counter++
	return int64(num) - 1
}

func fakeRand(nums []int) *rand.Rand {
	return rand.New(&fakeRolls{nums, 0})
}

func TestParseNotation(t *testing.T) {
	for _, tc := range []struct {
		name      string
		notation  string
		randRolls []int

		wantValue int
		wantRepr  string
	}{
		{
			name: "const", notation: "6", randRolls: []int{},
			wantValue: 6, wantRepr: "6",
		},
		{
			name: "one die", notation: "d4", randRolls: []int{3},
			wantValue: 3, wantRepr: "3",
		},
		{
			name: "die acing", notation: "d8", randRolls: []int{8, 5},
			wantValue: 13, wantRepr: "[8 + 5]",
		},
		{
			name: "multiple same dice choose best", notation: "2d10", randRolls: []int{6, 7},
			wantValue: 7, wantRepr: "[6, 7] 7",
		},
		{
			name: "different dice choose best", notation: "d6d4", randRolls: []int{1, 3},
			wantValue: 3, wantRepr: "[1, 3] 3",
		},
		{
			name: "different dice sum", notation: "d10 + d4", randRolls: []int{8, 2},
			wantValue: 10, wantRepr: "8 + 2",
		},
	} {
		rollNotation, err := ParseNotation(tc.notation)
		if err != nil {
			t.Fatalf("ParseNotation(%q) = _, err: %v", tc.notation, err)
		}
		rollNotation.SetRand(fakeRand(tc.randRolls))

		rollResult := rollNotation.Roll()
		if rollResult.Value() != tc.wantValue || rollResult.String() != tc.wantRepr {
			t.Fatalf("rollNotation.Roll(%q).SetRand(%v) = value: %v, repr: %q, want value: %v, want repr: %q", tc.notation, tc.randRolls, rollResult.Value(), rollResult.String(), tc.wantValue, tc.wantRepr)
		}
	}
}
