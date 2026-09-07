package artifactlimits

import "testing"

func TestBudgetBoundaries(t *testing.T) {
	for _, test := range []struct {
		name    string
		budget  Budget
		size    uint64
		allowed bool
	}{
		{"entry at limit", Budget{}, MaxEntryBytes, true},
		{"entry above limit", Budget{}, MaxEntryBytes + 1, false},
		{"total at limit", Budget{Bytes: MaxTotalBytes - 1}, 1, true},
		{"total above limit", Budget{Bytes: MaxTotalBytes}, 1, false},
		{"last file", Budget{Files: MaxFiles - 1}, 0, true},
		{"too many files", Budget{Files: MaxFiles}, 0, false},
		{"size overflow", Budget{}, ^uint64(0), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			before := test.budget
			err := test.budget.Add("asset", test.size)
			if (err == nil) != test.allowed {
				t.Fatalf("Add() = %v, allowed=%t", err, test.allowed)
			}
			if err != nil && test.budget != before {
				t.Fatal("rejected entry consumed the budget")
			}
		})
	}
}

func TestBudgetBoundsCumulativeEntryWrites(t *testing.T) {
	budget := Budget{}
	if err := budget.Add("asset", MaxEntryBytes-1); err != nil {
		t.Fatal(err)
	}
	if err := budget.Grow("asset", MaxEntryBytes-1, 1); err != nil {
		t.Fatal(err)
	}
	if err := budget.Grow("asset", MaxEntryBytes, 1); err == nil {
		t.Fatal("entry grew beyond its limit")
	}
}

func TestCompressionBudget(t *testing.T) {
	for _, test := range []struct {
		plain, compressed uint64
		allowed           bool
	}{{0, 0, true}, {1, 0, false}, {MaxExpandRatio, 1, true}, {MaxExpandRatio + 1, 1, false}} {
		if err := CheckCompression("asset", test.plain, test.compressed); (err == nil) != test.allowed {
			t.Fatalf("CheckCompression(%d, %d) = %v", test.plain, test.compressed, err)
		}
	}
}
