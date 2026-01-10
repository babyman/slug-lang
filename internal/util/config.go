package util

type Configuration struct {
	Version      string
	RootPath     string
	SlugHome     string
	Argv         []string
	DebugJsonAST bool
	DebugTxtAST  bool
	DefaultLimit int
}
