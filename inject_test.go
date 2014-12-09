package xweb

import (
	"fmt"
	"testing"
)

type Tester struct {
}

type TestInterceptor struct {
	tester *Tester
}

func (t *TestInterceptor) SetTester(tester *Tester) {
	t.tester = tester
}

func TestInjector(t *testing.T) {
	c := NewInjector()
	c.Map(&Tester{})
	var itor TestInterceptor
	c.Inject(&itor)

	fmt.Println("itor.tester:", itor.tester)
	if itor.tester == nil {
		t.Error("itor.tester is nil")
	}
}
