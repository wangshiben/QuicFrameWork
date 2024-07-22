package RouteDisPatch

import (
	"fmt"
	"testing"
)

type test struct {
	Name string `quick:"server"`
	Next *test
}

func TestMainFunc(t *testing.T) {
	asInterface := reflectBackToStructAsInterface(&test{Name: "test", Next: &test{Name: "aaac"}}, nil, "", "")
	t2, ok := asInterface.(*test)
	fmt.Println(t2.Name, ok)
}
