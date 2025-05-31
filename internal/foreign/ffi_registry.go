package foreign

import "slug/internal/object"

func GetForeignFunctions() map[string]*object.Foreign {
	return map[string]*object.Foreign{

		"slug.io.fs.readFile":   fnIoFsReadFile(),
		"slug.io.fs.writeFile":  fnIoFsWriteFile(),
		"slug.io.fs.appendFile": fnIoFsAppendFile(),
		"slug.io.fs.info":       fnIoFsInfo(),
		"slug.io.fs.exists":     fnIoFsExists(),
		"slug.io.fs.isDir":      fnIoFsIsDir(),
		"slug.io.fs.ls":         fnIoFsLs(),
		"slug.io.fs.rm":         fnIoFsRm(),
		"slug.io.fs.openFile":   fnIoFsOpenFile(),
		"slug.io.fs.readLine":   fnIoFsReadLine(),
		"slug.io.fs.write":      fnIoFsWrite(),
		"slug.io.fs.closeFile":  fnIoFsCloseFile(),

		"slug.io.tcp.bind":    fnIoTcpBind(),
		"slug.io.tcp.accept":  fnIoTcpAccept(),
		"slug.io.tcp.connect": fnIoTcpConnect(),
		"slug.io.tcp.read":    fnIoTcpRead(),
		"slug.io.tcp.write":   fnIoTcpWrite(),
		"slug.io.tcp.close":   fnIoTcpClose(),

		"slug.std.type":      fnStdType(),
		"slug.std.isDefined": fnStdIsDefined(),
		"slug.std.len":       fnStdLen(),
		"slug.std.print":     fnStdPrint(),
		"slug.std.get":       fnStdGet(),
		"slug.std.keys":      fnStdKeys(),
		"slug.std.put":       fnStdPut(),
		"slug.std.remove":    fnStdRemove(),

		// string functions
		"slug.string.indexOf": fnStringIndexOf(),
		"slug.string.toLower": fnStringToLower(),
		"slug.string.toUpper": fnStringToUpper(),
		"slug.string.trim":    fnStringTrim(),

		"slug.sys.env": fnSysEnv(),

		"slug.time.clock":      fnTimeClock(),
		"slug.time.clockNanos": fnTimeClockNanos(),
		"slug.time.sleep":      fnTimeSleep(),
	}
}
