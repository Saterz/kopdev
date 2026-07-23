package minify

import (
	"bytes"
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestLuaMinifier(t *testing.T) {
	t.Parallel()

	minifier, err := NewLuaMinifier()
	if err != nil {
		t.Fatalf("NewLuaMinifier() error = %v", err)
	}
	defer minifier.Close()

	source := []byte(`-- plugin comment
local greeting = "hello -- this is not a comment"
local escaped = "quote: \" slash: \\ newline: \n"

local function greet(name)
    return greeting .. ", " .. name
end

return {
    greet = greet,
    escaped = escaped,
}
`)

	result, err := minifier.Minify(source)
	if err != nil {
		t.Fatalf("Minify() error = %v", err)
	}

	if bytes.Contains(result, []byte("plugin comment")) {
		t.Errorf("Minify() retained a comment:\n%s", result)
	}
	for _, literal := range []string{
		`"hello -- this is not a comment"`,
		`"quote: \" slash: \\ newline: \n"`,
	} {
		if !bytes.Contains(result, []byte(literal)) {
			t.Errorf("Minify() changed or removed string literal %q:\n%s", literal, result)
		}
	}
	if len(result) >= len(source) {
		t.Errorf("Minify() result has length %d, want less than source length %d", len(result), len(source))
	}

	if _, err := minifier.Minify(result); err != nil {
		t.Errorf("Minify() result is not valid Lua: %v", err)
	}
}

func TestLuaMinifierPreservesLongString(t *testing.T) {
	t.Parallel()

	minifier, err := NewLuaMinifier()
	if err != nil {
		t.Fatalf("NewLuaMinifier() error = %v", err)
	}
	defer minifier.Close()

	source := []byte(`local text = [=[
This is not a comment: -- keep this
]=]

return text
`)

	result, err := minifier.Minify(source)
	if err != nil {
		t.Fatalf("Minify() error = %v", err)
	}

	longString := []byte("[=[\nThis is not a comment: -- keep this\n]=]")
	if !bytes.Contains(result, longString) {
		t.Errorf("Minify() changed long string contents:\n%s", result)
	}
}

func TestLuaMinifierHandlesBinarySafeStringsAndShebang(t *testing.T) {
	t.Parallel()

	minifier, err := NewLuaMinifier()
	if err != nil {
		t.Fatalf("NewLuaMinifier() error = %v", err)
	}
	defer minifier.Close()

	source := append([]byte("#!/usr/bin/env lua\nlocal value = [[a"), 0)
	source = append(source, []byte("b]] -- remove\nreturn value\n")...)

	result, err := minifier.Minify(source)
	if err != nil {
		t.Fatalf("Minify() error = %v", err)
	}

	if !bytes.HasPrefix(result, []byte("#!/usr/bin/env lua")) {
		t.Errorf("Minify() did not preserve shebang:\n%q", result)
	}
	if !bytes.Contains(result, []byte{'a', 0, 'b'}) {
		t.Errorf("Minify() did not preserve NUL in long string:\n%q", result)
	}
	if bytes.Contains(result, []byte("remove")) {
		t.Errorf("Minify() retained trailing comment:\n%q", result)
	}
}

func TestLuaMinifierRejectsInvalidLua(t *testing.T) {
	t.Parallel()

	minifier, err := NewLuaMinifier()
	if err != nil {
		t.Fatalf("NewLuaMinifier() error = %v", err)
	}
	defer minifier.Close()

	_, err = minifier.Minify([]byte("local function broken(\n"))
	if err == nil {
		t.Fatal("Minify() error = nil, want malformed Lua error")
	}
	if !strings.Contains(err.Error(), "LuaSrcDiet") {
		t.Errorf("Minify() error = %q, want LuaSrcDiet context", err)
	}
}

func TestLuaMinifierCloseIsIdempotent(t *testing.T) {
	t.Parallel()

	minifier, err := NewLuaMinifier()
	if err != nil {
		t.Fatalf("NewLuaMinifier() error = %v", err)
	}
	minifier.Close()
	minifier.Close()

	_, err = minifier.Minify([]byte("return true"))
	if err == nil {
		t.Fatal("Minify() after Close() error = nil")
	}
}

func TestLuaMinifierDoesNotExposeHostLibraries(t *testing.T) {
	t.Parallel()

	minifier, err := NewLuaMinifier()
	if err != nil {
		t.Fatalf("NewLuaMinifier() error = %v", err)
	}
	defer minifier.Close()

	for _, name := range []string{"io", "os", "package", "debug", "dofile", "loadfile"} {
		if value := minifier.state.GetGlobal(name); value != lua.LNil {
			t.Errorf("global %q = %s, want nil", name, value.Type())
		}
	}

	err = minifier.state.CallByParam(
		lua.P{Fn: minifier.state.GetGlobal("require"), NRet: 1, Protect: true},
		lua.LString("os"),
	)
	if err == nil {
		t.Errorf(`require("os") error = nil, want unavailable embedded module error`)
	}
}
