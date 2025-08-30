package service

//
//import (
//	"errors"
//	"reflect"
//	"regexp"
//	"slug/internal/kernel"
//	"strings"
//)
//
//// ===== Evaluator Service (stub) =====
//// Ops: { eval: EXEC }
//// Handler:
////   - Very small demo parser with three commands to prove wiring:
////     1) print "text"           -> returns stdout
////     2) read "path"            -> reads from FS service (needs caps)
////     3) now                     -> reads time.now
////   - Anything else: echoes back length of source.
//
//type EvaluatorEvaluate struct {
//	Source string
//}
//
//type EvaluatorResult struct {
//	Stdout    string
//	SourceLen int
//	Result    any
//	Err       error
//}
//
//var EvaluatorOperations = kernel.OpRights{
//	reflect.TypeOf(EvaluatorEvaluate{}): kernel.RightExec,
//}
//
//type Evaluator struct{}
//
//var (
//	cmdPrint = regexp.MustCompile(`^\s*print\s+\"(.*)\"\s*$`)
//	cmdRead  = regexp.MustCompile(`^\s*read\s+\"(.*)\"\s*$`)
//)
//
//func (e *Evaluator) Handler(ctx *kernel.ActCtx, msg kernel.Message) {
//	switch payload := msg.Payload.(type) {
//	case EvaluatorEvaluate:
//		src := payload.Source
//		if src == "" {
//			Reply(ctx, msg, EvaluatorResult{Err: errors.New("empty source")})
//			return
//		}
//		// Discover services by name; in a real runtime this would be passed as caps
//		fsID, _ := ctx.K.ActorByName("fs")
//		timeID, _ := ctx.K.ActorByName("time")
//
//		// Simple commands
//		if m := cmdPrint.FindStringSubmatch(src); len(m) == 2 {
//			Reply(ctx, msg, EvaluatorResult{Stdout: m[1]})
//			return
//		}
//		if m := cmdRead.FindStringSubmatch(src); len(m) == 2 {
//
//			fsResp, err := ctx.SendSync(fsID, FsRead{Path: m[1]})
//			switch {
//			case err != nil:
//				Reply(ctx, msg, EvaluatorResult{Err: err})
//				return
//			case fsResp.Payload.(FsReadResp).Err != nil:
//				Reply(ctx, msg, EvaluatorResult{Err: fsResp.Payload.(FsReadResp).Err})
//				return
//			default:
//				Reply(ctx, msg, EvaluatorResult{Stdout: fsResp.Payload.(FsReadResp).Data})
//				return
//			}
//		}
//		if strings.TrimSpace(src) == "now" {
//			if resp, err := ctx.SendSync(timeID, TsNow{}); err == nil {
//				Reply(ctx, msg, EvaluatorResult{Result: resp.Payload.(TsNowResp).Nanos})
//				return
//			}
//		}
//
//		Reply(ctx, msg, EvaluatorResult{SourceLen: len(src)})
//	default:
//		Reply(ctx, msg, EvaluatorResult{Err: errors.New("unknown op")})
//	}
//}
