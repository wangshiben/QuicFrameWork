package RouteDisPatch

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// 处理特殊路由标识，如{name:number}
const muchRegexp = `\{(.*)\}`
const numberRegexp = `-?\d+`
const (
	noArgsLen      = 0
	defaultArgsLen = 1
	argsLen        = 2
)

// 默认参数长度，封装至配置文件或配置项
const defaultRouteLen = 1

func getStrRegexpRes(str string) (string, int, error) {
	re := regexp.MustCompile(muchRegexp)
	number := regexp.MustCompile(numberRegexp)
	match := re.FindStringSubmatch(str)

	if len(match) < 1 {
		return "", 0, errors.New("not regex")
	}
	splitStr := strings.Split(match[1], ":")
	length := len(splitStr)
	switch length {
	case noArgsLen:
		return "", 0, errors.New("not regex")
	case defaultArgsLen:
		return splitStr[0], defaultRouteLen, nil
	case argsLen:
		if number.MatchString(splitStr[1]) {
			muchNumber, err := strconv.ParseInt(splitStr[1], 10, 0)
			if err != nil {
				return "", 0, errors.New("invalid Number")
			}
			if muchNumber <= 0 {
				muchNumber = defaultRouteLen
			}
			return splitStr[0], int(muchNumber), nil
		}
	}
	return "", 0, errors.New("not regex")
}
