# pool
English | [中文](README_ZH.md)

A goroutine pool in golang.

## Feature
- goroutine count limit
- goroutine reuse
- set goroutine idle timeout
- all goroutine shutdown gracefully

## Install
### go get:
```shell
$ go get -u github.com/sulybot/pool
```

### go module:(with GO111MODULE=on)
```shell
$ ls
main.go
$ go mod init main
go: creating new go.mod: module main
$ go mod tidy
go: finding module for package github.com/sulybot/pool
go: found github.com/sulybot/pool in github.com/sulybot/pool v0.1.0
$ go run main.go
```

## Usage
### run anonymous func in pool:
```go
package main
import (
	"fmt"
	"github.com/sulybot/pool"
)

func main() {
	// create a routine pool, max routine count 10, task cache queue 100
	p := pool.New(10, 100)
	// set routine idle timeout in second, default 10, set to 0 to disable timeout
	p.setIdleTimeout(60)

	for i := 0, i < 100; i++ {
		n := i

		// call Start to start a goroutine and execute the anonymous func
		p.Start(func() {
			fmt.Println(n)
		})
	}

	// call Shutdown to stop the pool and wait for all the goroutines shutdown
	<-p.Shutdown()
}
```

### implements of pool.Runnable:
```go
package main

import (
	"fmt"
	"github.com/sulybot/pool"
)

// define a struct
type Task struct {
	id int
}

// implement Run method for the struct
func (r *Task) Run() {
	fmt.Println(r.id)
}

func main() {
	p := pool.New(10, 100)
	p.SetIdleTimeout(60)

	for i := 0; i < 100; i++ {
		// declare a pool.Runnable variable
		var r pool.Runnable
		r = &Task{
			id: i,
		}

		// call start and pass the pool.Runnable
		p.Start(r)
	}

	<-p.Shutdown()
}
```
