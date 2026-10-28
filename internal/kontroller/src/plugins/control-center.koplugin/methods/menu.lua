local UIManager = require("ui/uimanager")
local Event = require("ui/event")

local function openAndGetMenu()
    local touch_menu
    for attempt = 1, 2 do
        local menu_container = UIManager:getTopmostVisibleWidget()
        touch_menu = menu_container and menu_container[1]

        if touch_menu and type(touch_menu.onMenuSelect) == "function" then
            return touch_menu
        end

        if attempt == 1 then
            UIManager:broadcastEvent(Event:new("ShowMenu"))
        else
            error("the opened widget is not a TouchMenu")
        end
    end
end

local function activateItemAtPath(touch_menu, path)
    if type(path) ~= "table" then
        error("invalid menu path: " .. tostring(path))
    end

    local tab_index = table.remove(path, 1)
    if not touch_menu.tab_item_table[tab_index] then
        error("menu tab does not exist at path: " .. tostring(path))
    end

    -- Switching through the bar follows the same code path as selecting a tab.
    touch_menu.bar:switchToTab(tab_index)

    for depth, index in ipairs(path) do
        local item = touch_menu.item_table[index]
        if not item then
            error("menu item does not exist at path: " .. path)
        end

        local enabled = item.enabled
        if item.enabled_func then
            enabled = item.enabled_func()
        end
        if enabled == false then
            error("menu item is disabled at path: " .. path)
        end

        if depth < #path then
            local sub_items = item.sub_item_table_func
                and item.sub_item_table_func()
                or item.sub_item_table

            if not sub_items or #sub_items == 0 then
                error("menu path enters an item without a submenu: " .. path)
            end
        end

        -- Intermediate selections enter submenus; the final selection invokes
        -- the target entry's normal tap action.
        touch_menu:onMenuSelect(item)
    end
end

return {
    --- Activates the menu entry at the given numerical path.
    ---@param params {path: table} The dot-separated menu path.
    ---@return boolean success The activated menu entry.
    ["menu.select"] = function(params)
        local touch_menu = openAndGetMenu()
        activateItemAtPath(touch_menu, params.path)
        UIManager:forceRePaint()
        return true
    end,
    --- Opens the menu.
    ---@return boolean
    ["menu.open"] = function()
        UIManager:broadcastEvent(Event:new("ShowMenu"))
        return true
    end,
}
