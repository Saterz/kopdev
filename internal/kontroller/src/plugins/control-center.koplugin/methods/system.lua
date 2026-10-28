local UIManager = require("ui/uimanager")
local Event = require("ui/event")
local logger = require("logger")
local Version = require("version")
local common = require("methods.controller_common")

return {
    ---Confirms that the control center is reachable.
    ---@return string "pong"
    ["ping"] = function()
        return "pong"
    end,
    ---Gets the status of the named plugin.
    ---@param params {name: string} The plugin name.
    ---@return table|nil _ The plugin details, if it is registered.
    ["plugin.status"] = function(params)
        return G_PLUGINS_DETAILS[params.name]
    end,
    ---Broadcasts the named system event.
    ---@param params {name: string} The event name.
    ---@return boolean Whether the event was scheduled.
    ["system.event"] = function(params)
        common:deferEvent(params.name)
        return true
    end,
    ---Returns the current KOReader revision.
    ---@return string The current revision.
    ["system.version"] = function()
        return Version:getCurrentRevision()
    end,
    ---Suspends KOReader and resumes it after the given duration.
    ---@param params {seconds: number} The duration before resuming.
    ---@return boolean Whether suspension was scheduled.
    ["system.suspend_for"] = function(params)
        UIManager:scheduleIn(params.seconds, function()
            UIManager.event_handlers.Resume()
        end)

        UIManager:nextTick(function()
            UIManager:broadcastEvent(Event:new("RequestSuspend"))
        end)
        return true
    end,
    ---Requests a KOReader restart.
    ---@return boolean Whether the restart was scheduled.
    ["system.restart"] = function()
        common:deferEvent("Restart")
        return true
    end,
    ---Requests a KOReader exit.
    ---@return boolean Whether the exit was scheduled.
    ["system.exit"] = function()
        common:deferEvent("Exit")
        return true
    end,
}
