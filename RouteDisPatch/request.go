package RouteDisPatch

import (
	"github.com/wangshiben/QuicFrameWork/Session"
	"github.com/wangshiben/QuicFrameWork/consts"
	"github.com/wangshiben/QuicFrameWork/size"
	"net/http"
)

const quicSessionName = "quickSession"

type Request struct {
	writer http.ResponseWriter
	*http.Request
	Param   interface{}
	session Session.ItemInterFace
}

// TODO: 当session大小超出的时候要throw ERROR
func (r *Request) GetSession() (Session.ItemInterFace, error) {
	//防止多处函数引用导致session重复读取
	if r.session != nil {
		return r.session, nil
	}
	context := r.Context()
	value := context.Value(consts.GetSession)
	initFunc := context.Value(consts.InitSessionFunc).(Session.GenerateItemInterFace)
	maxMemo := context.Value(consts.MaxSessionMemo).(int)
	sessionMap := value.(Session.ServerSession)
	key, exist := sessionMap.GetKeyFromRequest(r.Request)
	if exist {
		item := sessionMap.GetItem(key)
		if item != nil {
			return item, nil
		}
	}
	sessionMapSize := size.Of(sessionMap)
	if sessionMapSize >= maxMemo {
		sessionMap.CleanExpItem()
	}
	sessionMapSize = size.Of(sessionMap)
	if sessionMapSize >= maxMemo {
		return nil, Session.MaxMemo
	}
	key, session := sessionMap.GenerateName()(initFunc)
	sessionMap.StoreSession(key, session)
	sessionMap.SetKeyToResponse()(r.writer, key)
	r.session = session
	return session, nil

}

func NewRequest(r *http.Request) *Request {
	return &Request{
		Request: r,
	}
}
