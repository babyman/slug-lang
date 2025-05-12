package evaluator

import (
	"slug/internal/object"
)

var foreignFunctions = map[string]*object.Foreign{
	"slug.std.len":     fnStdLen(),
	"slug.std.println": fnStdPrintLn(),
	"slug.std.type":    fnStdType(),
	// map functions
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
}
