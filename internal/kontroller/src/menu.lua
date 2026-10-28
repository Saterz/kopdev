---@class Menu
---@field client Client
local Menu = {}
Menu.__index = Menu

function Menu:new(client)
    return setmetatable({ client = client }, self)
end

---Activate a menu entry using its dot-separated numerical path.
---@param path table A path such as {4,2,3}.
---@return boolean result Selection confirmation.
function Menu:select_entry(path)
    return self.client:call("menu.select", {
        path = path,
    })
end

return Menu
