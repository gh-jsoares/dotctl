package plugin

import (
	"testing"
)

func TestResolveOrder_Empty(t *testing.T) {
	result, err := ResolveOrder(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}

func TestResolveOrder_SinglePlugin(t *testing.T) {
	plugins := []*Plugin{
		{Name: "foo", Ordering: Ordering{After: []string{"stow"}}},
	}
	result, err := ResolveOrder(plugins)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 || result[0].Name != "foo" {
		t.Fatalf("expected [foo], got %v", result)
	}
}

func TestResolveOrder_PriorityBreaksTies(t *testing.T) {
	plugins := []*Plugin{
		{Name: "beta", Ordering: Ordering{After: []string{"stow"}, Priority: 50}},
		{Name: "alpha", Ordering: Ordering{After: []string{"stow"}, Priority: 10}},
	}
	result, err := ResolveOrder(plugins)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(result))
	}
	if result[0].Name != "alpha" {
		t.Fatalf("expected alpha first (priority 10), got %s", result[0].Name)
	}
	if result[1].Name != "beta" {
		t.Fatalf("expected beta second (priority 50), got %s", result[1].Name)
	}
}

func TestResolveOrder_DependencyChain(t *testing.T) {
	plugins := []*Plugin{
		{Name: "second", Ordering: Ordering{After: []string{"first"}}},
		{Name: "first", Ordering: Ordering{After: []string{"stow"}}},
	}
	result, err := ResolveOrder(plugins)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2, got %d", len(result))
	}
	if result[0].Name != "first" || result[1].Name != "second" {
		t.Fatalf("expected [first, second], got [%s, %s]", result[0].Name, result[1].Name)
	}
}

func TestResolveOrder_CircularDependency(t *testing.T) {
	plugins := []*Plugin{
		{Name: "a", Ordering: Ordering{After: []string{"b"}}},
		{Name: "b", Ordering: Ordering{After: []string{"a"}}},
	}
	_, err := ResolveOrder(plugins)
	if err == nil {
		t.Fatal("expected circular dependency error")
	}
}

func TestResolveOrder_ConflictsWithCoreID(t *testing.T) {
	plugins := []*Plugin{
		{Name: "stow", Ordering: Ordering{After: []string{"mise"}}},
	}
	_, err := ResolveOrder(plugins)
	if err == nil {
		t.Fatal("expected conflict with core step ID error")
	}
}

func TestResolveOrder_UnknownDependency(t *testing.T) {
	plugins := []*Plugin{
		{Name: "foo", Ordering: Ordering{After: []string{"nonexistent"}}},
	}
	_, err := ResolveOrder(plugins)
	if err == nil {
		t.Fatal("expected unknown dependency error")
	}
}

func TestResolveOrder_DefaultAfterMise(t *testing.T) {
	plugins := []*Plugin{
		{Name: "foo", Ordering: Ordering{}},
	}
	result, err := ResolveOrder(plugins)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 || result[0].Name != "foo" {
		t.Fatalf("expected [foo], got %v", result)
	}
}

func TestResolveOrder_BeforeTarget(t *testing.T) {
	plugins := []*Plugin{
		{Name: "late", Ordering: Ordering{After: []string{"stow"}}},
		{Name: "early", Ordering: Ordering{After: []string{"stow"}, Before: []string{"late"}}},
	}
	result, err := ResolveOrder(plugins)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result[0].Name != "early" || result[1].Name != "late" {
		t.Fatalf("expected [early, late], got [%s, %s]", result[0].Name, result[1].Name)
	}
}
