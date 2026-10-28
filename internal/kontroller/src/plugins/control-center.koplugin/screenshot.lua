local DataStorage = require("datastorage")
local Screen = require("device").screen
local UIManager = require("ui/uimanager")
local Blitbuffer = require("ffi/blitbuffer")
local Font = require("ui/font")
local TextWidget = require("ui/widget/textwidget")

local ROWS = 6
local COLUMNS = 4

local Screenshot = {}

function Screenshot:new(obj)
    obj = obj or {}
    obj = setmetatable(obj, self)
    self.__index = self
    return obj
end

local function paintLabel(bb, x, y, text)
    local label = TextWidget:new {
        text = text,
        face = Font:getFace("cfont", 14),
        padding = 0,
    }
    local size = label:getSize()

    x = math.max(2, math.min(x - math.floor(size.w / 2), bb:getWidth() - size.w - 2))
    y = math.max(2, math.min(y - math.floor(size.h / 2), bb:getHeight() - size.h - 2))

    bb:paintRect(x - 2, y - 2, size.w + 4, size.h + 4, Blitbuffer.COLOR_WHITE)
    label:paintTo(bb, x, y)
    label:free()
end

local function paintCoordinateOverlay(bb)
    local w, h = bb:getWidth(), bb:getHeight()
    for row = 0, ROWS do
        for column = 0, COLUMNS do
            local x = math.floor(w * column / COLUMNS)
            local y = math.floor(h * row / ROWS)
            bb:paintRect(math.max(0, x - ROWS), y, 9, 1, Blitbuffer.COLOR_BLACK)
            bb:paintRect(x, math.max(0, y - COLUMNS), 1, 9, Blitbuffer.COLOR_BLACK)
            paintLabel(bb, x, y + 14, string.format("(%d, %d)", x, y))
        end
    end
end

function Screenshot:takeScreenshot(query)
    local tmpScreenshot = DataStorage:getDataDir() .. "/cache/control_center_screenshot.png"
    local coordinates = query == "overlay=coordinates"
    local bb, original

    UIManager:forceRePaint()

    if coordinates then
        bb = Screen.bb
        original = Blitbuffer.new(bb:getWidth(), bb:getHeight(), bb:getType())
        original:blitFrom(bb, 0, 0, 0, 0, bb:getWidth(), bb:getHeight())

        paintCoordinateOverlay(bb)
    end

    Screen:shot(tmpScreenshot)

    if coordinates then
        bb:blitFrom(original, 0, 0, 0, 0, bb:getWidth(), bb:getHeight())
        original:free()
    end

    local file, err = io.open(tmpScreenshot, "rb")
    if not file then
        error("Could not read screenshot: " .. tostring(err))
    end

    local screenshotData = file:read("*all")
    file:close()
    os.remove(tmpScreenshot)

    return screenshotData
end

return Screenshot
