local Event = require("ui/event")
local JSON = require("json")
local logger = require("logger")
local socket = require("socket")

local MAX_BODY_SIZE = 1024 * 1024
local MAX_HEADER_SIZE = 16 * 1024

local HTTP_STATUS = {
    [200] = "OK",
    [204] = "No Content",
    [400] = "Bad Request",
    [404] = "Not Found",
    [405] = "Method Not Allowed",
    [411] = "Length Required",
    [413] = "Payload Too Large",
    [500] = "Internal Server Error",
}

local Server = {}

function Server:new(o)
    o = o or {}
    setmetatable(o, self)
    self.__index = self
    return o
end

function Server:start()
    local listener, err = socket.bind(self.host, self.port)
    if not listener then
        return nil, err
    end
    listener:settimeout(0.01)
    self.listener = listener
    return true
end

function Server:stop()
    if self.listener then
        self.listener:close()
        self.listener = nil
    end
end

function Server:sendResponse(client, status, body, content_type)
    body = body or ""
    content_type = content_type or "application/json; charset=utf-8"
    local headers = {
        string.format("HTTP/1.0 %d %s", status, HTTP_STATUS[status] or "Unknown"),
        string.format("Content-Type: %s", content_type),
        string.format("Content-Length: %d", #body),
        "Connection: close",
        "",
        body,
    }
    client:send(table.concat(headers, "\r\n"))
    client:close()
    logger.info("ControlCenter: Response sent")
end

function Server:readRequest(client)
    client:settimeout(0.5, "t")

    local request_line, err = client:receive("*l")
    if not request_line then
        return nil, 400, err
    end
    local method, target = request_line:match("^(%u+) ([^ ]+) HTTP/%d%.%d$")
    if not method then
        return nil, 400, "invalid request line"
    end

    local headers = {}
    local header_size = #request_line
    while true do
        local line
        line, err = client:receive("*l")
        if not line then
            return nil, 400, err
        end
        header_size = header_size + #line
        if header_size > MAX_HEADER_SIZE then
            return nil, 400, "headers too large"
        end
        if line == "" then
            break
        end
        local name, value = line:match("^([^:]+):%s*(.*)$")
        if not name then
            return nil, 400, "invalid header"
        end
        headers[name:lower()] = value
    end

    local body = ""
    if method == "POST" then
        local content_length = tonumber(headers["content-length"])
        if not content_length or content_length < 0 or content_length % 1 ~= 0 then
            return nil, 411, "a valid Content-Length is required"
        end
        if content_length > MAX_BODY_SIZE then
            return nil, 413, "request body is too large"
        end

        local receive_err, partial
        body, receive_err, partial = client:receive(content_length)
        if not body then
            logger.warn("ControlCenter: incomplete HTTP request body:", receive_err)
            return nil, 400, partial
        end
    end
    return { method = method, target = target, body = body }
end

local function errorResponse(code, message, id)
    return {
        jsonrpc = "2.0",
        error = {
            code = code,
            message = message,
        },
        id = id or JSON.util.null,
    }
end

function Server:dispatch(request)
    if type(request) ~= "table" or JSON.util.IsArray(request) then
        return errorResponse(-32600, "Invalid Request")
    end

    local has_id = rawget(request, "id") ~= nil
    local id_type = type(request.id)
    local valid_id = not has_id or id_type == "string" or id_type == "number"
        or request.id == JSON.util.null
    if request.jsonrpc ~= "2.0"
        or type(request.method) ~= "string"
        or (request.params ~= nil and type(request.params) ~= "table")
        or not valid_id then
        return errorResponse(-32600, "Invalid Request",
            has_id and valid_id and request.id or nil)
    end

    local handler = self.methods[request.method]
    if not handler then
        if not has_id then
            return nil
        end
        return errorResponse(-32601, "Method not found", request.id)
    end

    local ok, result = pcall(handler, request.params)
    if not ok then
        logger.err("ControlCenter: JSON-RPC method failed:", request.method, result)
        if not has_id then
            return nil
        end
        return errorResponse(-32603, "Internal error", request.id)
    end
    if not has_id then
        return nil
    end
    return {
        jsonrpc = "2.0",
        result = result == nil and JSON.util.null or result,
        id = request.id,
    }
end

function Server:handleJsonRpc(body)
    local ok, request = pcall(JSON.decode, body)
    if not ok then
        return JSON.encode(errorResponse(-32700, "Parse error"))
    end

    local response
    if JSON.util.IsArray(request) then
        if #request == 0 then
            response = errorResponse(-32600, "Invalid Request")
        else
            response = JSON.util.InitArray({})
            for _, item in ipairs(request) do
                local item_response = self:dispatch(item)
                if item_response then
                    table.insert(response, item_response)
                end
            end
            if #response == 0 then
                return nil
            end
        end
    else
        response = self:dispatch(request)
        if not response then
            return nil
        end
    end

    local encode_ok, encoded = pcall(JSON.encode, response)
    if not encode_ok then
        local encode_err = encoded
        logger.err("ControlCenter: failed to encode JSON-RPC response:", encode_err)
        return JSON.encode(errorResponse(-32603, "Internal error"))
    end
    return encoded
end

function Server:handleScreenshot(query)
    local Screenshot = require("screenshot")
    local screenshot = Screenshot:new()
    local ok, screenshotData = pcall(screenshot.takeScreenshot, screenshot, query)
    if not ok then
        logger.err("ControlCenter: failed to take screenshot:", screenshotData)
        return nil
    end

    return screenshotData
end

function Server:waitEvent()
    local client = self.listener and self.listener:accept()
    if not client then
        return
    end

    local request, http_status, err = self:readRequest(client)
    logger.info("ControlCenter: JSON-RPC request:", request)
    if not request then
        local response = JSON.encode({ error = err or HTTP_STATUS[http_status] })
        self:sendResponse(client, http_status, response)
        return Event:new("InputEvent")
    end

    local path, query = request.target:match("^([^?]+)%??(.*)$")

    local response
    local content_type
    if request.method == "GET" and path == "/screenshot" then
        response = self:handleScreenshot(query)
        content_type = "image/png"
    elseif request.method == "POST" and request.target == "/jsonrpc" then
        response = self:handleJsonRpc(request.body)
        logger.dbg("ControlCenter: JSON-RPC response:", response)
        content_type = "application/json; charset=utf-8"
    else
        response = JSON.encode({ error = "Not found" })
    end

    self:sendResponse(client, response and 200 or 204, response, content_type)
    return Event:new("InputEvent")
end

return Server
