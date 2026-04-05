package integration

import (
	"encoding/json"
	"testing"

	"github.com/gonest"
)

// ---------------------------------------------------------------------------
// Graph Inspector Integration Tests
// Mirror: original/integration/inspector/
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Test module structure for graph inspection
// ---------------------------------------------------------------------------

type inspectedServiceA struct{}

func newInspectedServiceA() *inspectedServiceA { return &inspectedServiceA{} }

type inspectedServiceB struct {
	a *inspectedServiceA
}

func newInspectedServiceB(a *inspectedServiceA) *inspectedServiceB {
	return &inspectedServiceB{a: a}
}

type inspectedController struct {
	b *inspectedServiceB
}

func newInspectedController(b *inspectedServiceB) *inspectedController {
	return &inspectedController{b: b}
}

func (c *inspectedController) Register(r gonest.Router) {
	r.Get("/inspected", func(ctx gonest.Context) error {
		return ctx.JSON(200, "ok")
	})
}

// ---------------------------------------------------------------------------
// Tests: GraphInspector.GetModules
// ---------------------------------------------------------------------------

func TestInspector_GetModules_ReturnsAllModules(t *testing.T) {
	childModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newInspectedServiceA},
		Exports:   []any{(*inspectedServiceA)(nil)},
	})

	rootModule := gonest.NewModule(gonest.ModuleOptions{
		Imports:   []*gonest.Module{childModule},
		Providers: []any{newInspectedServiceB},
	})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	gi := app.GetGraphInspector()
	modules := gi.GetModules()

	if len(modules) < 2 {
		t.Fatalf("expected at least 2 modules, got %d", len(modules))
	}

	// Verify at least one module has imports
	hasImports := false
	for _, mn := range modules {
		if len(mn.Imports) > 0 {
			hasImports = true
			break
		}
	}
	if !hasImports {
		t.Error("expected at least one module with imports")
	}
}

func TestInspector_GetModules_IncludesGlobalModules(t *testing.T) {
	globalModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newInspectedServiceA},
		Exports:   []any{(*inspectedServiceA)(nil)},
		Global:    true,
	})

	rootModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{globalModule},
	})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	gi := app.GetGraphInspector()
	modules := gi.GetModules()

	if len(modules) < 2 {
		t.Fatalf("expected at least 2 modules, got %d", len(modules))
	}
}

// ---------------------------------------------------------------------------
// Tests: GraphInspector.GetDependencies
// ---------------------------------------------------------------------------

func TestInspector_GetDependencies_ReturnsDependencyEdges(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newInspectedServiceA, newInspectedServiceB},
	})

	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	gi := app.GetGraphInspector()
	modules := gi.GetModules()

	found := false
	for _, mn := range modules {
		edges := gi.GetDependencies(mn.Module)
		for _, edge := range edges {
			// inspectedServiceB depends on inspectedServiceA
			if edge.Source.String() == "*integration.inspectedServiceB" &&
				edge.Target.String() == "*integration.inspectedServiceA" {
				found = true
			}
		}
	}

	if !found {
		t.Error("expected dependency edge from inspectedServiceB to inspectedServiceA")
	}
}

func TestInspector_GetAllDependencies(t *testing.T) {
	childModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newInspectedServiceA},
		Exports:   []any{(*inspectedServiceA)(nil)},
	})

	parentModule := gonest.NewModule(gonest.ModuleOptions{
		Imports:   []*gonest.Module{childModule},
		Providers: []any{newInspectedServiceB},
	})

	app := gonest.Create(parentModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	gi := app.GetGraphInspector()
	edges := gi.GetAllDependencies()

	if len(edges) == 0 {
		t.Fatal("expected at least 1 dependency edge")
	}
}

// ---------------------------------------------------------------------------
// Tests: GraphInspector.Serialize
// ---------------------------------------------------------------------------

func TestInspector_Serialize_ProducesGraph(t *testing.T) {
	childModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newInspectedServiceA},
		Exports:   []any{(*inspectedServiceA)(nil)},
	})

	rootModule := gonest.NewModule(gonest.ModuleOptions{
		Imports:     []*gonest.Module{childModule},
		Controllers: []any{newInspectedController},
		Providers:   []any{newInspectedServiceB},
	})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	gi := app.GetGraphInspector()
	graph := gi.Serialize()

	if len(graph.Nodes) == 0 {
		t.Fatal("expected nodes in serialized graph")
	}
	if len(graph.Edges) == 0 {
		t.Fatal("expected edges in serialized graph")
	}

	// Check node types
	moduleNodes := 0
	providerNodes := 0
	controllerNodes := 0
	for _, node := range graph.Nodes {
		switch node.Type {
		case "module":
			moduleNodes++
		case "provider":
			providerNodes++
		case "controller":
			controllerNodes++
		}
	}

	if moduleNodes < 2 {
		t.Errorf("expected at least 2 module nodes, got %d", moduleNodes)
	}
	if providerNodes < 2 {
		t.Errorf("expected at least 2 provider nodes, got %d", providerNodes)
	}
	if controllerNodes < 1 {
		t.Errorf("expected at least 1 controller node, got %d", controllerNodes)
	}

	// Check edge types
	edgeTypes := make(map[string]int)
	for _, edge := range graph.Edges {
		edgeTypes[edge.Type]++
	}

	if edgeTypes["import"] == 0 {
		t.Error("expected import edges in graph")
	}
	if edgeTypes["contains"] == 0 {
		t.Error("expected contains edges in graph")
	}
	if edgeTypes["dependency"] == 0 {
		t.Error("expected dependency edges in graph")
	}
}

func TestInspector_Serialize_ExportEdges(t *testing.T) {
	childModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newInspectedServiceA},
		Exports:   []any{(*inspectedServiceA)(nil)},
	})

	rootModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{childModule},
	})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	gi := app.GetGraphInspector()
	graph := gi.Serialize()

	hasExport := false
	for _, edge := range graph.Edges {
		if edge.Type == "export" {
			hasExport = true
			break
		}
	}
	if !hasExport {
		t.Error("expected export edges in graph for exported providers")
	}
}

// ---------------------------------------------------------------------------
// Tests: SerializedGraph.ToJSON
// ---------------------------------------------------------------------------

func TestInspector_ToJSON_ValidJSON(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers:   []any{newInspectedServiceA, newInspectedServiceB},
		Controllers: []any{newInspectedController},
	})

	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	gi := app.GetGraphInspector()
	graph := gi.Serialize()

	data, err := graph.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	// Verify it's valid JSON
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Check structure
	if _, ok := parsed["nodes"]; !ok {
		t.Error("JSON missing 'nodes' key")
	}
	if _, ok := parsed["edges"]; !ok {
		t.Error("JSON missing 'edges' key")
	}
}

func TestInspector_ToJSONIndent_FormattedOutput(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newInspectedServiceA},
	})

	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	gi := app.GetGraphInspector()
	graph := gi.Serialize()

	data, err := graph.ToJSONIndent()
	if err != nil {
		t.Fatal(err)
	}

	// Indented JSON should have newlines
	if len(data) == 0 {
		t.Fatal("expected non-empty JSON output")
	}

	str := string(data)
	if str[0] != '{' {
		t.Errorf("expected JSON to start with '{', got %q", string(str[0]))
	}
}

// ---------------------------------------------------------------------------
// Tests: SerializedGraph.FindNode
// ---------------------------------------------------------------------------

func TestInspector_FindNode_ByID(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newInspectedServiceA},
	})

	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	gi := app.GetGraphInspector()
	graph := gi.Serialize()

	if len(graph.Nodes) == 0 {
		t.Fatal("expected nodes in graph")
	}

	// Find the first node by its ID
	firstID := graph.Nodes[0].ID
	found := graph.FindNode(firstID)
	if found == nil {
		t.Errorf("FindNode(%q) returned nil", firstID)
	}
	if found != nil && found.ID != firstID {
		t.Errorf("expected ID=%q, got %q", firstID, found.ID)
	}
}

func TestInspector_FindNode_NotFound(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newInspectedServiceA},
	})

	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	gi := app.GetGraphInspector()
	graph := gi.Serialize()

	found := graph.FindNode("nonexistent-id")
	if found != nil {
		t.Error("expected nil for nonexistent node ID")
	}
}

// ---------------------------------------------------------------------------
// Tests: SerializedGraph.FindEdgesFrom / FindEdgesTo
// ---------------------------------------------------------------------------

func TestInspector_FindEdgesFrom(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers:   []any{newInspectedServiceA, newInspectedServiceB},
		Controllers: []any{newInspectedController},
	})

	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	gi := app.GetGraphInspector()
	graph := gi.Serialize()

	// Find a module node
	var moduleNodeID string
	for _, n := range graph.Nodes {
		if n.Type == "module" {
			moduleNodeID = n.ID
			break
		}
	}
	if moduleNodeID == "" {
		t.Fatal("no module node found")
	}

	edges := graph.FindEdgesFrom(moduleNodeID)
	if len(edges) == 0 {
		t.Error("expected edges from module node (contains providers/controllers)")
	}
}

func TestInspector_FindEdgesTo(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newInspectedServiceA, newInspectedServiceB},
	})

	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	gi := app.GetGraphInspector()
	graph := gi.Serialize()

	// Find a provider node and check incoming edges
	var providerNodeID string
	for _, n := range graph.Nodes {
		if n.Type == "provider" {
			providerNodeID = n.ID
			break
		}
	}
	if providerNodeID == "" {
		t.Fatal("no provider node found")
	}

	edges := graph.FindEdgesTo(providerNodeID)
	if len(edges) == 0 {
		t.Error("expected edges pointing to provider node (contains edge from module)")
	}
}

// ---------------------------------------------------------------------------
// Tests: DeterministicUUID stability
// ---------------------------------------------------------------------------

func TestInspector_DeterministicUUID_Stable(t *testing.T) {
	id1 := gonest.DeterministicUUID("test_input")
	id2 := gonest.DeterministicUUID("test_input")

	if id1 != id2 {
		t.Errorf("DeterministicUUID not stable: %q != %q", id1, id2)
	}
}

func TestInspector_DeterministicUUID_DifferentInputs(t *testing.T) {
	id1 := gonest.DeterministicUUID("input_a")
	id2 := gonest.DeterministicUUID("input_b")

	if id1 == id2 {
		t.Error("DeterministicUUID produced same ID for different inputs")
	}
}

// ---------------------------------------------------------------------------
// Tests: Serialize with metadata
// ---------------------------------------------------------------------------

func TestInspector_Serialize_ModuleMetadata(t *testing.T) {
	globalModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newInspectedServiceA},
		Exports:   []any{(*inspectedServiceA)(nil)},
		Global:    true,
	})

	rootModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{globalModule},
	})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	gi := app.GetGraphInspector()
	graph := gi.Serialize()

	// Find a module node with global=true
	foundGlobal := false
	for _, n := range graph.Nodes {
		if n.Type == "module" && n.Metadata != nil {
			if g, ok := n.Metadata["global"]; ok && g == true {
				foundGlobal = true
				break
			}
		}
	}
	if !foundGlobal {
		t.Error("expected a module node with global=true metadata")
	}
}

func TestInspector_Serialize_ProviderMetadata(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newInspectedServiceA},
	})

	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	gi := app.GetGraphInspector()
	graph := gi.Serialize()

	foundProvider := false
	for _, n := range graph.Nodes {
		if n.Type == "provider" && n.Metadata != nil {
			if _, ok := n.Metadata["scope"]; ok {
				foundProvider = true
				break
			}
		}
	}
	if !foundProvider {
		t.Error("expected provider node with scope metadata")
	}
}
