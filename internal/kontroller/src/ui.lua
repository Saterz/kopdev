local socket = require("socket")

---@class UI
---@field client Client
local UI = {}
UI.__index = UI

function UI:new(client)
    return setmetatable({ client = client }, self)
end

---@param show_invisible boolean
---@param max_depth boolean
---@return table
function UI:tree(show_invisible, max_depth)
    return self.client:call("ui.tree", { show_invisible = show_invisible, max_depth = max_depth, })
end

function UI:touch_zones()
    return self.client:call("ui.touch_zones")
end

---Find a visible widget by its stable test ID.
---@param id string The widget's `test_id`.
---@return WidgetNode|nil widget The matching serialized UI-tree node, or `nil`.
function UI:find_widget(id)
    return self.client:call("ui.find_widget", { id = id })
end

function UI:wait_until_widget(id, timeout)
    local deadline = socket.gettime() + timeout

    while socket.gettime() < deadline do
        local node = self:find_widget(id)
        if node then
            return true
        end

        --[[ Longer than Control Center's 0.01s accept timeout, so KOReader can
             leave processZMQs(), run the next UIManager tick, then accept again.]]
        socket.sleep(0.02)
    end

    error(string.format("Widget %s not found after %d", id, timeout))
end

return UI
