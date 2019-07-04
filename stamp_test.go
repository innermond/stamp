package main

import "testing"

func TestWhereIBelong(t *testing.T) {
	tt := []struct {
		i      int
		limits []int
		pos    int
	}{
		{1, []int{3, 1}, 1},
		{2, []int{3, 1}, 0},
		{3, []int{3, 1}, 0},
		{1, []int{1, 3}, 0},
		{2, []int{1, 3}, 1},
		{3, []int{1, 3}, 1},
	}

	for i, tc := range tt {
		got := whereIBelong(tc.i, tc.limits)
		if got != tc.pos {
			t.Errorf("case %d wanted %d got %d", i, tc.pos, got)
		}
	}
}
