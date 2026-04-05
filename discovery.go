package gonest

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
)

// DiscoveryService discovers providers, controllers, and metadata at runtime.
// Equivalent to NestJS DiscoveryService.
type DiscoveryService struct {
	mu          sync.RWMutex
	modules     []*Module
	reflector   *Reflector
}

// NewDiscoveryService creates a new discovery service.
func NewDiscoveryService(reflector *Reflector) *DiscoveryService {
	return &DiscoveryService{
		reflector: reflector,
	}
}

// SetModules populates the discovery service with the application module tree.
func (ds *DiscoveryService) SetModules(modules []*Module) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.modules = modules
}

// DiscoveredProvider wraps a provider instance with its host module.
type DiscoveredProvider struct {
	Instance any
	Type     reflect.Type
	Module   *Module
}

// DiscoveredController wraps a controller instance with its host module.
type DiscoveredController struct {
	Instance Controller
	Module   *Module
}

// GetProviders returns all resolved provider instances across all modules.
func (ds *DiscoveryService) GetProviders() []DiscoveredProvider {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	var result []DiscoveredProvider
	seen := make(map[reflect.Type]bool)

	for _, mod := range ds.modules {
		if mod.container == nil {
			continue
		}
		for t, entry := range mod.container.GetAllProviders() {
			if seen[t] {
				continue
			}
			seen[t] = true
			if entry.resolved {
				result = append(result, DiscoveredProvider{
					Instance: entry.instance,
					Type:     t,
					Module:   mod,
				})
			}
		}
	}
	return result
}

// GetControllers returns all registered controllers across all modules.
func (ds *DiscoveryService) GetControllers() []DiscoveredController {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	var result []DiscoveredController
	for _, mod := range ds.modules {
		for _, ctrl := range mod.controllers {
			result = append(result, DiscoveredController{
				Instance: ctrl,
				Module:   mod,
			})
		}
	}
	return result
}

// GetProvidersWithInterface returns all providers that implement the given interface.
func GetProvidersWithInterface[T any](ds *DiscoveryService) []T {
	ifaceType := reflect.TypeOf((*T)(nil)).Elem()
	providers := ds.GetProviders()

	var result []T
	for _, p := range providers {
		if p.Type.Implements(ifaceType) || reflect.PointerTo(p.Type).Implements(ifaceType) {
			if typed, ok := p.Instance.(T); ok {
				result = append(result, typed)
			}
		}
	}
	return result
}

// GraphInspector provides introspection into the application's dependency graph.
// Equivalent to NestJS GraphInspector.
type GraphInspector struct {
	modules []*Module
}

// NewGraphInspector creates a new graph inspector.
func NewGraphInspector() *GraphInspector {
	return &GraphInspector{}
}

// SetModules populates the graph inspector with the module tree.
func (gi *GraphInspector) SetModules(modules []*Module) {
	gi.modules = modules
}

// DependencyEdge represents a dependency relationship.
type DependencyEdge struct {
	Source reflect.Type // the dependent type
	Target reflect.Type // the dependency type
}

// ModuleNode represents a module in the dependency graph.
type ModuleNode struct {
	Module  *Module
	Imports []*Module
}

// GetModules returns all module nodes in the graph.
func (gi *GraphInspector) GetModules() []ModuleNode {
	var nodes []ModuleNode
	for _, mod := range gi.modules {
		nodes = append(nodes, ModuleNode{
			Module:  mod,
			Imports: mod.options.Imports,
		})
	}
	return nodes
}

// GetDependencies returns all dependency edges for providers in a module.
func (gi *GraphInspector) GetDependencies(mod *Module) []DependencyEdge {
	if mod.container == nil {
		return nil
	}

	var edges []DependencyEdge
	for _, entry := range mod.container.GetAllProviders() {
		p := entry.provider
		if p.Constructor == nil {
			continue
		}
		ct := reflect.TypeOf(p.Constructor)
		for i := 0; i < ct.NumIn(); i++ {
			edges = append(edges, DependencyEdge{
				Source: p.Type,
				Target: ct.In(i),
			})
		}
	}
	return edges
}

// GetAllDependencies returns all dependency edges across the entire application.
func (gi *GraphInspector) GetAllDependencies() []DependencyEdge {
	var edges []DependencyEdge
	for _, mod := range gi.modules {
		edges = append(edges, gi.GetDependencies(mod)...)
	}
	return edges
}

// SerializedGraph provides a full JSON-serializable representation of the
// application's dependency graph with nodes, edges, and metadata.
// Equivalent to NestJS SerializedGraph.
type SerializedGraph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// GraphNode represents a node in the serialized graph.
type GraphNode struct {
	ID       string            `json:"id"`
	Label    string            `json:"label"`
	Type     string            `json:"type"` // "module", "provider", "controller"
	Metadata map[string]any    `json:"metadata,omitempty"`
}

// GraphEdge represents an edge in the serialized graph.
type GraphEdge struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	Type     string `json:"type"` // "dependency", "import", "export"
	Metadata map[string]any `json:"metadata,omitempty"`
}

// DeterministicUUID generates a stable UUID-like identifier from a type name.
// This ensures IDs are consistent across runs.
func DeterministicUUID(name string) string {
	// Simple deterministic hash-based ID
	var h uint64
	for _, c := range name {
		h = h*31 + uint64(c)
	}
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		h&0xFFFFFFFF,
		(h>>32)&0xFFFF,
		(h>>48)&0xFFFF,
		(h>>16)&0xFFFF,
		h&0xFFFFFFFFFFFF,
	)
}

// Serialize produces a complete serialized graph of the application's
// module tree, providers, controllers, and their dependencies.
func (gi *GraphInspector) Serialize() *SerializedGraph {
	sg := &SerializedGraph{}
	edgeCount := 0

	for modIdx, mod := range gi.modules {
		// Module node
		modID := DeterministicUUID(fmt.Sprintf("module_%d", modIdx))
		sg.Nodes = append(sg.Nodes, GraphNode{
			ID:    modID,
			Label: fmt.Sprintf("Module[%d]", modIdx),
			Type:  "module",
			Metadata: map[string]any{
				"global":         mod.options.Global,
				"providerCount":  len(mod.options.Providers),
				"controllerCount": len(mod.controllers),
				"importCount":    len(mod.options.Imports),
			},
		})

		// Import edges
		for _, imp := range mod.options.Imports {
			for impIdx, m := range gi.modules {
				if m == imp {
					targetModID := DeterministicUUID(fmt.Sprintf("module_%d", impIdx))
					edgeCount++
					sg.Edges = append(sg.Edges, GraphEdge{
						ID:     DeterministicUUID(fmt.Sprintf("edge_import_%d", edgeCount)),
						Source: modID,
						Target: targetModID,
						Type:   "import",
					})
					break
				}
			}
		}

		if mod.container == nil {
			continue
		}

		// Provider nodes and dependency edges
		for t, entry := range mod.container.GetAllProviders() {
			providerID := DeterministicUUID("provider_" + t.String())
			sg.Nodes = append(sg.Nodes, GraphNode{
				ID:    providerID,
				Label: t.String(),
				Type:  "provider",
				Metadata: map[string]any{
					"scope":    entry.provider.Scope.String(),
					"resolved": entry.resolved,
					"module":   modID,
				},
			})

			// Module -> provider edge
			edgeCount++
			sg.Edges = append(sg.Edges, GraphEdge{
				ID:     DeterministicUUID(fmt.Sprintf("edge_contains_%d", edgeCount)),
				Source: modID,
				Target: providerID,
				Type:   "contains",
			})

			// Dependency edges
			if entry.provider.Constructor != nil {
				ct := reflect.TypeOf(entry.provider.Constructor)
				for i := 0; i < ct.NumIn(); i++ {
					depType := ct.In(i)
					depID := DeterministicUUID("provider_" + depType.String())
					edgeCount++
					sg.Edges = append(sg.Edges, GraphEdge{
						ID:     DeterministicUUID(fmt.Sprintf("edge_dep_%d", edgeCount)),
						Source: providerID,
						Target: depID,
						Type:   "dependency",
					})
				}
			}
		}

		// Controller nodes
		for ctrlIdx, ctrl := range mod.controllers {
			ctrlType := reflect.TypeOf(ctrl)
			ctrlID := DeterministicUUID(fmt.Sprintf("controller_%s_%d", ctrlType.String(), ctrlIdx))
			sg.Nodes = append(sg.Nodes, GraphNode{
				ID:    ctrlID,
				Label: ctrlType.String(),
				Type:  "controller",
				Metadata: map[string]any{
					"module": modID,
				},
			})

			// Module -> controller edge
			edgeCount++
			sg.Edges = append(sg.Edges, GraphEdge{
				ID:     DeterministicUUID(fmt.Sprintf("edge_ctrl_%d", edgeCount)),
				Source: modID,
				Target: ctrlID,
				Type:   "contains",
			})
		}

		// Export edges
		for _, exp := range mod.options.Exports {
			exportType := resolveExportType(exp)
			if exportType != nil {
				exportID := DeterministicUUID("provider_" + exportType.String())
				edgeCount++
				sg.Edges = append(sg.Edges, GraphEdge{
					ID:     DeterministicUUID(fmt.Sprintf("edge_export_%d", edgeCount)),
					Source: modID,
					Target: exportID,
					Type:   "export",
				})
			}
		}
	}

	return sg
}

// ToJSON serializes the graph to a JSON byte slice.
func (sg *SerializedGraph) ToJSON() ([]byte, error) {
	return json.Marshal(sg)
}

// ToJSONIndent serializes the graph to a pretty-printed JSON byte slice.
func (sg *SerializedGraph) ToJSONIndent() ([]byte, error) {
	return json.MarshalIndent(sg, "", "  ")
}

// FindNode returns the node with the given ID, or nil.
func (sg *SerializedGraph) FindNode(id string) *GraphNode {
	for i := range sg.Nodes {
		if sg.Nodes[i].ID == id {
			return &sg.Nodes[i]
		}
	}
	return nil
}

// FindEdgesFrom returns all edges originating from the given node ID.
func (sg *SerializedGraph) FindEdgesFrom(nodeID string) []GraphEdge {
	var result []GraphEdge
	for _, e := range sg.Edges {
		if e.Source == nodeID {
			result = append(result, e)
		}
	}
	return result
}

// FindEdgesTo returns all edges pointing to the given node ID.
func (sg *SerializedGraph) FindEdgesTo(nodeID string) []GraphEdge {
	var result []GraphEdge
	for _, e := range sg.Edges {
		if e.Target == nodeID {
			result = append(result, e)
		}
	}
	return result
}
