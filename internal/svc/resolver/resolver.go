package resolver

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"slug/internal/kernel"
	"slug/internal/svc"
	"slug/internal/svc/fs"
	"slug/internal/util"
	"strings"
)

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
	reflect.TypeOf(ResolveFile{}):   kernel.RightRead,
	reflect.TypeOf(ResolveModule{}): kernel.RightRead,
}

type Resolver struct {
	Config util.Configuration
}

func (r *Resolver) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	switch msg.Payload.(type) {
	case ResolveFile:
		worker := ResolveWorker{RootPath: r.Config.RootPath, SlugHome: r.Config.SlugHome}
		workedId, _ := ctx.SpawnChild("res-f-wrk", worker.resolveFileHandler)
		err := ctx.SendAsync(workedId, msg)
		if err != nil {
			slog.Error("error messaging file handler", slog.Any("error", err.Error()))
		}

	case ResolveModule:
		worker := ResolveWorker{RootPath: r.Config.RootPath, SlugHome: r.Config.SlugHome}
		workedId, _ := ctx.SpawnChild("res-m-wrk", worker.resolveModuleHandler)
		err := ctx.SendAsync(workedId, msg)
		if err != nil {
			slog.Error("error messaging module handler", slog.Any("error", err.Error()))
		}

	default:
		svc.Reply(ctx, msg, kernel.UnknownOperation{})
	}
	return kernel.Continue{}
}

type ResolveWorker struct {
	RootPath string
	SlugHome string
}

func (r *ResolveWorker) resolveModuleHandler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
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

func (r *ResolveWorker) resolveFileHandler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	fwdMsg := svc.UnpackFwd(msg)
	payload, _ := fwdMsg.Payload.(ResolveFile)

	newRootPath, modulePathParts, err := calculateModulePath(payload.Path, r.RootPath)
	name, path, data, err := r.resolveModule(ctx, newRootPath, modulePathParts)

	svc.Reply(ctx, fwdMsg, ResolvedResult{
		ModuleName: name,
		ModulePath: path,
		Data:       data,
		Error:      err,
	})

	return kernel.Terminate{}
}

func (r *ResolveWorker) resolveModule(ctx *kernel.ActCtx, rootPath string, pathParts []string) (string, string, string, error) {

	fsId, _ := ctx.K.ActorByName(svc.FsService)

	moduleName := strings.Join(pathParts, ".")

	slog.Info("Loading module",
		slog.Any("moduleName", moduleName),
		slog.Any("pathParts", pathParts),
		slog.Any("rootPath", rootPath))

	// Complete the module path
	moduleRelativePath := strings.Join(pathParts, "/")
	modulePath := fmt.Sprintf("%s/%s.slug", rootPath, moduleRelativePath)

	// Attempt to load the module's source
	fsResponse, err := ctx.SendSync(fsId, fs.Read{Path: modulePath})
	if err != nil {
		slog.Info("Failed to read file", slog.Any("error", err))
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

func (r *ResolveWorker) slugLibPath(moduleName string, moduleRelativePath string) (string, error) {
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
