package RouteDisPatch

import (
	"github.com/wangshiben/QuicFrameWork/server/Session"
	"github.com/wangshiben/QuicFrameWork/server/consts"
	"net/http"
)

const quicSessionName = "quickSession"

type Request struct {
	writer http.ResponseWriter
	*http.Request
	Param   interface{}
	session Session.ItemInterFace
}

func (r *Request) GetSession() Session.ItemInterFace {
	//防止多处函数引用导致session重复读取
	if r.session != nil {
		return r.session
	}
	context := r.Context()
	value := context.Value(consts.GetSession)
	initFunc := context.Value(consts.InitSessionFunc).(Session.GenerateItemInterFace)
	sessionMap := value.(Session.ServerSession)
	key, exist := sessionMap.GetKeyFromRequest(r.Request)
	if exist {
		item := sessionMap.GetItem(key)
		if item != nil {
			return item
		}
	}
	key, session := sessionMap.GenerateName()(initFunc)
	sessionMap.StoreSession(key, session)
	sessionMap.SetKeyToResponse()(r.writer, key)
	r.session = session
	return session

}

func NewRequest(r *http.Request) *Request {
	return &Request{
		Request: r,
	}
}
