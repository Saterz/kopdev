// Package minify provides source minifiers used while packaging plugins.
package minify

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/Saterz/kopdev/internal/luasrcdiet"
	lua "github.com/yuin/gopher-lua"
)

// LuaMinifier runs the vendored LuaSrcDiet BASIC_OPTS profile.
//
// A LuaMinifier may be reused sequentially, but it is not safe for concurrent
// use.
type LuaMinifier struct {
	state    *lua.LState
	optimize *lua.LFunction
	options  *lua.LTable
}

// NewLuaMinifier initializes an isolated LuaSrcDiet runtime.
func NewLuaMinifier() (_ *LuaMinifier, err error) {
	var state *lua.LState
	defer func() {
		if recovered := recover(); recovered != nil {
			if state != nil {
				state.Close()
			}
			err = fmt.Errorf("initialize embedded LuaSrcDiet runtime: %v", recovered)
		}
	}()

	state = lua.NewState(lua.Options{SkipOpenLibs: true})
	lua.OpenBase(state)
	lua.OpenTable(state)
	lua.OpenString(state)
	lua.OpenMath(state)
	installLua51Compatibility(state)

	// The base library's file helpers are not needed by LuaSrcDiet.
	state.SetGlobal("dofile", lua.LNil)
	state.SetGlobal("loadfile", lua.LNil)

	loader := luasrcdiet.NewLoader()
	state.SetGlobal("require", state.NewFunction(loader.Require))

	if err := state.CallByParam(
		lua.P{Fn: state.GetGlobal("require"), NRet: 1, Protect: true},
		lua.LString("luasrcdiet"),
	); err != nil {
		state.Close()
		return nil, fmt.Errorf("initialize embedded LuaSrcDiet runtime: %w", err)
	}

	moduleValue := state.Get(-1)
	state.Pop(1)
	module, ok := moduleValue.(*lua.LTable)
	if !ok {
		state.Close()
		return nil, fmt.Errorf("initialize embedded LuaSrcDiet runtime: module returned %s", moduleValue.Type())
	}

	optimize, ok := state.GetField(module, "optimize").(*lua.LFunction)
	if !ok {
		state.Close()
		return nil, fmt.Errorf("initialize embedded LuaSrcDiet runtime: optimize is not a function")
	}
	options, ok := state.GetField(module, "BASIC_OPTS").(*lua.LTable)
	if !ok {
		state.Close()
		return nil, fmt.Errorf("initialize embedded LuaSrcDiet runtime: BASIC_OPTS is not a table")
	}

	return &LuaMinifier{
		state:    state,
		optimize: optimize,
		options:  options,
	}, nil
}

// installLua51Compatibility fills two GopherLua compatibility gaps exercised
// by LuaSrcDiet. Standard Lua 5.1 coerces numeric strings passed to string.sub,
// while GopherLua does not. GopherLua also leaves string.dump unimplemented;
// BASIC_OPTS uses it only to compare the values of literal expressions during
// source-equivalence validation, so dumpLiteral returns a canonical encoding
// of that evaluated literal.
func installLua51Compatibility(state *lua.LState) {
	stringLibrary := state.GetGlobal("string").(*lua.LTable)
	state.SetField(stringLibrary, "sub", state.NewFunction(stringSub))
	state.SetField(stringLibrary, "dump", state.NewFunction(dumpLiteral))
}

func stringSub(state *lua.LState) int {
	value := state.CheckString(1)
	start := luaStringIndex(value, checkInteger(state, 2, 0), true)
	end := luaStringIndex(value, checkInteger(state, 3, -1), false)
	if start >= len(value) || end < start {
		state.Push(lua.LString(""))
		return 1
	}
	state.Push(lua.LString(value[start:end]))
	return 1
}

func checkInteger(state *lua.LState, index, defaultValue int) int {
	if index > state.GetTop() {
		return defaultValue
	}

	switch value := state.Get(index).(type) {
	case lua.LNumber:
		return int(value)
	case lua.LString:
		number, err := strconv.ParseFloat(strings.TrimSpace(string(value)), 64)
		if err == nil {
			return int(number)
		}
	}

	state.RaiseError("bad argument #%d to sub (number expected, got %s)", index, state.Get(index).Type())
	return 0
}

func luaStringIndex(value string, index int, start bool) int {
	if start && index != 0 {
		index--
	}
	if index < 0 {
		index = len(value) + index + 1
	}
	if index < 0 {
		index = 0
	}
	if !start && index > len(value) {
		index = len(value)
	}
	return index
}

func dumpLiteral(state *lua.LState) int {
	function := state.CheckFunction(1)
	if err := state.CallByParam(lua.P{Fn: function, NRet: 1, Protect: true}); err != nil {
		state.RaiseError("evaluate literal for source-equivalence validation: %v", err)
		return 0
	}

	value := state.Get(-1)
	state.Pop(1)

	var encoded string
	switch value := value.(type) {
	case *lua.LNilType:
		encoded = "nil"
	case lua.LBool:
		encoded = "boolean:"
		if value {
			encoded += "true"
		} else {
			encoded += "false"
		}
	case lua.LNumber:
		bytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(bytes, math.Float64bits(float64(value)))
		encoded = "number:" + string(bytes)
	case lua.LString:
		encoded = "string:" + string(value)
	default:
		state.RaiseError("unsupported %s literal in source-equivalence validation", value.Type())
		return 0
	}

	state.Push(lua.LString(encoded))
	return 1
}

// Minify removes comments, unnecessary whitespace, and empty lines while
// retaining LuaSrcDiet's source-equivalence validation.
func (minifier *LuaMinifier) Minify(source []byte) (_ []byte, err error) {
	if minifier == nil || minifier.state == nil {
		return nil, fmt.Errorf("LuaSrcDiet: minifier is closed")
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("LuaSrcDiet: runtime panic: %v", recovered)
		}
	}()

	if err := minifier.state.CallByParam(
		lua.P{Fn: minifier.optimize, NRet: 1, Protect: true},
		minifier.options,
		lua.LString(string(source)),
	); err != nil {
		return nil, fmt.Errorf("LuaSrcDiet: %w", err)
	}

	resultValue := minifier.state.Get(-1)
	minifier.state.Pop(1)
	result, ok := resultValue.(lua.LString)
	if !ok {
		return nil, fmt.Errorf("LuaSrcDiet: optimize returned %s", resultValue.Type())
	}
	return []byte(string(result)), nil
}

// Close releases the embedded Lua runtime.
func (minifier *LuaMinifier) Close() {
	if minifier == nil || minifier.state == nil {
		return
	}
	minifier.state.Close()
	minifier.state = nil
}
