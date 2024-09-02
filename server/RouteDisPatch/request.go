package RouteDisPatch

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/wangshiben/QuicFrameWork/server"
	"github.com/wangshiben/QuicFrameWork/server/Session"
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
	if r.session != nil {
		return r.session
	}
	cookie, err := r.Cookie(quicSessionName)
	if err != nil && err != http.ErrNoCookie {
		return nil
	}
	context := r.Context()
	value := context.Value(server.GetSession)
	initFunc := context.Value(server.InitSessionFunc).(Session.GenerateItemInterFace)
	if value != nil {
		sessionMap := value.(Session.ServerSession)
		item := sessionMap.GetItem(cookie.Name)
		if item != nil {
			return item
		}
		name, session := generateName(initFunc)
		sessionMap.StoreSession(name, session)
		cookieVal := fmt.Sprintf("%s=%s", quicSessionName, name)
		r.writer.Header().Add("Set-Cookie", cookieVal)
		return session
	}
	return initFunc()

}

func NewRequest(r *http.Request) *Request {
	return &Request{
		Request: r,
	}
}

func generateName(initFunc Session.GenerateItemInterFace) (string, Session.ItemInterFace) {
	uid := uuid.New()
	return uid.String(), initFunc()
}
