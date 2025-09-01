package kernel

//Kernel message payload types

type Boot struct {
}

type Shutdown struct {
	ExitCode int `json:"exitcode,omitempty"`
}

type UnknownOperation struct {
}
