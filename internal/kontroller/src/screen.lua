---@class Screen
---@field client Client
local Screen = {}
Screen.__index = Screen

function Screen:new(client)
    return setmetatable({ client = client }, self)
end

---@param x number
---@param y number
---@return boolean
function Screen:tap(x, y)
    return self.client:call("screen.tap", { x = x, y = y })
end

---@param x number
---@param y number
---@return boolean
function Screen:hold(x, y)
    return self.client:call("screen.hold", { x = x, y = y })
end

---@param x number
---@param y number
---@param direction Direction
---@param distance number
function Screen:swipe(x, y, direction, distance)
    return self.client:call("screen.swipe", { x = x, y = y, direction = direction, distance = distance })
end

---@param x number
---@param y number
---@param directions Direction[]
---@param distance number
function Screen:multi_swipe(x, y, directions, distance)
    return self.client:call("screen.multiswipe", { x = x, y = y, directions = directions, distance = distance })
end

---@return string PNG
function Screen:shot()
    return self.client:get_screen()
end

return Screen
