package RouteDisPatch

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

type Parse interface {
	ParseFunc(data []byte) interface{}
}

const (
	locationTag  = "quickLoc"     //参数位置在哪
	defaultValue = "quickDefault" //参数默认值
	param        = "quickParam"   //参数对应的param名字，类似于 `json:"name"`
	defaultTag   = "quic"         // new  tag
)
const (
	locate            = "location"
	paramNames        = "name"
	defaultValueInTag = "default"
)

const (
	header   = "header"
	body     = "body"
	reqParam = "param"
)

func reflectBackToStructAsInterface(i interface{}, r *http.Request, defaultLocation, RoutePath string) interface{} {
	requestURI := r.URL.Path //请求路径
	pathMap := getPathMap(RoutePath, requestURI)
	// 获取输入接口的反射值
	val := reflect.ValueOf(i)

	// 检查是否为非空指针且指向一个结构体
	if val.Kind() == reflect.Ptr && !val.IsNil() && val.Elem().Kind() == reflect.Struct {
		// 获取指针指向的结构体的实际类型
		elemType := val.Elem().Type()

		// 创建目标类型的实例
		result := val.Elem()
		//position := ""
		// 遍历结构体的所有字段
		for i := 0; i < elemType.NumField(); i++ {

			// 获取当前字段的值和名称
			fieldVal := val.Elem().Field(i)
			tags := elemType.Field(i).Tag
			tagDefault := tags.Get(defaultTag)
			NameMaps := parseAllTag(tagDefault)
			positionTag := NameMaps[locate] //获取到参数位置
			for j := 0; j < 2 && len(positionTag) != 0; j++ {
				if j == 1 {
					positionTag = defaultLocation
				} else if j == 0 {
					positionTag = tags.Get(locationTag)
				}
			}

			paramName := NameMaps[paramNames]
			for j := 0; j < 2 && len(paramName) != 0; j++ {
				if j == 1 {
					paramName = copyNameToLitter(elemType.Field(i).Name)
				} else if j == 0 {
					paramName = tags.Get(param)
				}
			}

			value := ""
			defaultVal := NameMaps[defaultValueInTag]
			if len(defaultVal) == 0 {
				defaultVal = tags.Get(defaultTag)
			}

			switch positionTag { //获取到需要注入的参数的位置
			case reqParam:
				value = copyFromRequestParam(r, paramName)
			case header:
				value = copyFromHeader(r, paramName)
			default: //Path传参
				vals := pathMap[paramName]
				for index, str := range vals {
					value += str
					if index < len(vals)-1 {
						value += "/"
					}
				}
			}
			if len(value) == 0 {
				value = defaultVal
			}
			if len(value) == 0 {
				continue
			}
			fieldName := elemType.Field(i).Name
			// 在结果实例中找到对应的字段
			structField := result.FieldByName(fieldName)
			// 检查字段是否有效且可设置，然后复制值
			if structField.IsValid() && structField.CanSet() {
				//继续处理value
				switch fieldVal.Kind() {
				case reflect.String:
					structField.SetString(value)
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					intValue, err := strconv.ParseInt(value, 10, 64)
					if err == nil {
						structField.SetInt(intValue)
					} else {
						panic(fmt.Sprintf("Failed to parse integer from header for field '%s': %v", fieldName, err))
					}
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					uintValue, err := strconv.ParseUint(value, 10, 64)
					if err == nil {
						structField.SetUint(uintValue)
					} else {
						panic(fmt.Sprintf("Failed to parse unsigned integer from header for field '%s': %v", fieldName, err))
					}
				case reflect.Bool:
					boolValue, err := strconv.ParseBool(value)
					if err == nil {
						structField.SetBool(boolValue)
					}
				default:
					continue
				}
			}

		}

		// 返回新实例的地址，转换为interface{}
		return result.Addr().Interface()
	}

	// 不满足条件时，返回nil或根据业务逻辑进行错误处理
	return nil
}
func parseAllTag(tag string) map[string]string {
	res := make(map[string]string)
	if len(tag) == 0 {
		return res
	}
	tags := strings.Split(tag, ",")
	for _, data := range tags {
		if strings.TrimSpace(data) != "" {
			KV := strings.Split(data, "=")
			if len(KV) == 1 && len(res[param]) == 0 {
				res[param] = KV[0]
			} else if len(KV) == 2 {
				res[KV[0]] = KV[1]
			} else {
				panic(ErrorInReflectTag)
			}
		}
	}
	return res
}

// OriginPath: 注册进route的原始路径
// RequestPath: 实际请求路径
func getPathMap(OriginPath, RequestPath string) map[string][]string { //匹配算法，要求:前缀必须要一样，否则则没有意义
	//TODO:路径匹配赋值出错首先排查这里
	//map: key-> {name}中的name
	// value -> 对应到requestPath中的Path
	mapPath := make(map[string][]string)
	if len(OriginPath) == 0 || len(RequestPath) == 0 {
		return mapPath
	}
	OriginPath = formatPath(OriginPath)
	RequestPath = formatPath(RequestPath)
	OriginPaths := strings.Split(OriginPath, "/")
	RequestPaths := strings.Split(RequestPath, "/")
	RequestIndex := 0
	for _, path := range OriginPaths {
		name, step, _ := getStrRegexpRes(path)
		if step <= 0 {
			RequestIndex++
			continue
		}
		mapPath[copyNameToLitter(name)] = RequestPaths[RequestIndex : RequestIndex+step]
		RequestIndex += step
	}
	return mapPath
}

func copyFromRequestParam(r *http.Request, ParamName string) string {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return ""
	}
	if query.Has(ParamName) {
		return query.Get(ParamName)
	}
	return ""
}
func copyFromHeader(r *http.Request, ParamName string) string {
	return r.Header.Get(ParamName)
}

func copyNameToLitter(name string) string { //首字母小写
	if name[0] >= 'A' && name[0] <= 'Z' {
		return string(name[0]+32) + name[1:]
	}
	return name
}
