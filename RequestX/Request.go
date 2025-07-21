package RequestX

import (
	"github.com/wangshiben/QuicFrameWork/Session"
	"net/http"
)

type Request interface {
	GetSession() (Session.ItemInterFace, error)
	GetParam() interface{}
	GetRequest() *http.Request
}
