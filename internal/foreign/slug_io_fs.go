package foreign

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"slug/internal/dec64"
	"slug/internal/object"
)

var (
	ioFileFiles   = map[int64]*os.File{}
	ioFileReaders = map[int64]*bufio.Reader{}
)

func fnIoFsReadFile() *object.Foreign {
	return &object.Foreign{
		Name: "readFile",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments to `readFile`, got=%d, want=1", len(args))
			}

			path, err := unpackString(args[0], "path")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			if !filepath.IsAbs(path) {
				path = filepath.Join(ctx.GetConfiguration().RootPath, path)
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return ctx.NewError("failed to read file: %s", err.Error())
			}

			return &object.String{Value: string(data)}
		},
	}
}

func fnIoFsWriteFile() *object.Foreign {
	return &object.Foreign{
		Name: "writeFile",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 2 {
				return ctx.NewError("wrong number of arguments to `writeFile`, got=%d, want=2", len(args))
			}

			content, err := unpackString(args[0], "contents")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			path, err := unpackString(args[1], "path")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			if !filepath.IsAbs(path) {
				path = filepath.Join(ctx.GetConfiguration().RootPath, path)
			}

			err = os.WriteFile(path, []byte(content), 0644)
			if err != nil {
				return ctx.NewError("failed to write file: %s", err.Error())
			}

			return ctx.Nil()
		},
	}
}

func fnIoFsAppendFile() *object.Foreign {
	return &object.Foreign{
		Name: "appendFile",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 2 {
				return ctx.NewError("wrong number of arguments to `appendFile`, got=%d, want=2", len(args))
			}

			content, err := unpackString(args[0], "contents")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			path, err := unpackString(args[1], "path")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			if !filepath.IsAbs(path) {
				path = filepath.Join(ctx.GetConfiguration().RootPath, path)
			}

			f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				return ctx.NewError("failed to open file: %s", err.Error())
			}
			defer f.Close()

			bytes, err := f.WriteString(content)
			if err != nil {
				return ctx.NewError("failed to append to file: %s", err.Error())
			}

			return &object.Number{Value: dec64.FromInt64(int64(bytes))}
		},
	}
}

func fnIoFsExists() *object.Foreign {
	return &object.Foreign{
		Name: "exists",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments to `fileExists`, got=%d, want=1", len(args))
			}

			path, err := unpackString(args[0], "path")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			if !filepath.IsAbs(path) {
				path = filepath.Join(ctx.GetConfiguration().RootPath, path)
			}

			_, err = os.Stat(path)
			if os.IsNotExist(err) {
				return ctx.NativeBoolToBooleanObject(false)
			}

			return ctx.NativeBoolToBooleanObject(true)
		},
	}
}

func fnIoFsInfo() *object.Foreign {
	return &object.Foreign{
		Name: "info",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments to `isDirectory`, got=%d, want=1", len(args))
			}

			path, err := unpackString(args[0], "path")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			if !filepath.IsAbs(path) {
				path = filepath.Join(ctx.GetConfiguration().RootPath, path)
			}

			info, err := os.Stat(path)
			if err != nil {
				return ctx.NewError("failed to get file info: %s", err.Error())
			}
			m := &object.Map{}
			m.Put(&object.String{Value: "name"}, &object.String{Value: info.Name()}).
				Put(&object.String{Value: "size"}, &object.Number{Value: dec64.FromInt64(info.Size())}).
				Put(&object.String{Value: "mode"}, &object.Number{Value: dec64.FromInt64(int64(info.Mode()))}).
				Put(&object.String{Value: "modTime"}, &object.Number{Value: dec64.FromInt64(info.ModTime().Unix())}).
				Put(&object.String{Value: "isDir"}, ctx.NativeBoolToBooleanObject(info.IsDir()))
			return m
		},
	}
}

func fnIoFsMkdirs() *object.Foreign {
	return &object.Foreign{
		Name: "mkDirs",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments to `mkDirs`, got=%d, want=1", len(args))
			}

			path, err := unpackString(args[0], "path")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			if !filepath.IsAbs(path) {
				path = filepath.Join(ctx.GetConfiguration().RootPath, path)
			}

			created := false
			if _, err := os.Stat(path); os.IsNotExist(err) {
				created = true
			}

			err = os.MkdirAll(path, 0755)
			if err != nil {
				return ctx.NewError("failed to create directories: %s", err.Error())
			}

			return ctx.NativeBoolToBooleanObject(created)
		},
	}
}

func fnIoFsIsDir() *object.Foreign {
	return &object.Foreign{
		Name: "isDir",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments to `isDir`, got=%d, want=1", len(args))
			}

			path, err := unpackString(args[0], "path")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			if !filepath.IsAbs(path) {
				path = filepath.Join(ctx.GetConfiguration().RootPath, path)
			}

			info, err := os.Stat(path)
			if err != nil {
				return ctx.NewError("failed to get file info: %s", err.Error())
			}

			return ctx.NativeBoolToBooleanObject(info.IsDir())
		},
	}
}

func fnIoFsLs() *object.Foreign {
	return &object.Foreign{
		Name: "ls",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments to `listDir`, got=%d, want=1", len(args))
			}

			path, err := unpackString(args[0], "path")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			if !filepath.IsAbs(path) {
				path = filepath.Join(ctx.GetConfiguration().RootPath, path)
			}

			files, err := os.ReadDir(path)
			if err != nil {
				return ctx.NewError("failed to read directory: %s", err.Error())
			}

			var result []object.Object
			for _, file := range files {
				result = append(result, &object.String{Value: file.Name()})
			}

			return &object.List{Elements: result}
		},
	}
}

func fnIoFsOpenFile() *object.Foreign {
	return &object.Foreign{
		Name: "openFile",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 2 {
				return ctx.NewError("wrong number of arguments to `openFile`, got=%d, want=2", len(args))
			}

			path, err := unpackString(args[0], "path")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			if !filepath.IsAbs(path) {
				path = filepath.Join(ctx.GetConfiguration().RootPath, path)
			}

			mode, err := unpackString(args[1], "mode")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			flag := os.O_RDONLY
			if mode == "w" {
				flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
			} else if mode == "a" {
				flag = os.O_APPEND | os.O_WRONLY | os.O_CREATE
			}

			file, err := os.OpenFile(path, flag, 0644)
			if err != nil {
				return ctx.NewError("failed to open file: %s", err.Error())
			}

			fileID := ctx.NextHandleID()
			ioFileFiles[fileID] = file

			return &object.Number{Value: dec64.FromInt64(fileID)}
		},
	}
}

func fnIoFsReadLine() *object.Foreign {
	return &object.Foreign{
		Name: "readLine",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments to `readLine`, got=%d, want=1", len(args))
			}

			handle, err := unpackNumber(args[0], "handle")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			file, ok := ioFileFiles[handle]
			if !ok {
				return ctx.NewError("invalid file handle: %d", handle)
			}

			reader, ok := ioFileReaders[handle]
			if !ok {
				reader = bufio.NewReader(file)
				ioFileReaders[handle] = reader
			}
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return ctx.Nil()
				} else {
					return ctx.NewError("failed to read line: %s", err.Error())
				}
			}

			return &object.String{Value: line}
		},
	}
}

func fnIoFsWrite() *object.Foreign {
	return &object.Foreign{
		Name: "write",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 2 {
				return ctx.NewError("wrong number of arguments to `write`, got=%d, want=2", len(args))
			}

			handle, err := unpackNumber(args[0], "handle")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			content, err := unpackString(args[1], "content")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			file, ok := ioFileFiles[handle]
			if !ok {
				return ctx.NewError("invalid file handle: %d", handle)
			}

			bytes, err := file.WriteString(content)
			if err != nil {
				return ctx.NewError("failed to write to file: %s", err.Error())
			}

			return &object.Number{Value: dec64.FromInt64(int64(bytes))}
		},
	}
}

func fnIoFsRm() *object.Foreign {
	return &object.Foreign{
		Name: "rm",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments to `rm`, got=%d, want=1", len(args))
			}

			path, err := unpackString(args[0], "path")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			if !filepath.IsAbs(path) {
				path = filepath.Join(ctx.GetConfiguration().RootPath, path)
			}

			err = os.Remove(path)
			if err != nil {
				return ctx.NewError("failed to remove file: %s", err.Error())
			}

			return ctx.Nil()
		},
	}
}

func fnIoFsCloseFile() *object.Foreign {
	return &object.Foreign{
		Name: "closeFile",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments to `closeFile`, got=%d, want=1", len(args))
			}

			handle, err := unpackNumber(args[0], "handle")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			file, ok := ioFileFiles[handle]
			if !ok {
				return ctx.NewError("invalid file handle: %d", handle)
			}

			err = file.Close()
			if err != nil {
				return ctx.NewError("failed to close file: %s", err.Error())
			}

			delete(ioFileReaders, handle)
			delete(ioFileFiles, handle)
			return ctx.Nil()
		},
	}
}
