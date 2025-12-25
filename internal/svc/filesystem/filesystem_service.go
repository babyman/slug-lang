package filesystem

import (
	"bufio"
	"io"
	"os"
	"reflect"
	"slug/internal/dec64"
	"slug/internal/kernel"
	"slug/internal/object"
	"slug/internal/svc"
	"slug/internal/svc/svcutil"
	"strings"
)

var Operations = kernel.OpRights{
	reflect.TypeOf(svc.SlugActorMessage{}): kernel.RightWrite,
}

var (
	pathKey = (&object.String{Value: "path"}).MapKey()
	modeKey = (&object.String{Value: "mode"}).MapKey()
)

type Service struct{}

type FileHandler struct {
	file   *os.File
	reader *bufio.Reader
}

func (s *Service) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	p, ok := msg.Payload.(svc.SlugActorMessage)
	if !ok {
		return kernel.Continue{}
	}

	to := svcutil.ReplyTarget(msg)
	m, ok := p.Msg.(*object.Map)
	if !ok {
		ctx.SendAsync(to, svcutil.ErrorResult("map expected"))
		return kernel.Continue{}
	}

	msgType, ok := svcutil.GetString(m, svcutil.MsgTypeKey)
	if !ok {
		ctx.SendAsync(to, svcutil.ErrorResult("missing type"))
		return kernel.Continue{}
	}

	switch msgType {
	case "readFile":
		path, _ := svcutil.GetString(m, pathKey)
		data, err := os.ReadFile(path)
		if err != nil {
			ctx.SendAsync(to, svcutil.ErrorResult(err.Error()))
		} else {
			ctx.SendAsync(to, readFileResult(data))
		}

	case "writeFile":
		path, _ := svcutil.GetString(m, pathKey)
		mode := svcutil.GetStringWithDefault(m, modeKey, "w")
		fileData, _ := svcutil.GetObj(m, svcutil.DataKey)
		var data []byte
		switch c := fileData.(type) {
		case *object.String:
			data = []byte(c.Value)
		case *object.Bytes:
			data = c.Value
		default:
			ctx.SendAsync(to, svcutil.ErrorResult("string or bytes expected"))
			return kernel.Continue{}
		}

		flag := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		if strings.Contains(mode, "a") {
			flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
		}

		f, err := os.OpenFile(path, flag, 0644)
		if err != nil {
			ctx.SendAsync(to, svcutil.ErrorResult(err.Error()))
		}
		defer f.Close()
		bytes, err := f.Write(data)

		if err != nil {
			ctx.SendAsync(to, svcutil.ErrorResult(err.Error()))
		} else {
			ctx.SendAsync(to, writeFileResult(bytes))
		}

	case "info":
		path, _ := svcutil.GetString(m, pathKey)
		info, err := os.Stat(path)
		if err != nil {
			ctx.SendAsync(to, svcutil.ErrorResult(err.Error()))
			return kernel.Continue{}
		}
		ctx.SendAsync(to, infoResult(info))

	case "exists":
		path, _ := svcutil.GetString(m, pathKey)
		_, err := os.Stat(path)
		exists := !os.IsNotExist(err)
		ctx.SendAsync(to, existsResult(exists))

	case "mkdirs":
		path, _ := svcutil.GetString(m, pathKey)
		err := os.MkdirAll(path, 0755)
		if err != nil {
			ctx.SendAsync(to, svcutil.ErrorResult(err.Error()))
		} else {
			ctx.SendAsync(to, SuccessResult("mkdirs"))
		}

	case "ls":
		path, _ := svcutil.GetString(m, pathKey)
		entries, err := os.ReadDir(path)
		if err != nil {
			ctx.SendAsync(to, svcutil.ErrorResult(err.Error()))
		} else {
			var result []object.Object
			for _, entry := range entries {
				result = append(result, &object.String{Value: entry.Name()})
			}
			ctx.SendAsync(to, lsResult(result))
		}

	case "rm":
		path, _ := svcutil.GetString(m, pathKey)
		err := os.Remove(path)
		if err != nil {
			ctx.SendAsync(to, svcutil.ErrorResult(err.Error()))
		} else {
			ctx.SendAsync(to, SuccessResult("rm"))
		}

	case "open":
		path, _ := svcutil.GetString(m, pathKey)
		handler := &FileHandler{}
		id, err := ctx.SpawnChild("file-handle: "+path, Operations, handler.Handler)
		if err != nil {
			ctx.SendAsync(to, svcutil.ErrorResult(err.Error()))
			return kernel.Continue{}
		}
		ctx.GrantChildAccess(msg.From, id, kernel.RightWrite, nil)
		ctx.ForwardAsync(id, msg)
	}

	return kernel.Continue{}
}

func (h *FileHandler) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	if _, ok := msg.Payload.(kernel.Shutdown); ok {
		if h.file != nil {
			h.file.Close()
		}
		return kernel.Terminate{Reason: "shutdown"}
	}

	p, ok := msg.Payload.(svc.SlugActorMessage)
	if !ok {
		return kernel.Continue{}
	}

	to := svcutil.ReplyTarget(msg)
	m, _ := p.Msg.(*object.Map)
	msgType, _ := svcutil.GetString(m, svcutil.MsgTypeKey)

	switch msgType {
	case "open":
		path, _ := svcutil.GetString(m, pathKey)
		mode := svcutil.GetStringWithDefault(m, modeKey, "w")

		flag := os.O_RDONLY
		if strings.Contains(mode, "a") {
			flag = os.O_APPEND | os.O_WRONLY | os.O_CREATE
		} else if strings.Contains(mode, "w") {
			flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		}

		f, err := os.OpenFile(path, flag, 0644)
		if err != nil {
			ctx.SendAsync(to, svcutil.ErrorResult(err.Error()))
			return kernel.Terminate{Reason: err.Error()}
		}
		h.file = f
		ctx.SendAsync(to, svc.SlugActorMessage{Msg: &object.Number{
			Value: dec64.FromInt64(int64(ctx.Self)),
		}})

	case "read":
		if h.file == nil {
			ctx.SendAsync(to, svcutil.ErrorResult("file not open"))
			return kernel.Continue{}
		}
		maxSize := svcutil.GetIntWithDefault(m, svcutil.MaxKey, 4096)
		buf := make([]byte, maxSize)
		n, err := h.file.Read(buf)
		if err != nil {
			if err == io.EOF {
				ctx.SendAsync(to, EOFResult())
			} else {
				ctx.SendAsync(to, svcutil.ErrorResult(err.Error()))
			}
		} else {
			ctx.SendAsync(to, readFileResult(buf[:n]))
		}

	case "readLine":
		if h.file == nil {
			ctx.SendAsync(to, svcutil.ErrorResult("file not open"))
			return kernel.Continue{}
		}
		if h.reader == nil {
			h.reader = bufio.NewReader(h.file)
		}
		line, err := h.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				ctx.SendAsync(to, EOFResult())
			} else {
				ctx.SendAsync(to, svcutil.ErrorResult(err.Error()))
			}
		} else {
			ctx.SendAsync(to, readLineResult(line))
		}

	case "write":
		if h.file == nil {
			ctx.SendAsync(to, svcutil.ErrorResult("file not open"))
			return kernel.Continue{}
		}
		dataObj, _ := svcutil.GetObj(m, svcutil.DataKey)
		var data []byte
		switch d := dataObj.(type) {
		case *object.String:
			data = []byte(d.Value)
		case *object.Bytes:
			data = d.Value
		default:
			ctx.SendAsync(to, svcutil.ErrorResult("string or bytes expected"))
			return kernel.Continue{}
		}
		n, err := h.file.Write(data)
		if err != nil {
			ctx.SendAsync(to, svcutil.ErrorResult(err.Error()))
		} else {
			ctx.SendAsync(to, svc.SlugActorMessage{Msg: &object.Number{Value: dec64.FromInt64(int64(n))}})
		}

	case "close":
		if h.file != nil {
			h.file.Close()
		}
		ctx.SendAsync(to, svcutil.CloseResult(ctx.Self))
		return kernel.Terminate{Reason: "closed"}
	}

	return kernel.Continue{}
}

func lsResult(result []object.Object) svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	svcutil.PutString(resultMap, "type", "ls")
	svcutil.PutList(resultMap, "entries", result)
	return svc.SlugActorMessage{Msg: resultMap}
}

func infoResult(info os.FileInfo) svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	svcutil.PutString(resultMap, "type", "info")
	svcutil.PutString(resultMap, "name", info.Name())
	svcutil.PutInt64(resultMap, "size", info.Size())
	svcutil.PutInt(resultMap, "mode", int(info.Mode()))
	svcutil.PutInt64(resultMap, "modTime", info.ModTime().Unix())
	svcutil.PutBool(resultMap, "isDir", info.IsDir())
	return svc.SlugActorMessage{Msg: resultMap}
}

func existsResult(exists bool) svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	svcutil.PutString(resultMap, "type", "exists")
	svcutil.PutBool(resultMap, "exists", exists)
	return svc.SlugActorMessage{Msg: resultMap}
}

func readLineResult(line string) svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	svcutil.PutString(resultMap, "type", "readLine")
	svcutil.PutString(resultMap, "line", line)
	return svc.SlugActorMessage{Msg: resultMap}
}

func readFileResult(data []byte) svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	svcutil.PutString(resultMap, "type", "readFile")
	svcutil.PutBytes(resultMap, "bytes", data)
	return svc.SlugActorMessage{Msg: resultMap}
}

func writeFileResult(bytesWritten int) svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	svcutil.PutString(resultMap, "type", "writeFile")
	svcutil.PutInt(resultMap, "written", bytesWritten)
	return svc.SlugActorMessage{Msg: resultMap}
}

func SuccessResult(typeStr string) svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	svcutil.PutString(resultMap, "type", typeStr)
	return svc.SlugActorMessage{Msg: resultMap}
}

func EOFResult() svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	svcutil.PutString(resultMap, "type", "eof")
	return svc.SlugActorMessage{Msg: resultMap}
}
