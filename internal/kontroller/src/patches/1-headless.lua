package.preload["device"] = function()
    local Device = require("device/dummy/device")

    -- The dummy device uses framebuffer_SDL3, but does not identify
    -- itself as SDL. Without this, Input falls back to
    -- libs/libkoreader-input, which is unavailable in emulator builds.
    Device.isSDL = function()
        return true
    end
    -- Enable touch-only UI controls
    Device.isTouchDevice = function()
        return true
    end

    Device:init()
    return Device
end
