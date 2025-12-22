package kernel

import (
	"reflect"
)

const KernelService = "kernel"

var Operations = OpRights{
	reflect.TypeOf(Broadcast{}):       RightExec,
	reflect.TypeOf(RequestShutdown{}): RightExec,
}

//Kernel message payload types

type Boot struct {
}

type RequestShutdown struct {
	ExitCode int `json:"exitcode,omitempty"`
}

type UnknownOperation struct {
}

type Broadcast struct {
	Payload any
}

// Exit represents a message sent to an actor to terminate it, this is handled by the kernel.
type Exit struct {
	Reason string
}

// Shutdown represents a message passed to an actor to allow it to gracefully terminate.
type Shutdown struct {
	Reason string
}
