--[[--


@module koplugin.ControlCenter
--]] --
local Server = require("server")
local Methods = require("methods/init")
local UIManager = require("ui/uimanager")
local WidgetContainer = require("ui/widget/container/widgetcontainer")
local logger = require("logger")

local ControlCenter = WidgetContainer:extend {
    name = "control_center",
    is_doc_only = false,
    host = "127.0.0.1",
    port = 8060,
}

function ControlCenter:init()
    self.methods = Methods
    self:startServer()
end

function ControlCenter:startServer()
    if self.server then
        return true
    end

    local server = Server:new {
        host = self.host,
        port = self.port,
        methods = self.methods,
    }
    local ok, err = server:start()
    if not ok then
        logger.err("ControlCenter: failed to start JSON-RPC server:", err)
        return nil, err
    end

    self.server = server
    self.server_event_source = UIManager:insertZMQ(server)
    logger.info(string.format(
        "ControlCenter: JSON-RPC server listening on %s:%d",
        self.host, self.port))
    return true
end

function ControlCenter:stopServer()
    if self.server_event_source then
        UIManager:removeZMQ(self.server_event_source)
        self.server_event_source = nil
    end
    if self.server then
        self.server:stop()
        self.server = nil
    end
end

function ControlCenter:onSuspend()
    self:stopServer()
end

function ControlCenter:onEnterStandby()
    self:stopServer()
end

function ControlCenter:onResume()
    self:startServer()
end

function ControlCenter:onLeaveStandby()
    self:startServer()
end

function ControlCenter:onExit()
    self:stopServer()
end

function ControlCenter:onCloseWidget()
    self:stopServer()
end

return ControlCenter
