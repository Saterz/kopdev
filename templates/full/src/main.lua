--[[--
This is a debug plugin to test Plugin functionality.

@module koplugin.{{PLUGIN_NAME_UPPER_FIRST}}
--]] --

-- This is a debug plugin, remove the following if block to enable it
if true then
    return { disabled = true, }
end

local Dispatcher = require("dispatcher") -- luacheck:ignore
local InfoMessage = require("ui/widget/infomessage")
local UIManager = require("ui/uimanager")
local WidgetContainer = require("ui/widget/container/widgetcontainer")
local _ = require("gettext")

local {{PLUGIN_NAME_UPPER_FIRST}} = WidgetContainer:extend {
    name = "{{PLUGIN_NAME_LOWER_FIRST}}",
    is_doc_only = false,
}

function {{PLUGIN_NAME_UPPER_FIRST}}:onDispatcherRegisterActions()
    Dispatcher:registerAction("{{PLUGIN_NAME_LOWER_FIRST}}_action",
        { category = "none", event = "{{PLUGIN_NAME_UPPER_FIRST}}", title = _("{{PLUGIN_NAME_UPPER_FIRST}}"), general = true, })
end

function {{PLUGIN_NAME_UPPER_FIRST}}:init()
    self:onDispatcherRegisterActions()
    self.ui.menu:registerToMainMenu(self)
end

function {{PLUGIN_NAME_UPPER_FIRST}}:addToMainMenu(menu_items)
    menu_items["{{PLUGIN_NAME_LOWER_FIRST}}"] = {
        text = _("{{PLUGIN_NAME_UPPER_FIRST}}"),
        -- in which menu this should be appended
        sorting_hint = "more_tools",
        -- a callback when tapping
        callback = function()
            UIManager:show(InfoMessage:new {
                text = _("Hello from {{PLUGIN_NAME_UPPER_FIRST}}"),
            })
        end,
    }
end

function {{PLUGIN_NAME_UPPER_FIRST}}:on{{PLUGIN_NAME_UPPER_FIRST}}()
    local popup = InfoMessage:new {
        text = _("{{PLUGIN_NAME_UPPER_FIRST}}"),
    }
    UIManager:show(popup)
end

return {{PLUGIN_NAME_UPPER_FIRST}}
