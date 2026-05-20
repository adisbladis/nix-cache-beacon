package index

import (
	"slices"
	"testing"
)

func TestNewCacheIndex(t *testing.T) {
	ci := NewCacheIndex()
	if ci == nil {
		t.Fatal("NewCacheIndex returned nil")
	}

	count := len(slices.Collect(ci.Iter()))
	if count != 0 {
		t.Errorf("expected 0 groups in empty index, got %d", count)
	}
}

func TestAdd_Single(t *testing.T) {
	ci := NewCacheIndex()
	ci.Add(NewBinaryCache("http://a", 5))

	var got [][]*BinaryCache
	for group := range ci.Iter() {
		got = append(got, group)
	}
	if len(got) != 1 || len(got[0]) != 1 {
		t.Fatalf("expected 1 group with 1 cache, got %v", got)
	}
	if got[0][0].URL != "http://a" {
		t.Errorf("unexpected URL: %q", got[0][0].URL)
	}
}

func TestAdd_Duplicate(t *testing.T) {
	ci := NewCacheIndex()
	for range 2 {
		ci.Add(NewBinaryCache("http://a", 5))
	}

	count := 0
	for group := range ci.Iter() {
		count += len(group)
	}
	if count != 1 {
		t.Errorf("duplicate add should be ignored; got %d caches", count)
	}
}

func TestAdd_MultiplePriorities_Order(t *testing.T) {
	ci := NewCacheIndex()
	ci.Add(NewBinaryCache("http://c", 30))
	ci.Add(NewBinaryCache("http://a", 10))
	ci.Add(NewBinaryCache("http://b", 20))

	var prios []int
	for group := range ci.Iter() {
		prios = append(prios, group[0].Priority)
	}
	for i := 1; i < len(prios); i++ {
		if prios[i] < prios[i-1] {
			t.Errorf("priorities not sorted ascending: %v", prios)
		}
	}
}

func TestAdd_SamePriority_MultipleURLs(t *testing.T) {
	ci := NewCacheIndex()
	ci.Add(NewBinaryCache("http://a", 5))
	ci.Add(NewBinaryCache("http://b", 5))

	var groups [][]*BinaryCache
	for group := range ci.Iter() {
		groups = append(groups, group)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 priority group, got %d", len(groups))
	}
	if len(groups[0]) != 2 {
		t.Errorf("expected 2 caches in group, got %d", len(groups[0]))
	}
}

func TestRemove_ExistingURL(t *testing.T) {
	ci := NewCacheIndex()
	ci.Add(NewBinaryCache("http://a", 5))
	ci.Add(NewBinaryCache("http://b", 5))
	ci.Remove("http://a")

	count := 0
	for group := range ci.Iter() {
		for _, c := range group {
			if c.URL == "http://a" {
				t.Error("removed cache still present")
			}
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 remaining cache, got %d", count)
	}
}

func TestRemove_EmptyPrio_NoGroup(t *testing.T) {
	ci := NewCacheIndex()
	ci.Add(NewBinaryCache("http://a", 5))
	ci.Remove("http://a")

	count := len(slices.Collect(ci.Iter()))
	if count != 0 {
		t.Errorf("expected empty index after removing sole cache, got %d groups", count)
	}
}

func TestRemove_NonExistentURL(t *testing.T) {
	ci := NewCacheIndex()
	ci.Add(NewBinaryCache("http://a", 5))
	ci.Remove("http://does-not-exist")

	count := 0
	for group := range ci.Iter() {
		count += len(group)
	}
	if count != 1 {
		t.Errorf("expected 1 cache after no-op remove, got %d", count)
	}
}

func TestRemove_EmptyIndex(t *testing.T) {
	ci := NewCacheIndex()
	ci.Remove("http://does-not-exist")
}
