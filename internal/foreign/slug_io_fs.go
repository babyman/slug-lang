package foreign

import (
	"bufio"
	"io"
	"math/rand"
	"os"
	"slug/internal/object"
	"sync"
)

var (
	ioFileFiles         = map[int64]*os.File{}
	ioFileReaders       = map[int64]*bufio.Reader{}
	ioFileNextID  int64 = 1
	ioFileMutex   sync.Mutex
)

func nextIoFileId() int64 {
	ioFileMutex.Lock()
	defer ioFileMutex.Unlock()
	id := ioFileNextID<<16 | int64(rand.Intn(0xFFFF))
	ioFileNextID++
	return id
}

func fnIoFsReadFile() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments to `readFile`, got=%d, want=1", len(args))
			}

			path, err := unpackString(args[0], "path")
			if err != nil {
				return ctx.NewError(err.Error())
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
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 2 {
				return ctx.NewError("wrong number of arguments to `writeFile`, got=%d, want=2", len(args))
			}

			path, err := unpackString(args[0], "path")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			content, err := unpackString(args[1], "contents")
			if err != nil {
				return ctx.NewError(err.Error())
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
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 2 {
				return ctx.NewError("wrong number of arguments to `appendFile`, got=%d, want=2", len(args))
			}

			path, err := unpackString(args[0], "path")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			content, err := unpackString(args[1], "contents")
			if err != nil {
				return ctx.NewError(err.Error())
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

			return &object.Integer{Value: int64(bytes)}
		},
	}
}

func fnIoFsExists() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments to `fileExists`, got=%d, want=1", len(args))
			}

			path, err := unpackString(args[0], "path")
			if err != nil {
				return ctx.NewError(err.Error())
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
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments to `isDirectory`, got=%d, want=1", len(args))
			}

			path, err := unpackString(args[0], "path")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			info, err := os.Stat(path)
			if err != nil {
				return ctx.NewError("failed to get file info: %s", err.Error())
			}
			m := &object.Map{}
			m.Put(&object.String{Value: "name"}, &object.String{Value: info.Name()}).
				Put(&object.String{Value: "size"}, &object.Integer{Value: info.Size()}).
				Put(&object.String{Value: "mode"}, &object.Integer{Value: int64(info.Mode())}).
				Put(&object.String{Value: "modTime"}, &object.Integer{Value: info.ModTime().Unix()}).
				Put(&object.String{Value: "isDir"}, ctx.NativeBoolToBooleanObject(info.IsDir()))
			return m
		},
	}
}

func fnIoFsIsDir() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments to `isDirectory`, got=%d, want=1", len(args))
			}

			path, err := unpackString(args[0], "path")
			if err != nil {
				return ctx.NewError(err.Error())
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
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments to `listDir`, got=%d, want=1", len(args))
			}

			path, err := unpackString(args[0], "path")
			if err != nil {
				return ctx.NewError(err.Error())
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
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 2 {
				return ctx.NewError("wrong number of arguments to `openFile`, got=%d, want=2", len(args))
			}

			path, err := unpackString(args[0], "path")
			if err != nil {
				return ctx.NewError(err.Error())
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

			fileID := nextIoFileId()
			ioFileFiles[fileID] = file

			return &object.Integer{Value: fileID}
		},
	}
}

func fnIoFsReadLine() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments to `readLine`, got=%d, want=1", len(args))
			}

			handle, err := unpackInt(args[0], "handle")
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
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 2 {
				return ctx.NewError("wrong number of arguments to `write`, got=%d, want=2", len(args))
			}

			handle, err := unpackInt(args[0], "handle")
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

			return &object.Integer{Value: int64(bytes)}
		},
	}
}

func fnIoFsRm() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments to `rm`, got=%d, want=1", len(args))
			}

			path, err := unpackString(args[0], "path")
			if err != nil {
				return ctx.NewError(err.Error())
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
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments to `closeFile`, got=%d, want=1", len(args))
			}

			handle, err := unpackInt(args[0], "handle")
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
