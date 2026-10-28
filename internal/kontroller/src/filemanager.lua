---@class FileManager
---@field client Client
local FileManager = {}
FileManager.__index = FileManager

function FileManager:new(client)
    return setmetatable({ client = client }, self)
end

function FileManager:state()
    return self.client:call("filemanager.state")
end

function FileManager:get_current_path()
    return self:state().path
end

return FileManager
