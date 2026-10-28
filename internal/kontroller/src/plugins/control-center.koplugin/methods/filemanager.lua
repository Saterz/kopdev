local ffiUtil = require("ffi/util")
local lfs = require("libs/libkoreader-lfs")
local util = require("util")

---Return the active FileManager and FileChooser instances.
---@return table|nil file_manager
---@return table|nil file_chooser
local function getFileManager()
    local FileManager = require("apps/filemanager/filemanager")
    local file_manager = FileManager.instance
    local file_chooser = file_manager and file_manager.file_chooser

    return file_manager, file_chooser
end

---Serialize one FileChooser item into JSON-safe data.
---@param item table
---@return table serialized_item
local function serializeFileManagerItem(item)
    local properties = type(item.doc_props) == "table"
        and item.doc_props
        or {}

    local is_directory = item.is_directory == true
        or item.mode == "directory"
        or type(item.attr) == "table"
        and item.attr.mode == "directory"

    return {
        text = item.text,
        path = item.path or item.file,
        is_file = item.is_file == true,
        is_directory = is_directory,
        is_parent = item.is_go_up == true,
        enabled = item.enabled ~= false,
        dimmed = item.dim == true,

        -- These fields are generally provided by CoverBrowser.
        title = properties.title or properties.display_title,
        authors = properties.authors,
        series = properties.series,
        pages = properties.pages,
    }
end

---Return serializable information about the active file manager.
---@return table state
local function getFileManagerState()
    local _, file_chooser = getFileManager()

    if not file_chooser then
        return {
            available = false,
        }
    end

    local items = {}

    for _, item in ipairs(file_chooser.item_table or {}) do
        table.insert(items, serializeFileManagerItem(item))
    end

    local focused_index =
        file_chooser.itemnumber or file_chooser.prev_itemnumber
    local focused_item = focused_index
        and file_chooser.item_table
        and file_chooser.item_table[focused_index]

    return {
        available = true,
        path = file_chooser.path,
        display_mode = file_chooser.display_mode_type
            or file_chooser.display_mode,
        page = file_chooser.page,
        page_count = file_chooser.page_num,
        items_per_page = file_chooser.perpage,
        focused_index = focused_index,
        focused_path = focused_item
            and (focused_item.path or focused_item.file),
        item_count = #items,
        items = items,
    }
end

local function openFileManagerFile(requested_path)
    if type(requested_path) ~= "string"
        or requested_path == ""
    then
        error("filemanager.open requires a non-empty path")
    end

    local path = ffiUtil.realpath(requested_path)

    if not path then
        error("file does not exist: " .. requested_path)
    end

    if lfs.attributes(path, "mode") ~= "file" then
        error("path is not a file: " .. path)
    end

    local file_manager, file_chooser = getFileManager()

    if not file_chooser then
        error("file manager is not available")
    end

    if file_manager.selected_files then
        error("file manager is currently in file-selection mode")
    end

    local parent = util.splitFilePathName(path)

    -- Make FileChooser load the directory containing the requested file.
    if ffiUtil.realpath(file_chooser.path) ~= ffiUtil.realpath(parent) then
        file_chooser:changeToPath(parent)
    end

    local selected_item

    for _, item in ipairs(file_chooser.item_table or {}) do
        if item.is_file == true and item.path then
            local item_path = ffiUtil.realpath(item.path)

            if item_path == path then
                selected_item = item
                break
            end
        end
    end

    if not selected_item then
        error(
            "file is not visible to FileManager or is unsupported: "
            .. path
        )
    end

    local accepted = file_chooser:onFileSelect(selected_item)

    if accepted == false then
        error("FileManager rejected the file: " .. path)
    end

    return true
end

return {
    ---Return the current FileManager state.
    ---@return {
    ---    available: boolean,
    ---    path: string|nil,
    ---    display_mode: string|nil,
    ---    page: number|nil,
    ---    page_count: number|nil,
    ---    items_per_page: number|nil,
    ---    focused_index: number|nil,
    ---    focused_path: string|nil,
    ---    item_count: number|nil,
    ---    items: table[]|nil}
    ["filemanager.state"] = function()
        return getFileManagerState()
    end,
    ---Open a file through the active FileManager.
    ---@param params {path: string}
    ---@return boolean
    ["filemanager.open"] = function(params)
        return openFileManagerFile(params.path)
    end,
}
