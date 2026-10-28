---@class Widget
---@field node WidgetNode
---@field kontroller Kontroller
local Widget = {}
Widget.__index = Widget

---@param node WidgetNode
local function get_widget_text(node)
    if node.type == "TextWidget" then
        return node.text or ""
    end
    local parts = {}
    for _, child in ipairs(node.children or {}) do
        table.insert(parts, get_widget_text(child))
    end
    return table.concat(parts, "\n")
end

---@param kontroller Kontroller
---@param id string
---@return Widget
function Widget:new(kontroller, id)
    local node = kontroller.ui:find_widget(id)
    if node == nil then
        error(string.format("No widget with the id \"%s\" was found", id))
    end
    return setmetatable({ kontroller = kontroller, node = node }, self)
end

function Widget:tap()
    self.kontroller.screen:tap(
        self.node.dimen.x + math.floor(self.node.dimen.w / 2),
        self.node.dimen.y + math.floor(self.node.dimen.h / 2)
    )
end

function Widget:hold()
    self.kontroller.screen:hold(
        self.node.dimen.x + math.floor(self.node.dimen.w / 2),
        self.node.dimen.y + math.floor(self.node.dimen.h / 2)
    )
end

function Widget:get_text()
    return get_widget_text(self.node)
end

function Widget:is_truncated()
    return self.node.fitted_text_has_ellipsis
end

return Widget
