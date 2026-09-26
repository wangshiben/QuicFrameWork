package RouteDisPatch

import (
	"encoding/json"
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

// TODO: 当session大小超出的时候要throw ERROR
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
	if !sessionMap.StoreSession(key, session) {
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
