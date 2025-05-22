package evaluator

import (
	"slug/internal/object"
)

var foreignFunctions = map[string]*object.Foreign{
	"slug.io.tcp.bind":    fnIoTcpBind(),
	"slug.io.tcp.accept":  fnIoTcpAccept(),
	"slug.io.tcp.connect": fnIoTcpConnect(),
	"slug.io.tcp.read":    fnIoTcpRead(),
	"slug.io.tcp.write":   fnIoTcpWrite(),
	"slug.io.tcp.close":   fnIoTcpClose(),

	"slug.std.len":    fnStdLen(),
	"slug.std.print":  fnStdPrint(),
	"slug.std.type":   fnStdType(),
	"slug.std.get":    fnStdGet(),
	"slug.std.keys":   fnStdKeys(),
	"slug.std.put":    fnStdPut(),
	"slug.std.remove": fnStdRemove(),

	// string functions
	"slug.strings.contains":   fnStringsContains(),
	"slug.strings.endsWith":   fnStringsEndsWith(),
	"slug.strings.indexOf":    fnStringsIndexOf(),
	"slug.strings.startsWith": fnStringsStartsWith(),
	"slug.strings.trim":       fnStringsTrim(),

	"slug.sys.env": fnSysEnv(),

	"slug.time.clock":      fnTimeClock(),
	"slug.time.clockNanos": fnTimeClockNanos(),
}
