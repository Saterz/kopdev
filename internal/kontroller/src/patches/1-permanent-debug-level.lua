-- When KOReader restarts the log level is no longer set to debug; at each restart this patch loads and sets the logging level to debug
local dbg = require("dbg")
dbg:turnOn()
dbg:setVerbose(true)
