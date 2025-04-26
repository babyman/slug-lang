package object

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	env.rootPath = outer.rootPath
	return env
}

func NewEnvironment() *Environment {
	s := make(map[string]Object)
	return &Environment{Store: s, outer: nil, rootPath: "."}
}

type Environment struct {
	Store    map[string]Object
	outer    *Environment
	Src      string
	Path     string
	rootPath string // Track the execution root path
}

// SetRootPath sets the root path for the environment
func (e *Environment) SetRootPath(path string) {
	e.rootPath = path
}

// GetRootPath retrieves the root path for the environment
func (e *Environment) GetRootPath() string {
	return e.rootPath
}

func (e *Environment) Get(name string) (Object, bool) {
	obj, ok := e.Store[name]
	if !ok && e.outer != nil {
		obj, ok = e.outer.Get(name)
	}
	return obj, ok
}

func (e *Environment) Set(name string, val Object) Object {
	e.Store[name] = val
	return val
}
