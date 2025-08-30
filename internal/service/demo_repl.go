package service

import (
	"errors"
	"reflect"
	"slug/internal/kernel"
)

// ===== REPL Service =====
// Ops: { eval: EXEC }
// Handler:
//   - Accepts {source} and forwards to Evaluator
//   - Returns evaluator Reply

type RsEval struct {
	Source string
}

type RsEvalResp struct {
	Result any
	Err    error
}

var RsOperations = kernel.OpRights{
	reflect.TypeOf(RsEval{}): kernel.RightExec,
}

type ReplService struct{ EvalID kernel.ActorID }

func (r *ReplService) Handler(ctx *kernel.ActCtx, msg kernel.Message) {
	switch payload := msg.Payload.(type) {
	case RsEval:
		src := payload.Source
		if src == "" {
			Reply(ctx, msg, RsEvalResp{Err: errors.New("empty source")})
			return
		}
		//resp, err := ctx.SendSync(r.EvalID, EvaluatorEvaluate{
		//	Source: src,
		//})
		//switch {
		//case err != nil:
		//	{
		//		Reply(ctx, msg, RsEvalResp{Err: err})
		//		return
		//	}
		//case resp.Payload == nil:
		//	{
		//		Reply(ctx, msg, RsEvalResp{Err: errors.New("no Reply")})
		//		return
		//	}
		//}
		//Reply(ctx, msg, RsEvalResp{
		//	Result: resp.Payload.(EvaluatorResult).Result,
		//})
	default:
		Reply(ctx, msg, RsEvalResp{Err: errors.New("unknown op")})
	}
}
