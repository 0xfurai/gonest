package gonest

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
)

// REPL provides an interactive Read-Eval-Print Loop for inspecting and debugging
// a running application. Equivalent to NestJS REPL.
type REPL struct {
	app    *Application
	ctx    *ApplicationContext
	reader io.Reader
	writer io.Writer
}

// NewREPL creates a new REPL attached to an HTTP application.
func NewREPL(app *Application) *REPL {
	return &REPL{
		app:    app,
		reader: os.Stdin,
		writer: os.Stdout,
	}
}

// NewREPLFromContext creates a new REPL attached to an application context.
func NewREPLFromContext(ctx *ApplicationContext) *REPL {
	return &REPL{
		ctx:    ctx,
		reader: os.Stdin,
		writer: os.Stdout,
	}
}

// SetIO overrides the default stdin/stdout for testing.
func (r *REPL) SetIO(reader io.Reader, writer io.Writer) {
	r.reader = reader
	r.writer = writer
}

// Start begins the interactive REPL loop.
func (r *REPL) Start() {
	fmt.Fprintln(r.writer, "GoNest REPL — type 'help' for available commands, 'exit' to quit")
	scanner := bufio.NewScanner(r.reader)
	for {
		fmt.Fprint(r.writer, "> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			fmt.Fprintln(r.writer, "Bye!")
			break
		}
		r.execute(line)
	}
}

func (r *REPL) execute(line string) {
	parts := strings.Fields(line)
	cmd := parts[0]

	switch cmd {
	case "help":
		r.cmdHelp()
	case "modules", "ls":
		r.cmdModules()
	case "providers":
		r.cmdProviders()
	case "controllers":
		r.cmdControllers()
	case "routes":
		r.cmdRoutes()
	case "resolve":
		if len(parts) < 2 {
			fmt.Fprintln(r.writer, "Usage: resolve <TypeName>")
			return
		}
		r.cmdResolve(parts[1])
	case "methods":
		if len(parts) < 2 {
			fmt.Fprintln(r.writer, "Usage: methods <TypeName>")
			return
		}
		r.cmdMethods(parts[1])
	case "debug":
		r.cmdDebug()
	default:
		fmt.Fprintf(r.writer, "Unknown command: %s (type 'help')\n", cmd)
	}
}

func (r *REPL) cmdHelp() {
	fmt.Fprintln(r.writer, "Available commands:")
	fmt.Fprintln(r.writer, "  help          — show this help")
	fmt.Fprintln(r.writer, "  modules / ls  — list all modules")
	fmt.Fprintln(r.writer, "  providers     — list all providers")
	fmt.Fprintln(r.writer, "  controllers   — list all controllers")
	fmt.Fprintln(r.writer, "  routes        — list all registered routes")
	fmt.Fprintln(r.writer, "  resolve <T>   — resolve a provider by type name")
	fmt.Fprintln(r.writer, "  methods <T>   — list methods on a type")
	fmt.Fprintln(r.writer, "  debug         — show dependency graph summary")
	fmt.Fprintln(r.writer, "  exit / quit   — exit the REPL")
}

func (r *REPL) getContainer() *Container {
	if r.app != nil {
		return r.app.GetContainer()
	}
	if r.ctx != nil {
		return r.ctx.GetContainer()
	}
	return nil
}

func (r *REPL) getModules() []*Module {
	if r.app != nil {
		return r.app.module.allModules()
	}
	if r.ctx != nil {
		return r.ctx.module.allModules()
	}
	return nil
}

func (r *REPL) cmdModules() {
	mods := r.getModules()
	fmt.Fprintf(r.writer, "Modules (%d):\n", len(mods))
	for i, mod := range mods {
		providerCount := 0
		ctrlCount := len(mod.controllers)
		if mod.container != nil {
			providerCount = len(mod.container.GetAllProviders())
		}
		fmt.Fprintf(r.writer, "  [%d] providers=%d controllers=%d global=%v\n",
			i, providerCount, ctrlCount, mod.options.Global)
	}
}

func (r *REPL) cmdProviders() {
	container := r.getContainer()
	if container == nil {
		fmt.Fprintln(r.writer, "No container available")
		return
	}
	providers := container.GetAllProviders()
	names := make([]string, 0, len(providers))
	for t := range providers {
		names = append(names, t.String())
	}
	sort.Strings(names)

	fmt.Fprintf(r.writer, "Providers (%d):\n", len(names))
	for _, name := range names {
		fmt.Fprintf(r.writer, "  - %s\n", name)
	}
}

func (r *REPL) cmdControllers() {
	mods := r.getModules()
	var ctrls []Controller
	for _, mod := range mods {
		ctrls = append(ctrls, mod.controllers...)
	}
	fmt.Fprintf(r.writer, "Controllers (%d):\n", len(ctrls))
	for _, ctrl := range ctrls {
		fmt.Fprintf(r.writer, "  - %s\n", reflect.TypeOf(ctrl).String())
	}
}

func (r *REPL) cmdRoutes() {
	if r.app == nil {
		fmt.Fprintln(r.writer, "Routes only available for HTTP applications")
		return
	}
	routes := r.app.GetRoutes()
	fmt.Fprintf(r.writer, "Routes (%d):\n", len(routes))
	for _, route := range routes {
		fmt.Fprintf(r.writer, "  %s %s\n", route.Method, route.Path)
	}
}

func (r *REPL) cmdResolve(typeName string) {
	container := r.getContainer()
	if container == nil {
		fmt.Fprintln(r.writer, "No container available")
		return
	}
	for t, entry := range container.GetAllProviders() {
		if strings.Contains(t.String(), typeName) {
			if entry.resolved {
				fmt.Fprintf(r.writer, "Resolved %s: %v\n", t.String(), reflect.TypeOf(entry.instance))
			} else {
				fmt.Fprintf(r.writer, "Found %s (not yet resolved)\n", t.String())
			}
			return
		}
	}
	fmt.Fprintf(r.writer, "No provider matching %q\n", typeName)
}

func (r *REPL) cmdMethods(typeName string) {
	container := r.getContainer()
	if container == nil {
		fmt.Fprintln(r.writer, "No container available")
		return
	}
	for t, entry := range container.GetAllProviders() {
		if !strings.Contains(t.String(), typeName) {
			continue
		}
		if !entry.resolved || entry.instance == nil {
			fmt.Fprintf(r.writer, "Provider %s not yet resolved\n", t.String())
			return
		}
		instType := reflect.TypeOf(entry.instance)
		fmt.Fprintf(r.writer, "Methods on %s (%d):\n", instType.String(), instType.NumMethod())
		for i := 0; i < instType.NumMethod(); i++ {
			m := instType.Method(i)
			fmt.Fprintf(r.writer, "  - %s%s\n", m.Name, methodSignature(m.Type))
		}
		return
	}
	fmt.Fprintf(r.writer, "No provider matching %q\n", typeName)
}

func (r *REPL) cmdDebug() {
	if r.app != nil && r.app.graphInspector != nil {
		edges := r.app.graphInspector.GetAllDependencies()
		fmt.Fprintf(r.writer, "Dependency edges (%d):\n", len(edges))
		for _, edge := range edges {
			fmt.Fprintf(r.writer, "  %s -> %s\n", edge.Source, edge.Target)
		}
		return
	}
	if r.ctx != nil && r.ctx.graphInspector != nil {
		edges := r.ctx.graphInspector.GetAllDependencies()
		fmt.Fprintf(r.writer, "Dependency edges (%d):\n", len(edges))
		for _, edge := range edges {
			fmt.Fprintf(r.writer, "  %s -> %s\n", edge.Source, edge.Target)
		}
		return
	}
	fmt.Fprintln(r.writer, "No graph inspector available")
}

func methodSignature(t reflect.Type) string {
	var params []string
	// Skip receiver (index 0)
	for i := 1; i < t.NumIn(); i++ {
		params = append(params, t.In(i).String())
	}
	var returns []string
	for i := 0; i < t.NumOut(); i++ {
		returns = append(returns, t.Out(i).String())
	}
	sig := "(" + strings.Join(params, ", ") + ")"
	if len(returns) > 0 {
		sig += " (" + strings.Join(returns, ", ") + ")"
	}
	return sig
}
