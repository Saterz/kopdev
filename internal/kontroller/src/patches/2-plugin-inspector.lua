local PluginLoader = require("pluginloader")
local logger = require("logger")

G_PLUGINS_DETAILS = G_PLUGINS_DETAILS or {}

local originalCreatePluginInstance = PluginLoader.createPluginInstance
local originalLoadPlugins = PluginLoader.loadPlugins

PluginLoader.loadPlugins = function(self, ...)
    local enabled, disabled = originalLoadPlugins(self, ...)

    for _, plugin in ipairs(enabled) do
        G_PLUGINS_DETAILS[plugin.name] = {
            fullname = plugin.fullname,
            name = plugin.name,
            path = plugin.path,
            disabled = false,
        }
    end
    for _, plugin in ipairs(disabled) do
        G_PLUGINS_DETAILS[plugin.name] = {
            fullname = plugin.fullname,
            name = plugin.name,
            path = plugin.path,
            disabled = true,
        }
    end

    return enabled, disabled
end

PluginLoader.createPluginInstance = function(self, plugin, attr)
    local status = G_PLUGINS_DETAILS[plugin.name] or {
        name = plugin.name,
        path = plugin.path,
        disabled = false,
    }
    G_PLUGINS_DETAILS[plugin.name] = status

    local ok, result = originalCreatePluginInstance(self, plugin, attr)

    status.status = ok and "loaded" or "failed"
    if ok then
        status.message = nil
    else
        status.message = tostring(result)
    end

    return ok, result
end
