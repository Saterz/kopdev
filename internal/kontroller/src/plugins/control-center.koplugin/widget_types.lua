local M = {}

-- Missing: BookMapRow, ConfigOption, ConfigPanel, ItemShortCutIcon, KeyValueItem, MenuItem, MinimalPaginator, NetworkItem, OptionIconItem, OptionTextItem, PhysicalNumericKey, SortItemWidget, ToggleLabel, TouchMenuBar, TouchMenuItem, VirtualKey
local widget_module_type_names = {
    ["ui/widget/bboxwidget"] = "BBoxWidget",
    ["ui/widget/booklist"] = "BookList",
    ["ui/widget/bookmapwidget"] = "BookMapWidget",
    ["ui/widget/bookmarkbrowser"] = "BookmarkBrowser",
    ["ui/widget/bookmetadataarchive"] = "BookMetadataArchive",
    ["ui/widget/bookstatuswidget"] = "BookStatusWidget",
    ["ui/widget/button"] = "Button",
    ["ui/widget/buttondialog"] = "ButtonDialog",
    ["ui/widget/buttonprogresswidget"] = "ButtonProgressWidget",
    ["ui/widget/buttonselector"] = "ButtonSelector",
    ["ui/widget/buttontable"] = "ButtonTable",
    ["ui/widget/checkbutton"] = "CheckButton",
    ["ui/widget/checkmark"] = "CheckMark",
    ["ui/widget/configdialog"] = "ConfigDialog",
    ["ui/widget/confirmbox"] = "ConfirmBox",

    ["ui/widget/container/alphacontainer"] = "AlphaContainer",
    ["ui/widget/container/bottomcontainer"] = "BottomContainer",
    ["ui/widget/container/centercontainer"] = "CenterContainer",
    ["ui/widget/container/custompositioncontainer"] = "CustomPositionContainer",
    ["ui/widget/container/framecontainer"] = "FrameContainer",
    ["ui/widget/container/inputcontainer"] = "InputContainer",
    ["ui/widget/container/leftcontainer"] = "LeftContainer",
    ["ui/widget/container/movablecontainer"] = "MovableContainer",
    ["ui/widget/container/rightcontainer"] = "RightContainer",
    ["ui/widget/container/scrollablecontainer"] = "ScrollableContainer",
    ["ui/widget/container/topcontainer"] = "TopContainer",
    ["ui/widget/container/underlinecontainer"] = "UnderlineContainer",
    ["ui/widget/container/widgetcontainer"] = "WidgetContainer",

    ["ui/widget/datetimewidget"] = "DateTimeWidget",
    ["ui/widget/dictquicklookup"] = "DictQuickLookup",
    ["ui/widget/doublespinwidget"] = "DoubleSpinWidget",
    ["ui/widget/eventlistener"] = "EventListener",
    ["ui/widget/filechooser"] = "FileChooser",
    ["ui/widget/fixedtextwidget"] = "FixedTextWidget",
    ["ui/widget/focusmanager"] = "FocusManager",
    ["ui/widget/fontchooser"] = "FontChooser",
    ["ui/widget/footnotewidget"] = "FootnoteWidget",
    ["ui/widget/frontlightwidget"] = "FrontLightWidget",
    ["ui/widget/horizontalgroup"] = "HorizontalGroup",
    ["ui/widget/horizontalscrollbar"] = "HorizontalScrollBar",
    ["ui/widget/horizontalspan"] = "HorizontalSpan",
    ["ui/widget/htmlboxwidget"] = "HtmlBoxWidget",
    ["ui/widget/iconbutton"] = "IconButton",
    ["ui/widget/iconwidget"] = "IconWidget",
    ["ui/widget/imageviewer"] = "ImageViewer",
    ["ui/widget/imagewidget"] = "ImageWidget",
    ["ui/widget/infomessage"] = "InfoMessage",
    ["ui/widget/inputdialog"] = "InputDialog",
    ["ui/widget/inputtext"] = "InputText",
    ["ui/widget/keyboardlayoutdialog"] = "KeyboardLayoutDialog",
    ["ui/widget/keyvaluepage"] = "KeyValuePage",
    ["ui/widget/linewidget"] = "LineWidget",
    ["ui/widget/linkbox"] = "LinkBox",
    ["ui/widget/listview"] = "ListView",
    ["ui/widget/menu"] = "Menu",
    ["ui/widget/multiconfirmbox"] = "MultiConfirmBox",
    ["ui/widget/multiinputdialog"] = "MultiInputDialog",
    ["ui/widget/naturallightwidget"] = "NaturalLightWidget",
    ["ui/widget/networksetting"] = "NetworkSetting",
    ["ui/widget/notification"] = "Notification",
    ["ui/widget/numberpickerwidget"] = "NumberPickerWidget",
    ["ui/widget/openwithdialog"] = "OpenWithDialog",
    ["ui/widget/overlapgroup"] = "OverlapGroup",
    ["ui/widget/pagebrowserwidget"] = "PageBrowserWidget",
    ["ui/widget/pathchooser"] = "PathChooser",
    ["ui/widget/physicalkeyboard"] = "PhysicalKeyboard",
    ["ui/widget/progressbardialog"] = "ProgressbarDialog",
    ["ui/widget/progresswidget"] = "ProgressWidget",
    ["ui/widget/qrmessage"] = "QRMessage",
    ["ui/widget/qrwidget"] = "QRWidget",
    ["ui/widget/radiobuttontable"] = "RadioButtonTable",
    ["ui/widget/radiobuttonwidget"] = "RadioButtonWidget",
    ["ui/widget/radiomark"] = "RadioMark",
    ["ui/widget/rectspan"] = "RectSpan",
    ["ui/widget/screensaverlockwidget"] = "ScreenSaverLockWidget",
    ["ui/widget/screensaverwidget"] = "ScreenSaverWidget",
    ["ui/widget/screenshoter"] = "Screenshoter",
    ["ui/widget/scrollhtmlwidget"] = "ScrollHtmlWidget",
    ["ui/widget/scrolltextwidget"] = "ScrollTextWidget",
    ["ui/widget/skimtowidget"] = "SkimToWidget",
    ["ui/widget/sortwidget"] = "SortWidget",
    ["ui/widget/spinwidget"] = "SpinWidget",
    ["ui/widget/textboxwidget"] = "TextBoxWidget",
    ["ui/widget/textviewer"] = "TextViewer",
    ["ui/widget/textwidget"] = "TextWidget",
    ["ui/widget/titlebar"] = "TitleBar",
    ["ui/widget/toggleswitch"] = "ToggleSwitch",
    ["ui/widget/touchmenu"] = "TouchMenu",
    ["ui/widget/trapwidget"] = "TrapWidget",
    ["ui/widget/verticalgroup"] = "VerticalGroup",
    ["ui/widget/verticalscrollbar"] = "VerticalScrollBar",
    ["ui/widget/verticalspan"] = "VerticalSpan",
    ["ui/widget/virtualkeyboard"] = "VirtualKeyboard",
    ["ui/widget/widget"] = "Widget",
}

local widget_type_names = {}

function M.getWidgetType(widget)
    if type(widget) ~= "table" then
        return nil
    end

    local widget_class = getmetatable(widget)
    if not widget_class then
        return nil
    end

    local cached_name = widget_type_names[widget_class]
    if cached_name then
        return cached_name
    end

    for module_name, class_name in pairs(widget_module_type_names) do
        if package.loaded[module_name] == widget_class then
            widget_type_names[widget_class] = class_name
            return class_name
        end
    end

    return nil
end

return M
