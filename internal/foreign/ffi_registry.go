package foreign

import (
	"slug/internal/object"
)

func GetForeignFunctions() map[string]*object.Foreign {
	return map[string]*object.Foreign{

		"slug.bytes.strToBytes":    fnBytesStrToBytes(),
		"slug.bytes.bytesToStr":    fnBytesBytesToStr(),
		"slug.bytes.hexStrToBytes": fnBytesHexStrToBytes(),
		"slug.bytes.bytesToHexStr": fnBytesBytesToHexStr(),
		"slug.bytes.base64Encode":  fnBytesBase64Encode(),
		"slug.bytes.base64Decode":  fnBytesBase64Decode(),

		"slug.crypto.md5":    fnCryptoMd5(),
		"slug.crypto.sha256": fnCryptoSha256(),
		"slug.crypto.sha512": fnCryptoSha512(),

		"slug.io.db.connect":  fnIoDbConnect(),
		"slug.io.db.query":    fnIoDbQuery(),
		"slug.io.db.exec":     fnIoDbExec(),
		"slug.io.db.close":    fnIoDbClose(),
		"slug.io.db.begin":    fnIoDbBegin(),
		"slug.io.db.commit":   fnIoDbCommit(),
		"slug.io.db.rollback": fnIoDbRollback(),

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

		"slug.list.sortWithComparator": fnListSortWithComparator(),

		"slug.math.ceil":     fnMathCeil(),
		"slug.math.floor":    fnMathFloor(),
		"slug.math.rndRange": fnMathRndRange(),
		"slug.math.sqrt":     fnMathSqrt(),

		"slug.meta.hasTag":           fnMetaHasTag(),
		"slug.meta.getTag":           fnMetaGetTag(),
		"slug.meta.docs":             fnMetaDocs(),
		"slug.meta.moduleDocs":       fnMetaModuleDocs(),
		"slug.meta.searchModuleTags": fnMetaSearchModuleTags(),
		"slug.meta.searchScopeTags":  fnMetaSearchScopeTags(),

		"slug.regex.findAll":       fnRegexFindAll(),
		"slug.regex.findAllGroups": fnRegexFindAllGroups(),
		"slug.regex.indexOf":       fnRegexIndexOf(),
		"slug.regex.matches":       fnRegexMatches(),
		"slug.regex.replaceAll":    fnRegexReplaceAll(),
		"slug.regex.split":         fnRegexSplit(),

		"slug.std.type":        fnStdType(),
		"slug.std.isDefined":   fnStdIsDefined(),
		"slug.std.printf":      fnStdPrintf(),
		"slug.std.sprintf":     fnStdSprintf(),
		"slug.std.update":      fnStdUpdate(),
		"slug.std.swap":        fnStdSwap(),
		"slug.std.parseNumber": fnStdParseNumber(),
		"slug.std.get":         fnStdGet(),
		"slug.std.keys":        fnStdKeys(),
		"slug.std.sym":         fnStdSym(),
		"slug.std.label":       fnStdLabel(),
		"slug.std.put":         fnStdPut(),
		"slug.std.remove":      fnStdRemove(),

		// string functions
		"slug.string.indexOf": fnStringIndexOf(),
		"slug.string.toLower": fnStringToLower(),
		"slug.string.toUpper": fnStringToUpper(),
		"slug.string.trim":    fnStringTrim(),

		"slug.sys.env":    fnSysEnv(),
		"slug.sys.exec":   fnSysExec(),
		"slug.sys.exit":   fnSysExit(),
		"slug.sys.setEnv": fnSysSetEnv(),

		"slug.time.clock":      fnTimeClock(),
		"slug.time.fmtClock":   fnTimeFmtClock(),
		"slug.time.clockNanos": fnTimeClockNanos(),
		"slug.time.sleep":      fnTimeSleep(),
	}
}
