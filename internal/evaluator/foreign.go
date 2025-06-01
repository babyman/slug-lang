package evaluator

import (
	"slug/internal/foreign"
	"slug/internal/object"
)

var builtins = map[string]*object.Foreign{
	"import":  fnBuiltinImport(),
	"len":     fnBuiltinLen(),
	"print":   fnBuiltinPrint(),
	"println": fnBuiltinPrintLn(),
}

var foreignFunctions map[string]*object.Foreign

func lookupForeign(name string) (*object.Foreign, bool) {
	if fn, ok := getForeignFunctions()[name]; ok {
		return fn, true
	}
	return nil, false
}

func getForeignFunctions() map[string]*object.Foreign {
	if foreignFunctions == nil {
		foreignFunctions = map[string]*object.Foreign{
			"slug.actor.spawn":      fnActorSpawn(),
			"slug.actor.self":       fnActorSelf(),
			"slug.actor.send":       fnActorSend(),
			"slug.actor.receive":    fnActorReceive(),
			"slug.actor.register":   fnActorRegister(),
			"slug.actor.unregister": fnActorUnregister(),
			"slug.actor.whereIs":    fnActorWhereIs(),
		}
		for k, v := range foreign.GetForeignFunctions() {
			foreignFunctions[k] = v
		}
	}
	return foreignFunctions
}
