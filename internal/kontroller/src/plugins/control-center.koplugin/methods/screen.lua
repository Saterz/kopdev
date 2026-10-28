local UIManager = require("ui/uimanager")
local Event = require("ui/event")
local Geom = require("ui/geometry")
local Screen = require("device").screen

local diagonal = 1 / math.sqrt(2)

local direction_offsets = {
    north = { x = 0, y = -1 },
    east = { x = 1, y = 0 },
    south = { x = 0, y = 1 },
    west = { x = -1, y = 0 },

    northeast = { x = diagonal, y = -diagonal },
    northwest = { x = -diagonal, y = -diagonal },
    southeast = { x = diagonal, y = diagonal },
    southwest = { x = -diagonal, y = diagonal },
}

return {
    --- Sends a tap gesture at the given screen coordinates.
    ---@param params {x: number, y: number} The tap coordinates.
    ---@return nil
    ["screen.tap"] = function(params)
        UIManager:handleInputEvent(Event:new("Gesture", {
            ges = "tap",
            pos = Geom:new({ x = params.x, y = params.y, w = 0, h = 0 })
        }))
        return true
    end,
    --- Sends a hold gesture at the given screen coordinates.
    ---@param params {x: number, y: number} The hold coordinates.
    ---@return nil
    ["screen.hold"] = function(params)
        UIManager:handleInputEvent(Event:new("Gesture", {
            ges = "hold",
            pos = Geom:new({ x = params.x, y = params.y, w = 0, h = 0 })
        }))
        return true
    end,
    --- Sends a swipe gesture from the given coordinates.
    ---@param params {x: number, y: number, direction: Direction, distance: number} The swipe origin, direction, and distance.
    ---@return boolean Whether the gesture was sent.
    ["screen.swipe"] = function(params)
        local offset = assert(direction_offsets[params.direction], "invalid swipe direction")
        local pos = Geom:new({ x = params.x, y = params.y, w = 0, h = 0 })

        local gesture = {
            ges = "swipe",
            pos = pos,
            direction = params.direction,
        }

        if params.distance ~= nil then
            gesture["distance"] = params.distance
            gesture["end_pos"] = Geom:new({
                x = pos.x + offset.x * params.distance,
                y = pos.y + offset.y * params.distance,
                w = 0,
                h = 0,
            })
        end

        UIManager:handleInputEvent(Event:new("Gesture", gesture))
        return true
    end,
    ---@param params {x: number, y: number, directions: Direction[], distance: number}
    ---@returns boolean
    ["screen.multiswipe"] = function(params)
        if type(params.directions) ~= "table" then
            error("Direction must be a table")
        end
        if #params.directions < 2 then
            error("A minimum of two directions for a multi-swipe")
        end

        for index, direction in params.directions do
            assert(
                direction_offsets[direction] ~= nil,
                "invalid direction at index " .. index .. ": " .. direction
            )
        end

        UIManager:handleInputEvent(Event:new("Gesture", {
            ges = "multiswipe",
            pos = Geom:new({ x = params.x, y = params.y, w = 0, h = 0 }),
            multiswipe_directions = table.concat(params.directions, " "),
        }))

        return true
    end,
    --- Returns the screen dimensions.
    ---@return {width: number, height: number} The screen dimensions.
    ["screen.size"] = function()
        return {
            width = Screen:getWidth(),
            height = Screen:getHeight(),
        }
    end,
}
