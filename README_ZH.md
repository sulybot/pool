# pool
[English](README.md) | 中文

一个 Go 语言中的 goroutine 协程池。

## 功能
- 限制 goroutine 总数
- 复用 goroutine
- 设置 goroutine 空闲时间
- 安全、平滑结束所有 goroutine

## 安装
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

## 用法
### 运行匿名函数
```go
package main
import (
	"fmt"
	"github.com/sulybot/pool"
)

func main() {
	// 创建一个协程池, 最大 10 个协程, task 队列最大缓存 100 个元素
	p := pool.New(10, 100)
	// 设置协程池中 goroutine 的空闲时长,单位为秒,默认 10 秒,设置为 0 永不超时
	p.setIdleTimeout(60)

	for i := 0, i < 100; i++ {
		n := i

		// 调用 Start 方法开启一个协程并执行匿名函数
		p.Start(func() {
			fmt.Println(n)
		})
	}

	// 关闭协程池并等待所有协程安全退出
	<-p.Shutdown()
}
```

### 实现 pool.Runnable 接口
```go
package main

import (
	"fmt"
	"github.com/sulybot/pool"
)

// 定义一个结构体
type Task struct {
	id int
}

// 为结构体实现 Run 方法
func (r *Task) Run() {
	fmt.Println(r.id)
}

func main() {
	p := pool.New(10, 100)
	p.SetIdleTimeout(60)

	for i := 0; i < 100; i++ {
        // 声明一个 pool.Runnable 接口变量
		var r pool.Runnable
		r = &Task{
			id: i,
		}

		// 调用 Start 方法,传入 Runnable
		p.Start(r)
	}

	<-p.Shutdown()
}
```
