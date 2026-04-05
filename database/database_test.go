package database

import "testing"

func TestPaginate_FirstPage(t *testing.T) {
	items := []string{"a", "b", "c", "d", "e"}
	result := Paginate(items, 1, 2)

	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
	if result.Items[0] != "a" || result.Items[1] != "b" {
		t.Errorf("expected [a b], got %v", result.Items)
	}
	if result.TotalCount != 5 {
		t.Errorf("expected total 5, got %d", result.TotalCount)
	}
	if result.Page != 1 {
		t.Errorf("expected page 1, got %d", result.Page)
	}
}

func TestPaginate_SecondPage(t *testing.T) {
	items := []string{"a", "b", "c", "d", "e"}
	result := Paginate(items, 2, 2)

	if len(result.Items) != 2 || result.Items[0] != "c" {
		t.Errorf("expected [c d], got %v", result.Items)
	}
}

func TestPaginate_LastPage(t *testing.T) {
	items := []string{"a", "b", "c", "d", "e"}
	result := Paginate(items, 3, 2)

	if len(result.Items) != 1 || result.Items[0] != "e" {
		t.Errorf("expected [e], got %v", result.Items)
	}
}

func TestPaginate_BeyondEnd(t *testing.T) {
	items := []string{"a", "b"}
	result := Paginate(items, 5, 10)

	if result.Items != nil {
		t.Errorf("expected nil, got %v", result.Items)
	}
	if result.TotalCount != 2 {
		t.Errorf("expected total 2, got %d", result.TotalCount)
	}
}

func TestPaginate_Defaults(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	result := Paginate(items, 0, 0) // invalid -> page=1, limit=10

	if len(result.Items) != 10 {
		t.Errorf("expected 10 items with default limit, got %d", len(result.Items))
	}
}

func TestPaginate_Empty(t *testing.T) {
	result := Paginate([]int{}, 1, 10)
	if result.Items != nil {
		t.Errorf("expected nil for empty input, got %v", result.Items)
	}
	if result.TotalCount != 0 {
		t.Errorf("expected 0, got %d", result.TotalCount)
	}
}
