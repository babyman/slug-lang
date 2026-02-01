package runtime

import (
	"errors"
	"reflect"
	"slug/internal/ast"
	"slug/internal/object"
	"time"
)

type selectEvalCase struct {
	kind       ast.SelectCaseType
	token      int
	channel    *object.Channel
	sendValue  object.Object
	afterValue object.Object
	afterTimer *time.Timer
	awaitTask  *Task
	handler    ast.Expression
	closedSend bool
}

func (e *Task) evalSelectExpression(node *ast.SelectExpression) object.Object {
	if err := e.cancellationError(); err != nil {
		return err
	}
	if len(node.Cases) == 0 {
		return e.newErrorWithPos(node.Token.Position, "select requires at least one case")
	}

	cases := make([]reflect.SelectCase, 0, len(node.Cases)+1)
	evals := make([]selectEvalCase, 0, len(node.Cases))
	awaitTasks := make([]*Task, 0, len(node.Cases))
	defaultSeen := false
	readySignal := make(chan struct{})
	close(readySignal)
	defer func() {
		for _, ev := range evals {
			if ev.afterTimer != nil {
				ev.afterTimer.Stop()
			}
		}
	}()

	for _, c := range node.Cases {
		switch c.Kind {
		case ast.SelectRecv:
			chObj := e.Eval(c.Channel)
			if e.isError(chObj) {
				return chObj
			}
			ch, ok := chObj.(*object.Channel)
			if !ok {
				return e.newErrorfWithPos(c.Token.Position, "recv expects a channel, got %s", chObj.Type())
			}
			cases = append(cases, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(ch.GoChan()),
			})
			evals = append(evals, selectEvalCase{
				kind:    ast.SelectRecv,
				token:   c.Token.Position,
				channel: ch,
				handler: c.Handler,
			})
		case ast.SelectSend:
			chObj := e.Eval(c.Channel)
			if e.isError(chObj) {
				return chObj
			}
			ch, ok := chObj.(*object.Channel)
			if !ok {
				return e.newErrorfWithPos(c.Token.Position, "send expects a channel, got %s", chObj.Type())
			}
			val := e.Eval(c.Value)
			if e.isError(val) {
				return val
			}
			if ch.IsClosed() {
				cases = append(cases, reflect.SelectCase{
					Dir:  reflect.SelectRecv,
					Chan: reflect.ValueOf(readySignal),
				})
				evals = append(evals, selectEvalCase{
					kind:       ast.SelectSend,
					token:      c.Token.Position,
					channel:    ch,
					sendValue:  val,
					handler:    c.Handler,
					closedSend: true,
				})
				continue
			}
			cases = append(cases, reflect.SelectCase{
				Dir:  reflect.SelectSend,
				Chan: reflect.ValueOf(ch.GoChan()),
				Send: reflect.ValueOf(val),
			})
			evals = append(evals, selectEvalCase{
				kind:      ast.SelectSend,
				token:     c.Token.Position,
				channel:   ch,
				sendValue: val,
				handler:   c.Handler,
			})
		case ast.SelectAfter:
			afterVal := e.Eval(c.After)
			if e.isError(afterVal) {
				return afterVal
			}
			num, ok := afterVal.(*object.Number)
			if !ok {
				return e.newErrorfWithPos(c.Token.Position, "after expects a number (ms)")
			}
			ms := num.Value.ToInt64()
			if ms < 0 {
				return e.newErrorfWithPos(c.Token.Position, "after expects a non-negative duration")
			}
			if ms > 0 {
				// if ms == 0 no timeout is expected, use `_` default for zero timeout
				timer := time.NewTimer(time.Duration(ms) * time.Millisecond)
				cases = append(cases, reflect.SelectCase{
					Dir:  reflect.SelectRecv,
					Chan: reflect.ValueOf(timer.C),
				})
				evals = append(evals, selectEvalCase{
					kind:       ast.SelectAfter,
					token:      c.Token.Position,
					afterValue: afterVal,
					afterTimer: timer,
					handler:    c.Handler,
				})
			}
		case ast.SelectAwait:
			taskObj := e.Eval(c.Await)
			if e.isError(taskObj) {
				return taskObj
			}
			task, ok := taskObj.(*Task)
			if !ok {
				return e.newErrorfWithPos(c.Token.Position, "await expects a task handle, got %s", taskObj.Type())
			}
			cases = append(cases, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(task.Done),
			})
			evals = append(evals, selectEvalCase{
				kind:      ast.SelectAwait,
				token:     c.Token.Position,
				awaitTask: task,
				handler:   c.Handler,
			})
			awaitTasks = append(awaitTasks, task)
		case ast.SelectDefault:
			if defaultSeen {
				return e.newErrorWithPos(c.Token.Position, "select cannot have multiple default cases")
			}
			defaultSeen = true
			cases = append(cases, reflect.SelectCase{Dir: reflect.SelectDefault})
			evals = append(evals, selectEvalCase{
				kind:    ast.SelectDefault,
				token:   c.Token.Position,
				handler: c.Handler,
			})
		default:
			return e.newErrorWithPos(c.Token.Position, "invalid select case")
		}
	}

	cancelIndex := len(cases)
	cases = append(cases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(e.Done),
	})

	if err := e.cancellationError(); err != nil {
		return err
	}

	chosen, recv, ok, selErr := e.safeSelect(cases)
	if selErr != nil {
		return e.newErrorWithPos(node.Token.Position, selErr.Error())
	}

	for i, ev := range evals {
		if ev.afterTimer == nil || i == chosen {
			continue
		}
		if !ev.afterTimer.Stop() {
			select {
			case <-ev.afterTimer.C:
			default:
			}
		}
	}

	if chosen == cancelIndex {
		if err := e.cancellationError(); err != nil {
			return err
		}
		return e.newErrorWithPos(node.Token.Position, "task cancelled")
	}

	selected := evals[chosen]
	if len(awaitTasks) > 0 {
		var keep *Task
		if selected.kind == ast.SelectAwait {
			keep = selected.awaitTask
		}
		for _, task := range awaitTasks {
			if keep != nil && task == keep {
				continue
			}
			task.Cancel(nil, "select cancelled")
		}
	}
	switch selected.kind {
	case ast.SelectRecv:
		var val object.Object
		if ok {
			val, _ = recv.Interface().(object.Object)
		}
		msg := e.recvResult(val, ok)
		return e.evalSelectHandler(selected.token, selected.handler, msg)
	case ast.SelectSend:
		if selected.closedSend {
			return e.runtimeErrorAt(selected.token, "send", map[string]object.Object{
				"message": &object.String{Value: "send on closed channel"},
			})
		}
		return e.evalSelectHandler(selected.token, selected.handler, selected.sendValue)
	case ast.SelectAfter, ast.SelectDefault:
		val := selected.afterValue
		if selected.kind == ast.SelectDefault {
			val = object.NIL
		}
		return e.evalSelectHandler(selected.token, selected.handler, val)
	case ast.SelectAwait:
		if selected.awaitTask.Err != nil {
			return selected.awaitTask.Err
		}
		val := selected.awaitTask.Result
		if val == nil {
			val = object.NIL
		}
		return e.evalSelectHandler(selected.token, selected.handler, val)
	default:
		return e.newErrorWithPos(node.Token.Position, "invalid select case")
	}
}

func (e *Task) evalSelectHandler(pos int, handler ast.Expression, value object.Object) object.Object {
	if handler == nil {
		return value
	}
	switch h := handler.(type) {
	case *ast.MatchExpression:
		if h.Value != nil {
			return e.runtimeErrorAt(pos, "select", map[string]object.Object{
				"message": &object.String{Value: "select case match must be subjectless"},
			})
		}
		return e.evalMatchExpressionWithValue(h, value)
	case *ast.CallExpression:
		fn := e.Eval(h.Function)
		if e.isError(fn) {
			return fn
		}
		positional, named, errObj := e.evalCallArguments(pos, h.Arguments)
		if errObj != nil {
			return errObj
		}
		positional = append([]object.Object{value}, positional...)
		return e.applySelectFunction(pos, fn, positional, named)
	default:
		fn := e.Eval(handler)
		if e.isError(fn) {
			return fn
		}
		return e.applySelectFunction(pos, fn, []object.Object{value}, nil)
	}
}

func (e *Task) applySelectFunction(pos int, fnObj object.Object, positional []object.Object, named map[string]object.Object) object.Object {
	fnObj = e.resolveValue(pos, fnObj)
	if e.isError(fnObj) {
		return fnObj
	}

	result := e.ApplyFunction(pos, "select", fnObj, positional, named)
	if errObj, ok := result.(*object.Error); ok {
		return e.runtimeErrorAt(pos, "select", map[string]object.Object{
			"message": &object.String{Value: errObj.Message},
		})
	}
	return result
}

func (e *Task) safeSelect(cases []reflect.SelectCase) (chosen int, recv reflect.Value, ok bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("select send on closed channel")
		}
	}()
	chosen, recv, ok = reflect.Select(cases)
	return
}
