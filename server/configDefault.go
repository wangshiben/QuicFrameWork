package server

import "github.com/wangshiben/QuicFrameWork/consts"

const DefaultMaxSessionMemoryBytes int64 = consts.MB * 50

var defaultConfig = &Config{maxMemo: DefaultMaxSessionMemoryBytes}
