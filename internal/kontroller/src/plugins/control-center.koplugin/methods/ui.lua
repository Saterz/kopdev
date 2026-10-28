local UIManager = require("ui/uimanager")
local WidgetTypes = require("widget_types")

local function serializeDimen(dimen)
    if not dimen then return nil end

    return {
        x = dimen.x,
        y = dimen.y,
        w = dimen.w,
        h = dimen.h,
    }
end

local function serializeWidget(widget)
    local fitted_text
    local fitted_text_has_ellipsis
    if type(widget.getFittedText) == 'function' then
        fitted_text, fitted_text_has_ellipsis = widget:getFittedText()
    end

    --
    local test_id = widget.test_id
    if not test_id and type(widget.item) == "table" then
        test_id = widget.item.test_id
    end

    return {
        ref = tostring(widget),
        test_id = test_id,
        id = widget.id,
        name = widget.name,
        type = WidgetTypes.getWidgetType(widget),
        text = widget.text,
        fitted_text = fitted_text,
        fitted_text_has_ellipsis = fitted_text_has_ellipsis,
        enabled = widget.enabled ~= false,
        invisible = widget.invisible == true,
        modal = widget.modal == true,
        toast = widget.toast == true,
        dimen = serializeDimen(widget.dimen),
        child_count = #widget,
    }
end

---@param widget table
---@param visited table
---@param show_invisible boolean
local function serializeWidgetTree(widget, visited, show_invisible, depth, max_depth)
    if not show_invisible then
        if widget.invisible then
            return nil
        end
    end

    if visited[widget] then
        return {
            ref = tostring(widget),
            repeated = true
        }
    end

    visited[widget] = true

    local node = serializeWidget(widget)
    node.ref = tostring(widget)
    node.depth = depth
    node.children = {}

    if max_depth ~= nil and depth >= max_depth then
        node.truncated = #widget > 0
        return node
    end

    for _, child in ipairs(widget) do
        local serialized_child = serializeWidgetTree(child, visited, show_invisible, depth + 1, max_depth)

        if serialized_child then
            table.insert(node.children, serialized_child)
        end
    end

    return node
end

local function getUITree(show_invisible, max_depth)
    local tree = {}
    local visited = {}

    for i = #UIManager._window_stack, 1, -1 do
        local entry = UIManager._window_stack[i]
        local root = serializeWidgetTree(entry.widget, visited, show_invisible, 0, max_depth)

        if root then
            root.pos = {
                x = entry.x,
                y = entry.y
            }

            table.insert(tree, root)
        end
    end

    return tree
end

local function findWidget(tree, widget_test_id)
    local function search(node)
        if node.test_id == widget_test_id then
            return node
        end

        for _, child in ipairs(node.children or {}) do
            local result = search(child)
            if result then
                return result
            end
        end

        return nil
    end

    for _, root in ipairs(tree) do
        local result = search(root)
        if result then
            return result
        end
    end

    return nil
end

return {
    ---@param params {show_invisible: boolean, max_depth: boolean}
    ["ui.tree"] = function(params)
        return getUITree(params.show_invisible, params.max_depth)
    end,
    --- Finds the topmost visible widget with the given test ID.
    ---@param params {id: string} The widget's test_id.
    ---@return {test_id: string|nil, id: string|nil, name: string|nil, text: string|nil, enabled: boolean, invisible: boolean, modal: boolean, toast: boolean, dimen: {x: number, y: number, w: number, h: number}|nil, child_count: number}|nil The widget metadata, if found.
    ["ui.find_widget"] = function(params)
        return findWidget(getUITree(false), params.id)
    end,
    --- Returns touch zones for the topmost visible widget.
    ---@return table[] The touch zones.
    ["ui.touch_zones"] = function()
        local widget = UIManager:getTopmostVisibleWidget()
        local zones = {}
        if not widget or not widget._ordered_touch_zones then
            return zones
        end

        for _, zone in ipairs(widget._ordered_touch_zones) do
            local range = zone.gs_range.range
            table.insert(zones, {
                id = zone.def.id,
                ges = zone.gs_range.ges,
                range = range and {
                    x = range.x,
                    y = range.y,
                    w = range.w,
                    h = range.h,
                } or nil,
            })
        end
        return zones
    end,
}
