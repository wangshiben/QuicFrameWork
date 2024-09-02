package server

import (
	"context"
	"time"
)

type Context struct {
	valueMap map[string]any
	parent   context.Context
}

func (Context) Deadline() (deadline time.Time, ok bool) {
	return
}

func (Context) Done() <-chan struct{} {
	return nil
}

func (c *Context) Err() error {
	return nil
}

func (c *Context) Value(key any) any {

	switch key.(type) {
	case string:
		Val := c.valueMap[key.(string)]
		if Val == nil {
			return c.parent.Value(key)
		}
		return Val
	//break
	default:
		return c.parent.Value(key)
		//return nil
	}
	//return nil
}
func (c *Context) SetValue(key string, val any) {
	c.valueMap[key] = val
}
func (Context) String() string {
	return "context.Service"
}

func withParent(parent context.Context) *Context {
	return &Context{
		valueMap: make(map[string]any),
		parent:   parent,
	}
}
