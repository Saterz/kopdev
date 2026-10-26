--[[
This is a KOReader template plugin.

@module koplugin.TEMP_PLUGIN_NAME_UPPER
]]

local Dispatcher = require("dispatcher")
local InfoMessage = require("ui/widget/infomessage")
local UIManager = require("ui/uimanager")
local WidgetContainer = require("ui/widget/container/widgetcontainer")
local _ = require("gettext")

local TEMP_PLUGIN_NAME_UPPER = WidgetContainer:extend {
    name = "TEMP_PLUGIN_NAME_LOWER",
    is_doc_only = false,
}

function TEMP_PLUGIN_NAME_UPPER:onDispatcherRegisterActions()
    Dispatcher:registerAction("TEMP_PLUGIN_NAME_LOWER_action",
        { category = "none", event = "TEMP_PLUGIN_NAME_UPPER", title = _("TEMP_PLUGIN_NAME_UPPER"), general = true, })
end

function TEMP_PLUGIN_NAME_UPPER:init()
    self:onDispatcherRegisterActions()
    self.ui.menu:registerToMainMenu(self)
end

function TEMP_PLUGIN_NAME_UPPER:addToMainMenu(menu_items)
    menu_items["TEMP_PLUGIN_NAME_LOWER"] = {
        text = _("TEMP_PLUGIN_NAME_UPPER"),
        -- in which menu this should be appended
        sorting_hint = "more_tools",
        -- a callback when tapping
        callback = function()
            UIManager:show(InfoMessage:new {
                text = _("Hello from TEMP_PLUGIN_NAME_UPPER"),
            })
        end,
    }
end

function TEMP_PLUGIN_NAME_UPPER:onTEMP_PLUGIN_NAME_UPPER()
    local popup = InfoMessage:new {
        text = _("TEMP_PLUGIN_NAME_UPPER"),
    }
    UIManager:show(popup)
end

-- KOReader will call this function when users choose to delete the plugin's settings.
function TEMP_PLUGIN_NAME_UPPER:deletePluginSettings()
    return true
end

return TEMP_PLUGIN_NAME_UPPER
