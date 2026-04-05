package gonest

import (
	"bytes"
	"strings"
	"testing"
)

func TestREPL_Help(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})
	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.Init()

	repl := NewREPL(app)
	input := strings.NewReader("help\nexit\n")
	output := &bytes.Buffer{}
	repl.SetIO(input, output)
	repl.Start()

	if !strings.Contains(output.String(), "Available commands") {
		t.Error("expected help output")
	}
}

func TestREPL_Modules(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})
	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.Init()

	repl := NewREPL(app)
	input := strings.NewReader("modules\nexit\n")
	output := &bytes.Buffer{}
	repl.SetIO(input, output)
	repl.Start()

	if !strings.Contains(output.String(), "Modules") {
		t.Error("expected modules output")
	}
}

func TestREPL_Routes(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})
	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.Init()

	repl := NewREPL(app)
	input := strings.NewReader("routes\nexit\n")
	output := &bytes.Buffer{}
	repl.SetIO(input, output)
	repl.Start()

	if !strings.Contains(output.String(), "Routes") {
		t.Error("expected routes output")
	}
}

func TestREPL_Providers(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})
	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.Init()

	repl := NewREPL(app)
	input := strings.NewReader("providers\nexit\n")
	output := &bytes.Buffer{}
	repl.SetIO(input, output)
	repl.Start()

	if !strings.Contains(output.String(), "Providers") {
		t.Error("expected providers output")
	}
}

func TestREPL_UnknownCommand(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})
	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.Init()

	repl := NewREPL(app)
	input := strings.NewReader("foobar\nexit\n")
	output := &bytes.Buffer{}
	repl.SetIO(input, output)
	repl.Start()

	if !strings.Contains(output.String(), "Unknown command") {
		t.Error("expected unknown command message")
	}
}

func TestREPL_FromContext(t *testing.T) {
	module := NewModule(ModuleOptions{
		Providers: []any{newGreetingService},
	})

	ctx, err := CreateApplicationContext(module, ApplicationOptions{Logger: NopLogger{}})
	if err != nil {
		t.Fatalf("create context failed: %v", err)
	}
	defer ctx.Close()

	repl := NewREPLFromContext(ctx)
	input := strings.NewReader("providers\nexit\n")
	output := &bytes.Buffer{}
	repl.SetIO(input, output)
	repl.Start()

	if !strings.Contains(output.String(), "Providers") {
		t.Error("expected providers output from context REPL")
	}
}

func TestREPL_Debug(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})
	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.Init()

	repl := NewREPL(app)
	input := strings.NewReader("debug\nexit\n")
	output := &bytes.Buffer{}
	repl.SetIO(input, output)
	repl.Start()

	if !strings.Contains(output.String(), "Dependency edges") {
		t.Error("expected debug output")
	}
}
