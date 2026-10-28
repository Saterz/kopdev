local groups = {
    require("methods.system"),
    require("methods.screen"),
    require("methods.ui"),
    require("methods.filemanager"),
    require("methods.reader"),
    require("methods.menu")
}

local methods = {}

for _, group in ipairs(groups) do
    for name, handler in pairs(group) do
        assert(methods[name] == nil, "Duplicate RPC method" .. name)
        methods[name] = handler
    end
end

return methods
