package utils

import (
	"testing"
	"math/rand"
)


// Generate a array filled with unique ids, and a shuffled copy
func BuildShuffledSlices(length int) (initial, shuffled []int) {


	s1 := make([]int, length, length)
	s2 := make([]int, length, length) // This one will be shuffled later
	
	// two initial equal slices filled with unique random values
	for i := 0; i < length; i++ {
		s1[i] = rand.Int()
		s2[i] = s1[i]
	}

	// shuffle one of the slices
	Shuffle(s2, NewRandSource())
	return s1, s2
}


func EqualSliceContents(s1, s2 []int) bool {
	

	// Count ocurrences of each values in s1
	hits := make(map[int]int)
	for i:= 0; i<len(s1); i++ {
		if count, ok := hits[s1[i]]; ok {
			hits[s1[i]] = count + 1
		} else {
			hits[s1[i]] = 1
		}
	}

	// Compare s2 against s1
	for i:=0; i<len(s2); i++ {
		if count, ok := hits[s2[i]]; ok && count > 0{
			hits[s2[i]] = count - 1
		} else {
			return false
		}
	}

	return true
}


func IsSliceShuffled(s1, s2 []int) bool {
	for i, value := range s1 {
		if value != s2[i] {
			return true
		}
	}

	return false
}

func TestShuffle(t *testing.T) {


	for l:=0; l<100; l++ {
		initial, shuffled := BuildShuffledSlices(l)
		if len(initial) != len(shuffled) {
			t.Errorf("Shuffled changed slice lenghts %v %v", len(initial), len(shuffled))
			return
		}

		// check all elements are present
		if !EqualSliceContents(initial, shuffled) {
			t.Errorf("Slice contents aren't equal")
			return
		}

		// Check shuffled
		if l > 10 && !IsSliceShuffled(initial, shuffled) {
			t.Errorf("The slice wasn't shuffled")
		}
	}
}
