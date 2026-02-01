package runtime

import "slug/internal/object"

func (e *Task) recvResult(value object.Object, ok bool) object.Object {
	if ok {
		return &object.StructValue{
			Schema: e.Runtime.FullSchema,
			Fields: map[string]object.Object{"value": value},
		}
	}
	return &object.StructValue{
		Schema: e.Runtime.EmptySchema,
		Fields: map[string]object.Object{},
	}
}

func (e *Task) cancellationError() *object.RuntimeError {
	select {
	case <-e.Done:
		return e.Err
	default:
		return nil
	}
}
