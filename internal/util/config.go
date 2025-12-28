package util

type Configuration struct {
	Version      string
	BuildDate    string
	Commit       string
	RootPath     string
	SlugHome     string
	DebugJsonAST bool
	DebugTxtAST  bool
}
