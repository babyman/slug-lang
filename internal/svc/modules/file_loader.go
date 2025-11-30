package modules

import (
	"slug/internal/kernel"
	"slug/internal/object"
	"slug/internal/svc"
	"slug/internal/svc/resolver"
)

type FileLoader struct {
	DebugAST bool
}

func (fl *FileLoader) loadFileHandler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	fwdMsg := svc.UnpackFwd(msg)
	payload, _ := fwdMsg.Payload.(LoadFile)
	mod, err := fl.loadFile(ctx, msg, payload)
	svc.Reply(ctx, fwdMsg, LoadModuleResult{
		Module: mod,
		Error:  err,
	})
	return kernel.Terminate{}
}

func (fl *FileLoader) loadFile(ctx *kernel.ActCtx, msg kernel.Message, payload LoadFile) (*object.Module, error) {

	resId, _ := ctx.K.ActorByName(svc.ResolverService)

	resResult, err := ctx.SendSync(resId, resolver.ResolveFile{
		Path: payload.Path,
	})
	if err != nil {
		return nil, err
	}

	modData, _ := resResult.Payload.(resolver.ResolvedResult)

	if modData.Error != nil {
		return nil, modData.Error
	}

	return lexAndParseModule(ctx, modData, fl.DebugAST)
}
