package service

import (
	"log"
	"slug/internal/kernel"
)

// ===== Demo actor (unchanged API-wise) =====

func DemoBehavior(ctx *kernel.ActCtx, msg kernel.Message) {
	if msg.Op != "start" {
		return
	}
	log.Println("[demo] starting demo sequence…")

	fsID, _ := ctx.K.ActorByName("fs")
	timeID, _ := ctx.K.ActorByName("time")

	// Write a file
	if _, err := ctx.SendSync(fsID, "write", FsWrite{
		Path: "/tmp/hello.txt",
		Data: []byte("hello from demo"),
	}); err != nil {
		log.Println("[demo] fs.write error:", err)
	} else {
		log.Println("[demo] fs.write ok")
	}
	// Read it back
	if resp, err := ctx.SendSync(fsID, "read", FsRead{Path: "/tmp/hello.txt"}); err != nil {
		log.Println("[demo] fs.read error:", err)
	} else {
		log.Printf("[demo] fs.read -> %v\n", resp.Payload.(FsReadResp).Data)
	}
	// Time.now
	if resp, err := ctx.SendSync(timeID, "now", TsNow{}); err == nil {
		log.Printf("[demo] time.now -> nanos=%v\n", resp.Payload.(TsNowResp).Nanos)
	}
	// Sleep 100 ms
	if _, err := ctx.SendSync(timeID, "sleep", TsSleep{Ms: 100}); err == nil {
		log.Println("[demo] time.sleep 100ms ok")
	}
}
