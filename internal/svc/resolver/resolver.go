package resolver

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slug/internal/kernel"
	"slug/internal/logger"
	"slug/internal/svc"
	"slug/internal/svc/fs"
	"strings"
)

const DefaultRootPath = "."

var log = logger.NewLogger("resolver-svc", svc.LogLevel)

type ResolveFile struct {
	Path string
}

type ResolveModule struct {
	PathParts []string
}

type ResolvedResult struct {
	ModuleName string
	ModulePath string
	Data       string
	Error      error
}

var Operations = kernel.OpRights{
	reflect.TypeOf(kernel.ConfigureSystem{}): kernel.RightExec,
	reflect.TypeOf(ResolveFile{}):            kernel.RightRead,
	reflect.TypeOf(ResolveModule{}):          kernel.RightRead,
}

type Resolver struct {
	RootPath string
	SlugHome string
}

func NewResolver() *Resolver {
	return &Resolver{
		RootPath: DefaultRootPath,
		SlugHome: os.Getenv("SLUG_HOME"),
	}
}

func (r *Resolver) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	switch payload := msg.Payload.(type) {
	case kernel.ConfigureSystem:
		r.RootPath = payload.SystemRootPath
		svc.Reply(ctx, msg, nil)

	case ResolveFile:
		workedId, _ := ctx.SpawnChild("res-f-wrk", r.resolveFileHandler)
		err := ctx.SendAsync(workedId, msg)
		if err != nil {
			log.Error(err.Error())
		}

	case ResolveModule:
		workedId, _ := ctx.SpawnChild("res-m-wrk", r.resolveModuleHandler)
		err := ctx.SendAsync(workedId, msg)
		if err != nil {
			log.Error(err.Error())
		}

	default:
		svc.Reply(ctx, msg, kernel.UnknownOperation{})
	}
	return kernel.Continue{}
}

func (r *Resolver) resolveModuleHandler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	fwdMsg := svc.UnpackFwd(msg)
	payload, _ := fwdMsg.Payload.(ResolveModule)

	name, path, data, err := r.resolveModule(ctx, r.RootPath, payload.PathParts)

	svc.Reply(ctx, fwdMsg, ResolvedResult{
		ModuleName: name,
		ModulePath: path,
		Data:       data,
		Error:      err,
	})

	return kernel.Terminate{}
}

func (r *Resolver) resolveFileHandler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	fwdMsg := svc.UnpackFwd(msg)
	payload, _ := fwdMsg.Payload.(ResolveFile)

	newRootPath, modulePathParts, err := calculateModulePath(payload.Path, r.RootPath)
	name, path, data, err := r.resolveModule(ctx, newRootPath, modulePathParts)

	if err != nil && r.RootPath != DefaultRootPath {
		r.RootPath = newRootPath
	}

	svc.Reply(ctx, fwdMsg, ResolvedResult{
		ModuleName: name,
		ModulePath: path,
		Data:       data,
		Error:      err,
	})

	return kernel.Terminate{}
}

func (r *Resolver) resolveModule(ctx *kernel.ActCtx, rootPath string, pathParts []string) (string, string, string, error) {

	fsId, _ := ctx.K.ActorByName(svc.FsService)

	moduleName := strings.Join(pathParts, ".")

	log.Infof("Loading module '%s' from path parts: %v  Root path: %s\n",
		moduleName, pathParts, rootPath)

	// Complete the module path
	moduleRelativePath := strings.Join(pathParts, "/")
	modulePath := fmt.Sprintf("%s/%s.slug", rootPath, moduleRelativePath)

	// Attempt to load the module's source
	fsResponse, err := ctx.SendSync(fsId, fs.Read{Path: modulePath})
	if err != nil {
		log.Infof("Failed to read file: %s", err)
		return "", "", "", err
	}

	// check the response
	readResp := fsResponse.Payload.(fs.ReadResp)

	if readResp.Err != nil {
		libPath, err := r.slugLibPath(moduleName, moduleRelativePath)
		if err != nil {
			return "", "", "", err
		}

		fsResponse, err = ctx.SendSync(fsId, fs.Read{Path: libPath})
		if err != nil {
			return "", "", "", fmt.Errorf("error reading module (%s / %s) '%s': %s", modulePath, libPath, moduleName, err)
		} else {
			modulePath = libPath
			readResp = fsResponse.Payload.(fs.ReadResp)
		}
	}

	return moduleName, modulePath, readResp.Data, readResp.Err
}

func (r *Resolver) slugLibPath(moduleName string, moduleRelativePath string) (string, error) {
	if r.SlugHome == "" {
		return "", fmt.Errorf("error reading module '%s': SLUG_HOME environment variable is not set", moduleName)
	}
	libPath := fmt.Sprintf("%s/lib/%s.slug", r.SlugHome, moduleRelativePath)
	return libPath, nil
}

func calculateModulePath(filename string, rootPath string) (string, []string, error) {

	// Check if file exists and is not a directory
	isSource, err := isSourceFile(filename)
	if err != nil {
		return "", nil, err
	}

	if rootPath == "." && isSource {
		rootPath = filepath.Dir(filename)
	}

	// Calculate the module path relative to root path
	absFilePath, err := filepath.Abs(filename)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get absolute path for '%s': %v", filename, err)
	}

	absRootPath, err := filepath.Abs(rootPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get absolute path for root '%s': %v", rootPath, err)
	}

	if !isSource {
		absFilePath = absRootPath
	}

	modulePath, err := filepath.Rel(absRootPath, absFilePath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to calculate relative path: %v", err)
	}
	if !isSource {
		modulePath = filename
	}

	// Remove .slug extension if present
	modulePath = strings.TrimSuffix(modulePath, ".slug")

	modulePathParts := strings.Split(modulePath, string(filepath.Separator))
	return absRootPath, modulePathParts, nil
}

func isSourceFile(filename string) (bool, error) {

	fileInfo, err := os.Stat(filename)

	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("error accessing file '%s': %v", filename, err)
	}

	if fileInfo.IsDir() {
		return false, fmt.Errorf("'%s' is a directory, not a file", filename)
	}

	return true, nil
}
