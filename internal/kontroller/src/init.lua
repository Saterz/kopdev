local Client = require("kontroller.client")
local pl = require('pl.import_into')()

local Widget = require("kontroller.widget")
local Screen = require("kontroller.screen")
local System = require("kontroller.system")
local UI = require("kontroller.ui")
local FileManager = require("kontroller.filemanager")
local Menu = require("kontroller.menu")

---@class Kontroller
---@field screen Screen
---@field system System
---@field ui UI
---@field private client Client
---@field private run_dir string?
local Kontroller = {}
Kontroller.__index = Kontroller

---@return Kontroller
function Kontroller:new()
    local host = os.getenv("KONTROLLER_HOST") or "127.0.0.1"
    local port = tonumber(os.getenv("KONTROLLER_PORT")) or 8060
    local client = Client:new(host, port)

    return setmetatable({
        plugin_name = os.getenv("PLUGIN_NAME"),
        run_dir = os.getenv("RUN_DIR"),
        client = client,
        screen = Screen:new(client),
        system = System:new(client),
        ui = UI:new(client),
        file_manager = FileManager:new(client),
        menu = Menu:new(client)
    }, self)
end

---@return string `pong`
function Kontroller:ping()
    return self.client:call("ping")
end

---@param id string The widget's `test_id`.
---@return Widget|nil widget
function Kontroller:widget(id)
    return Widget:new(self, id)
end

---Opens the book at the specified path
---@param path string the path to the book
---@return boolean `true` if the file was opened
function Kontroller:open_book(path)
    if not pl.path.isabs(path) then
        path = pl.path.abspath(path, self.run_dir)
    end
    self.client:call("filemanager.open", { path = path })
    self.system:wait_until_ready()
    local state = self.client:call("reader.state")
    return state.open
end

---Closes the currently opened book
---@return boolean `true` if returned to the file manager
function Kontroller:close_book()
    return self.client:call("reader.close")
end

---@param setup function
function Kontroller:set_startup_hook(setup)
    assert(type(setup) == "function", "setup must be a function")

    local path = assert(self.run_dir) .. "/patches/1-startup-hook.lua"
    local file = assert(io.open(path, "wb"))
    assert(file:write(string.dump(setup)))
    assert(file:close())
end

function Kontroller:clear_startup_hook()
    os.remove(assert(self.run_dir) .. "/patches/1-startup-hook.lua")
end

return Kontroller
