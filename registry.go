package glue

import (
	"fmt"
	"reflect"
	"sync"
)

type (
	// Registry interface.
	// Deprecated: use Context instead of Registry. Will be removed in 3.0.
	Registry interface {
		Get(name string) interface{}
		Set(name string, value interface{})
		Fill(name string, value interface{}) error
	}

	// registry is implementation of Registry.
	registry struct {
		mux    sync.RWMutex
		values map[string]interface{}
	}
)

// newRegistry is Registry constructor.
func newRegistry() Registry {
	return &registry{
		values: make(map[string]interface{}),
	}
}

// Get implements Registry.
func (p *registry) Get(name string) interface{} {
	p.mux.RLock()
	defer p.mux.RUnlock()

	return p.values[name]
}

// Set implements Registry.
func (p *registry) Set(name string, value interface{}) {
	p.mux.Lock()
	defer p.mux.Unlock()

	p.values[name] = value
}

// Fill implements Registry.
func (p *registry) Fill(name string, target interface{}) (err error) {
	var src interface{}
	defer func() {
		if r := recover(); r != nil {
			t := reflect.TypeOf(target)
			s := reflect.TypeOf(src)
			err = fmt.Errorf("target is `%s` but should be a pointer to the source type `%s`", t, s)
		}
	}()

	src = p.Get(name)
	reflect.ValueOf(target).Elem().Set(reflect.ValueOf(src))
	return
}
