local socket = require("socket")
local JSON = require("dkjson")

---@class Client
---@field host string
---@field port number
---@field next_id number
local Client = {}
Client.__index = Client

---Creates a new Client object
---@param host string|nil
---@param port number|nil
---@return Client
function Client:new(host, port)
    return setmetatable({
        host = host or "127.0.0.1",
        port = port or 8060,
        next_id = 1,
    }, self)
end

---Makes a JSON-RPC request to the control center plugin
---@param method string The method name to call
---@param params any|nil Its optional arguments
---@return any
function Client:call(method, params)
    local id = self.next_id
    self.next_id = id + 1

    local body = JSON.encode({
        jsonrpc = "2.0",
        method = method,
        params = params,
        id = id,
    })

    local request = table.concat({
        "POST /jsonrpc HTTP/1.0",
        "Host: " .. self.host,
        "Content-Type: application/json",
        "Content-Length: " .. #body,
        "Connection: close",
        "",
        body,
    }, "\r\n")

    local tcp, connect_error = socket.connect(self.host, self.port)
    assert(tcp, "Could not connect: " .. tostring(connect_error))
    tcp:settimeout(5)

    local sent, send_error = tcp:send(request)
    if not sent then
        tcp:close()
        error("Could not send request: " .. tostring(send_error))
    end

    local response, receive_error, partial = tcp:receive("*a")
    tcp:close()

    -- LuaSocket may report "closed" while returning the response as partial data.
    response = response or partial
    if not response or response == "" then
        error("Could not receive response: " .. tostring(receive_error))
    end

    local headers, response_body =
        response:match("^(.-)\r\n\r\n(.*)$")

    if not headers then
        error("Malformed HTTP response")
    end

    local status = tonumber(headers:match("^HTTP/%d%.%d%s+(%d+)"))
    if status ~= 200 then
        error(string.format(
            "Control Center returned HTTP %s: %s",
            tostring(status),
            response_body
        ))
    end

    local decoded = JSON.decode(response_body)

    if decoded == nil then
        error("Received an empty response")
    end

    if decoded.error then
        error(string.format(
            "JSON-RPC error %s: %s",
            tostring(decoded.error.code),
            tostring(decoded.error.message)
        ))
    end

    return decoded.result
end

function Client:get_screen()
    local request = table.concat({ "GET /screenshot HTTP/1.0", "Host: " .. self.host, "Connection: close", "", "" },
        "\r\n")

    local tcp, connect_error = socket.connect(self.host, self.port)
    assert(tcp, "Could not connect: " .. tostring(connect_error))
    tcp:settimeout(5)

    local sent, send_error = tcp:send(request)
    if not sent then
        tcp:close()
        error("Could not send request: " .. tostring(send_error))
    end

    local response, receive_error, partial = tcp:receive("*a")
    tcp:close()

    -- LuaSocket may report "closed" while returning the response as partial data.
    response = response or partial
    if not response or response == "" then
        error("Could not receive response: " .. tostring(receive_error))
    end

    local headers, response_body =
        response:match("^(.-)\r\n\r\n(.*)$")

    if not headers then
        error("Malformed HTTP response")
    end

    local status = tonumber(headers:match("^HTTP/%d%.%d%s+(%d+)"))
    if status ~= 200 then
        error(string.format(
            "Control Center returned HTTP %s: %s",
            tostring(status),
            response_body
        ))
    end

    local content_type = headers:lower():match(
        "\r\ncontent%-type:%s*([^;\r\n]+)"
    )

    if content_type ~= "image/png" then
        error("Expected image/png, received " .. tostring(content_type))
    end

    if response_body:sub(1, 8) ~= "\137PNG\r\n\26\n" then
        error("Response is not a valid PNG")
    end

    return response_body
end

return Client
