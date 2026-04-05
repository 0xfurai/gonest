package integration

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/gonest"
)

// ---------------------------------------------------------------------------
// REPL Integration Tests
// Mirror: original/integration/repl/
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Helper: create app and REPL with string I/O
// ---------------------------------------------------------------------------

func createREPLApp(t *testing.T) (*gonest.Application, *gonest.REPL) {
	t.Helper()

	svc := &replTestService{Name: "test-svc"}
	ctrl := &replTestController{svc: svc}

	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{func(s *replTestService) *replTestController {
			return ctrl
		}},
		Providers: []any{gonest.ProvideValue[*replTestService](svc)},
	})

	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}

	repl := gonest.NewREPL(app)
	return app, repl
}

func runREPL(t *testing.T, repl *gonest.REPL, input string) string {
	t.Helper()
	reader := strings.NewReader(input + "\nexit\n")
	var output bytes.Buffer
	repl.SetIO(reader, &output)
	repl.Start()
	return output.String()
}

// ---------------------------------------------------------------------------
// Test services and controllers
// ---------------------------------------------------------------------------

type replTestService struct {
	Name string
}

func (s *replTestService) Greet(name string) string {
	return "Hello, " + name
}

func (s *replTestService) GetName() string {
	return s.Name
}

type replTestController struct {
	svc *replTestService
}

func (c *replTestController) Register(r gonest.Router) {
	r.Prefix("/repl-test")
	r.Get("", c.handler)
	r.Post("/create", c.createHandler)
}

func (c *replTestController) handler(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{"service": c.svc.Name})
}

func (c *replTestController) createHandler(ctx gonest.Context) error {
	return ctx.JSON(http.StatusCreated, "created")
}

// ---------------------------------------------------------------------------
// Tests: help command
// ---------------------------------------------------------------------------

func TestREPL_Help(t *testing.T) {
	app, repl := createREPLApp(t)
	defer app.Close()

	output := runREPL(t, repl, "help")

	if !strings.Contains(output, "Available commands") {
		t.Error("help output should contain 'Available commands'")
	}
	if !strings.Contains(output, "modules") {
		t.Error("help output should list 'modules' command")
	}
	if !strings.Contains(output, "providers") {
		t.Error("help output should list 'providers' command")
	}
	if !strings.Contains(output, "controllers") {
		t.Error("help output should list 'controllers' command")
	}
	if !strings.Contains(output, "routes") {
		t.Error("help output should list 'routes' command")
	}
	if !strings.Contains(output, "resolve") {
		t.Error("help output should list 'resolve' command")
	}
	if !strings.Contains(output, "methods") {
		t.Error("help output should list 'methods' command")
	}
	if !strings.Contains(output, "debug") {
		t.Error("help output should list 'debug' command")
	}
}

// ---------------------------------------------------------------------------
// Tests: modules command
// ---------------------------------------------------------------------------

func TestREPL_Modules(t *testing.T) {
	app, repl := createREPLApp(t)
	defer app.Close()

	output := runREPL(t, repl, "modules")

	if !strings.Contains(output, "Modules") {
		t.Error("modules output should contain 'Modules' header")
	}
	if !strings.Contains(output, "providers=") {
		t.Error("modules output should show provider count")
	}
	if !strings.Contains(output, "controllers=") {
		t.Error("modules output should show controller count")
	}
}

func TestREPL_Modules_AliasLs(t *testing.T) {
	app, repl := createREPLApp(t)
	defer app.Close()

	output := runREPL(t, repl, "ls")

	if !strings.Contains(output, "Modules") {
		t.Error("'ls' should be alias for 'modules'")
	}
}

// ---------------------------------------------------------------------------
// Tests: providers command
// ---------------------------------------------------------------------------

func TestREPL_Providers(t *testing.T) {
	app, repl := createREPLApp(t)
	defer app.Close()

	output := runREPL(t, repl, "providers")

	if !strings.Contains(output, "Providers") {
		t.Error("providers output should contain 'Providers' header")
	}
	if !strings.Contains(output, "replTestService") {
		t.Error("providers output should list replTestService")
	}
}

// ---------------------------------------------------------------------------
// Tests: controllers command
// ---------------------------------------------------------------------------

func TestREPL_Controllers(t *testing.T) {
	app, repl := createREPLApp(t)
	defer app.Close()

	output := runREPL(t, repl, "controllers")

	if !strings.Contains(output, "Controllers") {
		t.Error("controllers output should contain 'Controllers' header")
	}
	if !strings.Contains(output, "replTestController") {
		t.Error("controllers output should list replTestController")
	}
}

// ---------------------------------------------------------------------------
// Tests: routes command
// ---------------------------------------------------------------------------

func TestREPL_Routes(t *testing.T) {
	app, repl := createREPLApp(t)
	defer app.Close()

	output := runREPL(t, repl, "routes")

	if !strings.Contains(output, "Routes") {
		t.Error("routes output should contain 'Routes' header")
	}
	if !strings.Contains(output, "GET") {
		t.Error("routes output should show GET route")
	}
	if !strings.Contains(output, "POST") {
		t.Error("routes output should show POST route")
	}
}

// ---------------------------------------------------------------------------
// Tests: resolve command
// ---------------------------------------------------------------------------

func TestREPL_Resolve_ExistingProvider(t *testing.T) {
	app, repl := createREPLApp(t)
	defer app.Close()

	output := runREPL(t, repl, "resolve replTestService")

	if !strings.Contains(output, "Resolved") || !strings.Contains(output, "replTestService") {
		t.Errorf("resolve should show resolved provider, got: %s", output)
	}
}

func TestREPL_Resolve_NotFound(t *testing.T) {
	app, repl := createREPLApp(t)
	defer app.Close()

	output := runREPL(t, repl, "resolve NonExistentService")

	if !strings.Contains(output, "No provider matching") {
		t.Errorf("resolve of missing provider should show 'No provider matching', got: %s", output)
	}
}

func TestREPL_Resolve_MissingArg(t *testing.T) {
	app, repl := createREPLApp(t)
	defer app.Close()

	output := runREPL(t, repl, "resolve")

	if !strings.Contains(output, "Usage:") {
		t.Error("resolve without arg should show usage")
	}
}

// ---------------------------------------------------------------------------
// Tests: methods command
// ---------------------------------------------------------------------------

func TestREPL_Methods_ExistingProvider(t *testing.T) {
	app, repl := createREPLApp(t)
	defer app.Close()

	output := runREPL(t, repl, "methods replTestService")

	if !strings.Contains(output, "Methods on") {
		t.Errorf("methods should show method list, got: %s", output)
	}
	if !strings.Contains(output, "Greet") {
		t.Error("methods should list Greet method")
	}
	if !strings.Contains(output, "GetName") {
		t.Error("methods should list GetName method")
	}
}

func TestREPL_Methods_NotFound(t *testing.T) {
	app, repl := createREPLApp(t)
	defer app.Close()

	output := runREPL(t, repl, "methods NonExistent")

	if !strings.Contains(output, "No provider matching") {
		t.Errorf("methods of missing provider should show error, got: %s", output)
	}
}

func TestREPL_Methods_MissingArg(t *testing.T) {
	app, repl := createREPLApp(t)
	defer app.Close()

	output := runREPL(t, repl, "methods")

	if !strings.Contains(output, "Usage:") {
		t.Error("methods without arg should show usage")
	}
}

// ---------------------------------------------------------------------------
// Tests: debug command
// ---------------------------------------------------------------------------

func TestREPL_Debug(t *testing.T) {
	app, repl := createREPLApp(t)
	defer app.Close()

	output := runREPL(t, repl, "debug")

	if !strings.Contains(output, "Dependency edges") {
		t.Errorf("debug should show dependency edges, got: %s", output)
	}
}

// ---------------------------------------------------------------------------
// Tests: unknown command
// ---------------------------------------------------------------------------

func TestREPL_UnknownCommand(t *testing.T) {
	app, repl := createREPLApp(t)
	defer app.Close()

	output := runREPL(t, repl, "foobar")

	if !strings.Contains(output, "Unknown command") {
		t.Error("unknown command should show error message")
	}
}

// ---------------------------------------------------------------------------
// Tests: exit command
// ---------------------------------------------------------------------------

func TestREPL_Exit(t *testing.T) {
	app, repl := createREPLApp(t)
	defer app.Close()

	reader := strings.NewReader("exit\n")
	var output bytes.Buffer
	repl.SetIO(reader, &output)
	repl.Start()

	if !strings.Contains(output.String(), "Bye!") {
		t.Error("exit should print 'Bye!'")
	}
}

func TestREPL_Quit(t *testing.T) {
	app, repl := createREPLApp(t)
	defer app.Close()

	reader := strings.NewReader("quit\n")
	var output bytes.Buffer
	repl.SetIO(reader, &output)
	repl.Start()

	if !strings.Contains(output.String(), "Bye!") {
		t.Error("quit should print 'Bye!'")
	}
}

// ---------------------------------------------------------------------------
// Tests: REPL banner
// ---------------------------------------------------------------------------

func TestREPL_Banner(t *testing.T) {
	app, repl := createREPLApp(t)
	defer app.Close()

	reader := strings.NewReader("exit\n")
	var output bytes.Buffer
	repl.SetIO(reader, &output)
	repl.Start()

	if !strings.Contains(output.String(), "GoNest REPL") {
		t.Error("REPL should show banner on start")
	}
}

// ---------------------------------------------------------------------------
// Tests: Multiple commands in sequence
// ---------------------------------------------------------------------------

func TestREPL_MultipleCommands(t *testing.T) {
	app, repl := createREPLApp(t)
	defer app.Close()

	output := runREPL(t, repl, "modules\nproviders\nroutes")

	if !strings.Contains(output, "Modules") {
		t.Error("expected modules output")
	}
	if !strings.Contains(output, "Providers") {
		t.Error("expected providers output")
	}
	if !strings.Contains(output, "Routes") {
		t.Error("expected routes output")
	}
}

// ---------------------------------------------------------------------------
// Tests: Empty input is ignored
// ---------------------------------------------------------------------------

func TestREPL_EmptyInput(t *testing.T) {
	app, repl := createREPLApp(t)
	defer app.Close()

	// Empty lines followed by help then exit
	output := runREPL(t, repl, "\n\nhelp")

	if !strings.Contains(output, "Available commands") {
		t.Error("empty inputs should be skipped, then help should work")
	}
}
