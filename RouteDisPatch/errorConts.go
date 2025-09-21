package RouteDisPatch

type paramError string

func (p paramError) Error() string {
	return string(p)
}

const (
	ErrorParamType    = "paramPointer must be a pointer"
	ErrorInReflectTag = "You have wrong in a request Param reflect pointer"
)
