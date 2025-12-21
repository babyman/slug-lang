package svcutil

import (
	"slug/internal/kernel"
	"slug/internal/object"
	"slug/internal/svc"
)

func CloseResult(from kernel.ActorID) svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	PutString(resultMap, "type", "closed")
	PutInt(resultMap, "from", int(from))
	return svc.SlugActorMessage{Msg: resultMap}
}

func ErrorResult(msg string) svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	PutString(resultMap, "type", "error")
	PutString(resultMap, "msg", msg)
	return svc.SlugActorMessage{Msg: resultMap}
}

func ReplyTarget(msg kernel.Message) kernel.ActorID {
	if msg.ReplyTo > 0 {
		return msg.ReplyTo
	}
	return msg.From
}

func ExtractParameters(m *object.Map) ([]any, bool) {

	paramsObj := m.Pairs[ParamsKey].Value

	paramsList, ok := paramsObj.(*object.List)
	if !ok {
		return nil, false
	}

	params := make([]any, len(paramsList.Elements))
	for i, elem := range paramsList.Elements {
		params[i] = elem.Inspect()
	}
	return params, true
}
