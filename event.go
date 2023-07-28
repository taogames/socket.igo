package socketigo

import (
	"fmt"
	"reflect"
)

type EventManager struct {
	m map[string]*handler
}

type handler struct {
	f     reflect.Value
	types []reflect.Type
}

func (eh *EventManager) Register(eName string, h any) {
	rv := reflect.ValueOf(h)
	if rv.Kind() != reflect.Func {
		panic(fmt.Sprintln("reflect kind is ", rv.Kind()))
	}

	rt := rv.Type()
	types := make([]reflect.Type, rt.NumIn())
	for i := 0; i < rt.NumIn(); i++ {
		types[i] = rt.In(i)
	}

	eh.m[eName] = &handler{
		f:     rv,
		types: types,
	}
}

func (eh *EventManager) GetHandler(eName string) *handler {
	return eh.m[eName]
}
