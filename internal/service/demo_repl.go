package service

import (
	"errors"
	"reflect"
	"slug/internal/kernel"
)

// ===== REPL Service =====
// Ops: { eval: EXEC }
// Behavior:
//   - Accepts {source} and forwards to Evaluator
//   - Returns evaluator reply

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

func (r *ReplService) Behavior(ctx *kernel.ActCtx, msg kernel.Message) {
	switch payload := msg.Payload.(type) {
	case RsEval:
		src := payload.Source
		if src == "" {
			reply(ctx, msg, RsEvalResp{Err: errors.New("empty source")})
			return
		}
		resp, err := ctx.SendSync(r.EvalID, EvaluatorEvaluate{
			Source: src,
		})
		switch {
		case err != nil:
			{
				reply(ctx, msg, RsEvalResp{Err: err})
				return
			}
		case resp.Payload == nil:
			{
				reply(ctx, msg, RsEvalResp{Err: errors.New("no reply")})
				return
			}
		}
		reply(ctx, msg, RsEvalResp{
			Result: resp.Payload.(EvaluatorResult).Result,
		})
	default:
		reply(ctx, msg, RsEvalResp{Err: errors.New("unknown op")})
	}
}
