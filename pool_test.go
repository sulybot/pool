package pool

import (
	"fmt"
	"testing"
	"time"
)

func TestPoolSerial(t *testing.T) {
	p := New()
	for i := 0; i < 5; i++ {
		second := i
		p.Start(func() {
			time.Sleep(time.Duration(second) * time.Second)
			fmt.Printf("Serial output: %d\n", second)
		})
	}

	fmt.Println("Serial push finished")
	<-p.Shutdown()
}

func TestPoolParallel(t *testing.T) {
	p := New(5)
	for i := 0; i < 5; i++ {
		second := i
		p.Start(func() {
			time.Sleep(time.Duration(second) * time.Second)
			fmt.Printf("Parallel output: %d\n", second)
		})
	}

	fmt.Println("Parallel push finished")
	<-p.Shutdown()
}

type testRunnable struct {
	second int
}

func (r *testRunnable) Run() {
	time.Sleep(time.Duration(r.second) * time.Second)
	fmt.Printf("Runnable output: %d\n", r.second)
}

func TestPoolRunnable(t *testing.T) {
	p := New(5, 10)
	for i := 0; i < 5; i++ {
		p.Start(&testRunnable{
			second: i,
		})
	}

	<-p.Shutdown()
}
