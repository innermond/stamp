package main

import (
	"reflect"
	"testing"
)

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

func TestPagesFromInput(t *testing.T) {
	tt := []struct {
		np        int
		p         string
		selection map[int]bool
		peaks     []int
	}{
		{3, "1,2,3",
			map[int]bool{1: true, 2: true, 3: true},
			[]int{1, 2, 3},
		},
		{10, "1-3,8",
			map[int]bool{1: true, 2: true, 3: true, 8: true},
			[]int{3, 8},
		},
		{3, "3-1,1-3,2,1,4",
			map[int]bool{1: true, 2: true, 3: true},
			[]int{3, 3, 2, 1, 3},
		},
		{3, "",
			map[int]bool{1: true, 2: true, 3: true},
			[]int{3},
		},
		{3, "2-",
			map[int]bool{2: true, 3: true},
			[]int{3},
		},
		{3, "-2",
			map[int]bool{1: true, 2: true},
			[]int{2},
		},
		{4, "-2,1,3-,2-3",
			map[int]bool{1: true, 2: true, 3: true, 4: true},
			[]int{2, 1, 4, 3},
		},
		{4, "2,5-6",
			map[int]bool{2: true},
			[]int{2, 4},
		},
	}

	for i, tc := range tt {
		selection, peaks, err := pagesFromInput(tc.p, tc.np)
		if err != nil {
			t.Errorf("unexpected error %v", err)
		}
		if reflect.DeepEqual(selection, tc.selection) == false {
			t.Errorf("selection: case %d wanted %v got %v", i, tc.selection, selection)
		}
		if reflect.DeepEqual(tc.peaks, peaks) == false {
			t.Errorf("peaks: case %d wanted %v got %v", i, tc.peaks, peaks)
		}
	}
}
