package algos

import (
	"fmt"
	"testing"
)

func expectWithMap(t *testing.T, pq *AdressablePQ[string, int],
	keyMap map[string]int,
	values []int) {
	t.Helper()

	if len(values) != len(pq.entries) {
		t.Errorf("want queue of length %v, got %v", len(values), len(pq.entries))
		return
	}

	if len(values) != len(pq.keyMap) {
		t.Errorf("want map of size %v, got %v", len(values), len(pq.keyMap))
		return
	}

	for idx, val := range values {
		if pq.entries[idx].value != val {
			t.Errorf("want entry[%v].value=%v, got %v", idx, val, pq.entries[idx].value)
		}
	}

	for key, val := range keyMap {
		if pq.keyMap[key] != val {
			t.Errorf("want keyMap[%v]=%v, got %v", key, val, pq.keyMap[key])
		}
		if pq.entries[val].key != key {
			t.Errorf("want entry[%v].key=%v, got %v", val, key, pq.entries[val].key)
		}
	}
}

func expectValues(t *testing.T, pq *AdressablePQ[string, int], values []int) {
	t.Helper()
	expectMap := make(map[string]int)
	for idx, val := range values {
		valStr := fmt.Sprintf("%v", val)
		expectMap[valStr] = idx
	}
	expectWithMap(t, pq, expectMap, values)
}

func setUp(t *testing.T, values []int) *AdressablePQ[string, int] {
	existMap := make(map[string]int)
	existingVals := make([]keyValue[string, int], len(values))
	for idx, val := range values {
		key := fmt.Sprintf("%v", val)
		existMap[key] = idx
		existingVals[idx] = keyValue[string, int]{key, val}
	}
	return &AdressablePQ[string, int]{
		comp:    OrderedComparator[int]{},
		entries: existingVals,
		keyMap:  existMap,
	}
}

func TestPush(t *testing.T) {
	tests := []struct {
		name     string
		existing []int
		toAdd    int
		want     []int
	}{
		{
			name:     "Add first",
			existing: []int{},
			toAdd:    4,
			want:     []int{4},
		},
		{
			name:     "Add larger moves to top",
			existing: []int{3, 1, 2},
			toAdd:    4,
			want:     []int{4, 3, 2, 1},
		},
		{
			name:     "Add smaller remains at end",
			existing: []int{5, 2, 3},
			toAdd:    1,
			want:     []int{5, 2, 3, 1},
		},
		{
			name:     "Add number moves one level",
			existing: []int{5, 3, 2},
			toAdd:    4,
			want:     []int{5, 4, 2, 3},
		},
		{
			name:     "Add existing key is ignored",
			existing: []int{5, 3, 2},
			toAdd:    3,
			want:     []int{5, 3, 2},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pq := setUp(t, tc.existing)
			pq.Push(fmt.Sprintf("%v", tc.toAdd), tc.toAdd)
			expectValues(t, pq, tc.want)
		})
	}
}

func TestRemove(t *testing.T) {
	tests := []struct {
		name     string
		existing []int
		toRemove string
		want     []int
	}{
		{
			name:     "Remove last elem",
			existing: []int{3},
			toRemove: "3",
			want:     []int{},
		},
		{
			name:     "Remove last index",
			existing: []int{3, 1, 2},
			toRemove: "2",
			want:     []int{3, 1},
		},
		{
			name:     "Remove head",
			existing: []int{5, 2, 3},
			toRemove: "5",
			want:     []int{3, 2},
		},
		{
			name:     "Remove head heapifies",
			existing: []int{5, 3, 2},
			toRemove: "5",
			want:     []int{3, 2},
		},
		{
			name: "Remove in middle, replacement stays",
			existing: []int{
				8,
				6, 3,
				1, 4, 5},
			toRemove: "6",
			want: []int{
				8,
				5, 3,
				1, 4},
		},
		{
			name: "Remove in middle, replacement heapifies down",
			existing: []int{
				8,
				6, 3,
				1, 4, 2},
			toRemove: "6",
			want: []int{
				8,
				4, 3,
				1, 2},
		},
		{
			name: "Remove in middle, replacement bubles up",
			existing: []int{
				10,
				6, 9,
				1, 4, 7},
			toRemove: "4",
			want: []int{
				10,
				7, 9,
				1, 6},
		},
		{
			name: "Remove prev to last, bubbles up",
			existing: []int{
				10,
				6, 9,
				1, 4, 7},
			toRemove: "4",
			want: []int{
				10,
				7, 9,
				1, 6},
		},
		{
			name:     "Remove inexistent is ignored",
			existing: []int{8, 6, 3, 1, 4, 2},
			toRemove: "5",
			want:     []int{8, 6, 3, 1, 4, 2},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pq := setUp(t, tc.existing)
			pq.Remove(tc.toRemove)
			expectValues(t, pq, tc.want)
		})
	}
}

func TestPop(t *testing.T) {
	tests := []struct {
		name     string
		existing []int
		want     []int
		wantRes  int
	}{
		{
			name:     "Pop last",
			existing: []int{3},
			wantRes:  3,
			want:     []int{},
		},
		{
			name:     "Last replaces directly",
			existing: []int{5, 2, 3},
			want:     []int{3, 2},
			wantRes:  5,
		},
		{
			name:     "Replacement head heapifies down",
			existing: []int{5, 3, 2},
			want:     []int{3, 2},
			wantRes:  5,
		},
		{
			name:     "Empty heap returns default",
			existing: []int{},
			want:     []int{},
			wantRes:  0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pq := setUp(t, tc.existing)
			got := pq.Pop()
			expectValues(t, pq, tc.want)
			if got != tc.wantRes {
				t.Errorf("want %v, got %v", tc.wantRes, got)
			}
		})
	}
}

func TestEmplace(t *testing.T) {
	tests := []struct {
		name         string
		existing     []int
		wantValues   []int
		wantMap      map[string]int
		emplaceKey   string
		emplaceValue int
	}{
		{
			name:         "Last replaced with lower stays",
			existing:     []int{5, 2, 3},
			emplaceKey:   "3",
			emplaceValue: 1,
			wantValues:   []int{5, 2, 1},
			wantMap: map[string]int{
				"5": 0, "2": 1, "3": 2,
			},
		},
		{
			name:         "Last replaced with higher, bubbles up",
			existing:     []int{5, 2, 3},
			emplaceKey:   "3",
			emplaceValue: 10,
			wantValues:   []int{10, 2, 5},
			wantMap: map[string]int{
				"3": 0, "2": 1, "5": 2,
			},
		},
		{
			name:         "Head replaced with smaller, heapifies down",
			existing:     []int{5, 2, 3},
			emplaceKey:   "5",
			emplaceValue: 1,
			wantValues:   []int{3, 2, 1},
			wantMap: map[string]int{
				"3": 0, "2": 1, "5": 2,
			},
		},
		{
			name: "Emplace in middle, heapifies down",
			existing: []int{
				8,
				6, 3,
				1, 4, 2},
			emplaceKey:   "6",
			emplaceValue: 2,
			wantValues: []int{
				8,
				4, 3,
				1, 2, 2},
			wantMap: map[string]int{
				"8": 0, "4": 1, "3": 2, "1": 3, "6": 4, "2": 5,
			},
		},
		{
			name: "Emplace in middle, bubles up",
			existing: []int{
				10,
				6, 9,
				1, 4, 7},
			emplaceKey:   "6",
			emplaceValue: 12,
			wantValues: []int{
				12,
				10, 9,
				1, 4, 7},
			wantMap: map[string]int{
				"6": 0, "10": 1, "9": 2, "1": 3, "4": 4, "7": 5,
			},
		},
		{
			name: "Emplace in middle stays",
			existing: []int{
				10,
				6, 9,
				1, 4, 7},
			emplaceKey:   "6",
			emplaceValue: 8,
			wantValues: []int{
				10,
				8, 9,
				1, 4, 7},
			wantMap: map[string]int{
				"10": 0, "6": 1, "9": 2, "1": 3, "4": 4, "7": 5,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pq := setUp(t, tc.existing)
			pq.Emplace(tc.emplaceKey, tc.emplaceValue)
			expectWithMap(t, pq, tc.wantMap, tc.wantValues)
		})
	}
}

func TestTop(t *testing.T) {
	tests := []struct {
		name     string
		existing []int
		want     int
	}{
		{
			name:     "Top with existing",
			existing: []int{5, 2, 3},
			want:     5,
		},
		{
			name:     "Top empty returns default",
			existing: []int{},
			want:     0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pq := setUp(t, tc.existing)
			got := pq.Top()
			if got != tc.want {
				t.Errorf("want %v, got %v", tc.want, got)
			}
		})
	}
}
