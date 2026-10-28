local common = require("methods.controller_common")

local function getReaderInstance()
    local ReaderUI = require("apps/reader/readerui")
    return ReaderUI.instance
end

---Return serializable information about the active reader.
---@return table state
local function getReaderState()
    local reader = getReaderInstance()

    if not reader or not reader.document then
        return {
            open = false,
        }
    end

    local page
    local page_count

    if type(reader.getCurrentPage) == "function" then
        local succeeded, result =
            pcall(reader.getCurrentPage, reader)

        if succeeded then
            page = result
        end
    end

    if type(reader.document.getPageCount) == "function" then
        local succeeded, result =
            pcall(reader.document.getPageCount, reader.document)

        if succeeded then
            page_count = result
        end
    end

    return {
        open = true,
        file = reader.document.file,
        page = page,
        page_count = page_count,
        view_mode = reader.paging and "paging" or "rolling",
    }
end

return {
    ---Return the current ReaderUI state.
    ---@return {
    ---    open: boolean,
    ---    file: string|nil,
    ---    page: number|nil,
    ---    page_count: number|nil,
    ---    view_mode: "paging"|"rolling"|nil}
    ["reader.state"] = function()
        return getReaderState()
    end,
    ["reader.close"] = function()
        local reader = getReaderInstance()

        if not reader or not reader.document then
            error("No book was opened")
            return false
        end

        common.defer(function()
            reader:onHome()
        end)

        return true
    end,
}
