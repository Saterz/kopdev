local UIManager = require("ui/uimanager")
local Event = require("ui/event")

local M = {}

---@param func function
function M.defer(func)
    UIManager:nextTick(func)
end

---@param name string
function M:deferEvent(name)
    self.defer(function()
        UIManager:broadcastEvent(
            Event:new(name)
        )
    end)
end

return M
