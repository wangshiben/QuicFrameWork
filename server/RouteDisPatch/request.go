package RouteDisPatch

import (
	"github.com/google/uuid"
	"github.com/wangshiben/QuicFrameWork/server/Session"
	"github.com/wangshiben/QuicFrameWork/server/consts"
	"net/http"
	"time"
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
	value := context.Value(consts.GetSession)
	initFunc := context.Value(consts.InitSessionFunc).(Session.GenerateItemInterFace)
	sessionMap := value.(Session.ServerSession)
	if cookie != nil {
		item := sessionMap.GetItem(cookie.Value)
		if item != nil {
			return item
		}
	}
	name, session := generateName(initFunc)
	sessionMap.StoreSession(name, session)
	cookieIn := &http.Cookie{
		Name:     quicSessionName,
		Value:    name,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   0,
		Domain:   "localhost",
		Expires:  time.Now().AddDate(0, 0, 30),
	}
	//r.writer.Header()["Set-Cookie"] = []string{cookieIn.String()}

	r.writer.Header().Set("Set-Cookie", cookieIn.String())
	r.session = session
	//cookieVal := fmt.Sprintf("%s=%s;HttpOnly;SameSite=Lax", quicSessionName, name)
	//r.writer.Header().Set("Set-Cookie", cookieVal)
	return session
	//return initFunc()

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
