package plugin

import "fmt"

var CoreStepIDs = []string{
	"git-pull",
	"submodule-update",
	"nix-darwin",
	"commit-flake-lock",
	"stow",
	"sheldon",
	"mise",
}

func ResolveOrder(plugins []*Plugin) ([]*Plugin, error) {
	if len(plugins) == 0 {
		return nil, nil
	}

	allIDs := make(map[string]bool)
	for _, id := range CoreStepIDs {
		allIDs[id] = true
	}
	for _, p := range plugins {
		if allIDs[p.Name] {
			return nil, fmt.Errorf("plugin %q conflicts with core step ID", p.Name)
		}
		allIDs[p.Name] = true
	}

	graph := make(map[string][]string)
	inDegree := make(map[string]int)
	pluginMap := make(map[string]*Plugin)

	for _, p := range plugins {
		pluginMap[p.Name] = p
		inDegree[p.Name] = 0
	}

	for _, p := range plugins {
		after := p.Ordering.After
		if len(after) == 0 {
			after = []string{"mise"}
		}

		for _, dep := range after {
			if !allIDs[dep] {
				return nil, fmt.Errorf("plugin %q: unknown dependency %q", p.Name, dep)
			}
			if pluginMap[dep] != nil {
				graph[dep] = append(graph[dep], p.Name)
				inDegree[p.Name]++
			}
		}

		for _, target := range p.Ordering.Before {
			if !allIDs[target] {
				return nil, fmt.Errorf("plugin %q: unknown before-target %q", p.Name, target)
			}
			if pluginMap[target] != nil {
				graph[p.Name] = append(graph[p.Name], target)
				inDegree[target]++
			}
		}
	}

	// Kahn's algorithm
	var queue []string
	for _, p := range plugins {
		if inDegree[p.Name] == 0 {
			queue = append(queue, p.Name)
		}
	}

	var sorted []*Plugin
	for len(queue) > 0 {
		// Pick lowest priority from queue
		best := 0
		for i, name := range queue {
			if pluginMap[name].Ordering.Priority < pluginMap[queue[best]].Ordering.Priority {
				best = i
			}
		}
		current := queue[best]
		queue = append(queue[:best], queue[best+1:]...)

		sorted = append(sorted, pluginMap[current])

		for _, next := range graph[current] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	if len(sorted) != len(plugins) {
		return nil, fmt.Errorf("circular dependency detected among plugins")
	}

	return sorted, nil
}
