package service

//
//import (
//	"reflect"
//	"slug/internal/kernel"
//	"slug/internal/token"
//)
//
//type EvaluateTokens struct {
//	Tokens []token.Token
//}
//
//var EvaluatorOperations = kernel.OpRights{
//	reflect.TypeOf(EvaluateTokens{}): kernel.RightExec,
//}
//
//type EvaluatorService struct {
//}
//
//func (m *EvaluatorService) Handler(ctx *kernel.ActCtx, msg kernel.Message) {
//	switch payload := msg.Payload.(type) {
//	case EvaluateTokens:
//		println(payload.Tokens)
//	}
//}
