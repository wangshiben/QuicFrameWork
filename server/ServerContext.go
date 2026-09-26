package server

import (
	"context"
	"time"
)

type Context struct {
	valueMap map[string]any
	parent   context.Context
}

func (c *Context) Deadline() (deadline time.Time, ok bool) {
	if c.parent == nil {
		return time.Time{}, false
	}
	return c.parent.Deadline()
}

func (c *Context) Done() <-chan struct{} {
	if c.parent == nil {
		return nil
	}
	return c.parent.Done()
}

func (c *Context) Err() error {
	if c.parent == nil {
		return nil
	}
	return c.parent.Err()
}

func (c *Context) Value(key any) any {

	switch key.(type) {
	case string:
		Val := c.valueMap[key.(string)]
		if Val == nil {
			if c.parent == nil {
				return nil
			}
			return c.parent.Value(key)
		}
		return Val
	//break
	default:
		if c.parent == nil {
			return nil
		}
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
func (c *Context) reset() {
	c.parent = nil
}

func withParent(parent context.Context) *Context {
	return &Context{
		valueMap: make(map[string]any),
		parent:   parent,
	}
}
