package foreign

import (
	"slug/internal/logger"
	"slug/internal/object"
	"slug/internal/svc"
)

var log = logger.NewLogger("foreign", svc.LogLevel)

func GetForeignFunctions() map[string]*object.Foreign {
	return map[string]*object.Foreign{

		"slug.io.fs.readFile":   fnIoFsReadFile(),
		"slug.io.fs.writeFile":  fnIoFsWriteFile(),
		"slug.io.fs.appendFile": fnIoFsAppendFile(),
		"slug.io.fs.info":       fnIoFsInfo(),
		"slug.io.fs.exists":     fnIoFsExists(),
		"slug.io.fs.mkDirs":     fnIoFsMkdirs(),
		"slug.io.fs.isDir":      fnIoFsIsDir(),
		"slug.io.fs.ls":         fnIoFsLs(),
		"slug.io.fs.rm":         fnIoFsRm(),
		"slug.io.fs.openFile":   fnIoFsOpenFile(),
		"slug.io.fs.readLine":   fnIoFsReadLine(),
		"slug.io.fs.write":      fnIoFsWrite(),
		"slug.io.fs.closeFile":  fnIoFsCloseFile(),

		"slug.io.http.request": fnIoHttpRequest(),

		"slug.io.tcp.bind":    fnIoTcpBind(),
		"slug.io.tcp.accept":  fnIoTcpAccept(),
		"slug.io.tcp.connect": fnIoTcpConnect(),
		"slug.io.tcp.read":    fnIoTcpRead(),
		"slug.io.tcp.write":   fnIoTcpWrite(),
		"slug.io.tcp.close":   fnIoTcpClose(),

		"slug.bytes.strToBytes":    fnBytesStrToBytes(),
		"slug.bytes.bytesToStr":    fnBytesBytesToStr(),
		"slug.bytes.hexStrToBytes": fnBytesHexStrToBytes(),
		"slug.bytes.bytesToHexStr": fnBytesBytesToHexStr(),
		"slug.bytes.base64Encode":  fnBytesBase64Encode(),
		"slug.bytes.base64Decode":  fnBytesBase64Decode(),

		"slug.crypto.md5":    fnCryptoMd5(),
		"slug.crypto.sha256": fnCryptoSha256(),
		"slug.crypto.sha512": fnCryptoSha512(),

		"slug.list.sortWithComparator": fnListSortWithComparator(),

		"slug.math.rndRange": fnMathRndRange(),

		"slug.meta.hasTag":           fnMetaHasTag(),
		"slug.meta.getTag":           fnMetaGetTag(),
		"slug.meta.searchModuleTags": fnMetaSearchModuleTags(),
		"slug.meta.searchScopeTags":  fnMetaSearchScopeTags(),
		"slug.meta.rebindScopeTags":  fnMetaRebindScopeTags(),
		"slug.meta.withEnv":          fnMetaWithEnv(),

		"slug.regex.matches":       fnRegexMatches(),
		"slug.regex.split":         fnRegexSplit(),
		"slug.regex.findAll":       fnRegexFindAll(),
		"slug.regex.findAllGroups": fnRegexFindAllGroups(),
		"slug.regex.replaceAll":    fnRegexReplaceAll(),

		"slug.std.type":        fnStdType(),
		"slug.std.isDefined":   fnStdIsDefined(),
		"slug.std.printf":      fnStdPrintf(),
		"slug.std.sprintf":     fnStdSprintf(),
		"slug.std.update":      fnStdUpdate(),
		"slug.std.swap":        fnStdSwap(),
		"slug.std.parseNumber": fnStdParseNumber(),
		"slug.std.get":         fnStdGet(),
		"slug.std.keys":        fnStdKeys(),
		"slug.std.put":         fnStdPut(),
		"slug.std.remove":      fnStdRemove(),

		// string functions
		"slug.string.indexOf": fnStringIndexOf(),
		"slug.string.toLower": fnStringToLower(),
		"slug.string.toUpper": fnStringToUpper(),
		"slug.string.trim":    fnStringTrim(),

		"slug.sys.env":    fnSysEnv(),
		"slug.sys.setEnv": fnSysSetEnv(),
		"slug.sys.exit":   fnSysExit(),

		"slug.time.clock":      fnTimeClock(),
		"slug.time.fmtClock":   fnTimeFmtClock(),
		"slug.time.clockNanos": fnTimeClockNanos(),
		"slug.time.sleep":      fnTimeSleep(),
	}
}
