local socket = require("socket")

---@class System
---@field client Client
---@field plugin_name string
local System = {}
System.__index = System

function System:new(client, plugin_name)
    return setmetatable({ client = client, plugin_name = plugin_name }, self)
end

function System:wait_until_ready(timeout)
    local deadline = socket.gettime() + (timeout or 30)
    local became_unavailable = false

    while socket.gettime() < deadline do
        local available, response = pcall(function()
            return self.client:call("ping")
        end)

        if not available then
            became_unavailable = true
        elseif became_unavailable and response == "pong" then
            return true
        end

        socket.sleep(0.1)
    end

    error(string.format(
        "KOReader did not restart within %.1f seconds",
        timeout
    ))
end

---Restart KOReader and wait for the Control Center to become available again.
---@param timeout number|nil Maximum seconds to wait; defaults to 30.
---@return boolean restarted `true` after the restarted server responds.
function System:restart(timeout)
    timeout = timeout or 30
    if timeout <= 0 then
        error("restart timeout must be greater than zero")
    end

    local scheduled = self.client:call("system.restart")
    if scheduled ~= true then
        error("KOReader did not accept the restart request")
    end

    return self:wait_until_ready(timeout)
end

---@param seconds number
---@param timeout number|nil Maximum seconds to wait; defaults to 30.
---@return boolean
function System:suspend_for(seconds, timeout)
    self.client:call("system.suspend_for", { seconds = seconds })

    return self:wait_until_ready(timeout)
end

function System:version()
    return self.client:call("system.version")
end

---@param name? string
function System:plugin_status(name)
    return self.client:call("plugin.status", { name = name or self.plugin_name })
end

---@param name string
function System:plugin_loaded(name)
    local status = self:plugin_status(name)
    if not status then
        return false
    elseif status.status == "failed" or status.disabled then
        return false
    end
    return true
end

return System
