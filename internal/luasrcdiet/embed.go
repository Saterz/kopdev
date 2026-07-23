// Package luasrcdiet provides access to the vendored LuaSrcDiet modules.
package luasrcdiet

import (
	"embed"
	"fmt"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

//go:embed lua/luasrcdiet/*.lua
var moduleFiles embed.FS

var modulePaths = map[string]string{
	"luasrcdiet":           "lua/luasrcdiet/init.lua",
	"luasrcdiet.equiv":     "lua/luasrcdiet/equiv.lua",
	"luasrcdiet.llex":      "lua/luasrcdiet/llex.lua",
	"luasrcdiet.lparser":   "lua/luasrcdiet/lparser.lua",
	"luasrcdiet.optlex":    "lua/luasrcdiet/optlex.lua",
	"luasrcdiet.optparser": "lua/luasrcdiet/optparser.lua",
	"luasrcdiet.utils":     "lua/luasrcdiet/utils.lua",
}

// Loader resolves LuaSrcDiet modules from the embedded filesystem.
//
// Loader deliberately implements only the module names in the vendored
// luasrcdiet namespace. It does not fall back to the host filesystem.
type Loader struct {
	loaded  map[string]lua.LValue
	loading map[string]bool
}

// NewLoader creates an isolated embedded module loader.
func NewLoader() *Loader {
	return &Loader{
		loaded:  make(map[string]lua.LValue),
		loading: make(map[string]bool),
	}
}

// Require is a GopherLua function compatible with Lua's require function.
func (loader *Loader) Require(state *lua.LState) int {
	name := state.CheckString(1)

	if value, ok := loader.loaded[name]; ok {
		state.Push(value)
		return 1
	}
	if loader.loading[name] {
		state.RaiseError("loop loading embedded module %q", name)
		return 0
	}

	modulePath, err := resolveModule(name)
	if err != nil {
		state.RaiseError("%v", err)
		return 0
	}

	source, err := moduleFiles.ReadFile(modulePath)
	if err != nil {
		state.RaiseError("read embedded module %q: %v", name, err)
		return 0
	}

	chunk, err := state.Load(strings.NewReader(string(source)), "@embedded/"+modulePath)
	if err != nil {
		state.RaiseError("compile embedded module %q: %v", name, err)
		return 0
	}

	loader.loading[name] = true
	state.Push(chunk)
	callErr := state.PCall(0, 1, nil)
	delete(loader.loading, name)
	if callErr != nil {
		state.RaiseError("load embedded module %q: %v", name, callErr)
		return 0
	}

	value := state.Get(-1)
	state.Pop(1)
	if value == lua.LNil {
		value = lua.LTrue
	}
	loader.loaded[name] = value
	state.Push(value)
	return 1
}

func resolveModule(name string) (string, error) {
	modulePath, ok := modulePaths[name]
	if !ok {
		return "", fmt.Errorf("embedded Lua module %q is not available", name)
	}
	return modulePath, nil
}
