package Session

import "reflect"

// MemoryUsageOf returns a reporter's dynamic usage or a concurrency-safe,
// shallow estimate when the implementation can't safely expose its internals.
func MemoryUsageOf(value any) int64 {
	if value == nil {
		return 0
	}
	valueOf := reflect.ValueOf(value)
	typeOf := valueOf.Type()
	if valueOf.Kind() == reflect.Ptr && valueOf.IsNil() {
		return int64(typeOf.Size())
	}
	if reporter, ok := value.(MemoryUsageReporter); ok {
		if usage := reporter.MemoryUsage(); usage >= 0 {
			return usage
		}
	}
	switch valueOf.Kind() {
	case reflect.String:
		return int64(typeOf.Size()) + int64(valueOf.Len())
	case reflect.Slice:
		return int64(typeOf.Size()) + int64(valueOf.Cap())*int64(typeOf.Elem().Size())
	case reflect.Array:
		return int64(typeOf.Size())
	default:
		return int64(typeOf.Size())
	}
}
