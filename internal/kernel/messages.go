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

type Exit struct {
	Reason string
}

type Kill struct {
	Target ActorID
	Reason string
}

type RegisterCleanup struct {
	Target ActorID
	Msg    Message
}
