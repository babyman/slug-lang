package service

import (
	"slug/internal/kernel"
)

// ===== Demo actor (unchanged API-wise) =====

func DemoHandler(ctx *kernel.ActCtx, msg kernel.Message) {
	_, ok := msg.Payload.(kernel.DemoStart)
	if !ok {
		return
	}
	SendInfo(ctx, "[demo] starting demo sequence…")

	fsID, _ := ctx.K.ActorByName("fs")
	timeID, _ := ctx.K.ActorByName("time")

	// Write a file
	if _, err := ctx.SendSync(fsID, FsWrite{
		Path: "/tmp/hello.txt",
		Data: []byte("hello from demo"),
	}); err != nil {
		SendInfof(ctx, "[demo] fs.write error: %v", err)
	} else {
		SendInfo(ctx, "[demo] fs.write ok")
	}
	// Read it back
	if resp, err := ctx.SendSync(fsID, FsRead{Path: "/tmp/hello.txt"}); err != nil {
		SendInfof(ctx, "[demo] fs.read error: %s", err)
	} else {
		SendInfof(ctx, "[demo] fs.read -> %v\n", resp.Payload.(FsReadResp).Data)
	}
	// Time.now
	if resp, err := ctx.SendSync(timeID, TsNow{}); err == nil {
		SendInfof(ctx, "[demo] time.now -> nanos=%v\n", resp.Payload.(TsNowResp).Nanos)
	}
	// Sleep 100 ms
	if _, err := ctx.SendSync(timeID, TsSleep{Ms: 100}); err == nil {
		SendInfo(ctx, "[demo] time.sleep 100ms ok")
	}
}
