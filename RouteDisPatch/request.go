package RouteDisPatch

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/wangshiben/QuicFrameWork/Session"
	"github.com/wangshiben/QuicFrameWork/consts"
	"net/http"
)

const quicSessionName = "quickSession"

type Request struct {
	writer  http.ResponseWriter
	req     *http.Request
	Request *http.Request
	Param   interface{}
	session Session.ItemInterFace
}

// GetSession returns an existing session or creates one within the configured memory limit.
func (r *Request) GetSession() (Session.ItemInterFace, error) {
	//防止多处函数引用导致session重复读取
	if r.session != nil {
		return r.session, nil
	}
	context := r.req.Context()
	sessionMap, ok := context.Value(consts.GetSession).(Session.ServerSession)
	if !ok || sessionMap == nil {
		return nil, fmt.Errorf("session store is not configured in the request context")
	}
	initFunc, ok := context.Value(consts.InitSessionFunc).(Session.GenerateItemInterFace)
	if !ok || initFunc == nil {
		return nil, fmt.Errorf("session initializer is not configured in the request context")
	}
	key, exist := sessionMap.GetKeyFromRequest(r.req)
	if exist {
		item := sessionMap.GetItem(key)
		if item != nil {
			r.session = item
			return item, nil
		}
	}
	key, session := sessionMap.GenerateName()(initFunc)
	maxMemory, _ := context.Value(consts.MaxSessionMemo).(int64)
	if limitedSession, ok := sessionMap.(Session.MemoryLimitedServerSession); ok && maxMemory > 0 {
		err := limitedSession.StoreSessionWithinLimit(key, session, maxMemory)
		if errors.Is(err, Session.MaxMemo) {
			sessionMap.CleanExpItem()
			err = limitedSession.StoreSessionWithinLimit(key, session, maxMemory)
		}
		if err != nil {
			return nil, err
		}
	} else if !sessionMap.StoreSession(key, session) {
		return nil, fmt.Errorf("failed to store generated session")
	}
	sessionMap.SetKeyToResponse()(r.writer, key)
	r.session = session
	return session, nil

}
func (r *Request) GetParam() interface{} {
	return r.Param
}
func (r *Request) GetRequest() *http.Request {
	return r.req
}
func (r *Request) ToJson(v any) error {
	marshal, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = r.writer.Write(marshal)
	return err
}

func NewRequest(r *http.Request, w http.ResponseWriter) *Request {
	return &Request{
		req:     r,
		Request: r,
		writer:  w,
	}
}
