package luasrcdiet

import (
	"embed"
	"fmt"
	"io"
	"os"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

//go:embed src/*.lua
var luaSrcDietFiles embed.FS

func Optimize(source *os.File) (string, error) {
	L := lua.NewState(lua.Options{
		RegistrySize:     1024 * 20,
		RegistryMaxSize:  1024 * 256,
		RegistryGrowStep: 1024 * 20,
	})
	defer L.Close()

	entries, err := luaSrcDietFiles.ReadDir("src")
	if err != nil {
		return "", fmt.Errorf("list LuaSrcDiet modules: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".lua") {
			continue
		}

		filename := "src/" + entry.Name()
		moduleName := "luasrcdiet"
		if entry.Name() != "init.lua" {
			moduleName += "." + strings.TrimSuffix(entry.Name(), ".lua")
		}

		L.PreloadModule(moduleName, func(L *lua.LState) int {
			data, err := luaSrcDietFiles.ReadFile(filename)
			if err != nil {
				L.RaiseError("read %s: %v", filename, err)
				return 0
			}

			chunk, err := L.LoadString(string(data))
			if err != nil {
				L.RaiseError("load %s: %v", filename, err)
				return 0
			}

			// Execute the module and return its table to require().
			L.Push(chunk)
			L.Call(0, 1)
			return 1
		})
	}

	if err := L.CallByParam(lua.P{
		Fn:      L.GetGlobal("require"),
		NRet:    1,
		Protect: true,
	}, lua.LString("luasrcdiet")); err != nil {
		return "", fmt.Errorf("load LuaSrcDiet: %w", err)
	}

	moduleValue := L.Get(-1)
	module, ok := moduleValue.(*lua.LTable)
	if !ok {
		return "", fmt.Errorf(
			"luasrcdiet returned %s, expected table",
			moduleValue.Type(),
		)
	}
	L.Pop(1)

	optimizeFn := L.GetField(module, "optimize")
	opts := L.NewTable()

	L.SetField(opts, "comments", lua.LTrue)
	L.SetField(opts, "emptylines", lua.LTrue)
	L.SetField(opts, "whitespace", lua.LTrue)

	sourceData, err := io.ReadAll(source)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	if err := L.CallByParam(lua.P{
		Fn:      optimizeFn,
		NRet:    1,
		Protect: true,
	}, opts, lua.LString(string(sourceData))); err != nil {
		return "", fmt.Errorf("LuaSrcDiet optimize fail: %w", err)
	}

	result := L.Get(-1)
	minifiedSource, ok := result.(lua.LString)
	if !ok {
		return "", fmt.Errorf(
			"luasrcdiet.optimize returned %s, expected string",
			result.Type(),
		)
	}
	L.Pop(1)

	return string(minifiedSource), nil
}
