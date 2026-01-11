# safego

`safego` 是一个简单的 Go 工具包，旨在通过自动捕获 panic 来提供更安全的 goroutine 运行方式。

## 功能特性

- **安全协程**：启动 goroutine 并自动恢复（recover）任何 panic。
- **函数包装**：包装返回 `error` 的函数，确保即使发生 panic 也能安全处理。
- **日志打印**：使用`github.com/kakkk/gopkglogger`打印错误日志和堆栈信息。

## 安装

```bash
go get github.com/kakkk/gopkg/safego@latest
```

## 使用示例

### `Go`

使用 `safego.Go` 启动一个带有 panic 恢复机制的协程。

```go
import (
    "context"
    "github.com/kakkk/gopkg/safego"
)

func main() {
    ctx := context.Background()
    
    safego.Go(ctx, func() {
        // 这里的 panic 会被捕获并记录日志，不会导致程序崩溃
        panic("发生了一些错误")
    })
    
    // 等待协程执行
    time.Sleep(time.Second)
}
```

### `GoFn`

使用 `safego.GoFn` 包装一个返回 `error` 的函数。包装后的函数在调用时如果发生 panic，会被自动恢复。

```go
import (
    "context"
    "github.com/kakkk/gopkg/safego"
)

func main() {
    ctx := context.Background()
    
    fn := safego.GoFn(ctx, func() error {
        // 这里的 panic 会被捕获并记录日志
        panic("哎呀")
        return nil
    })
    
    err := fn() // err 不再为 nil，而是包含 "panic recovered: 哎呀" 的错误信息
}
```

## License

MIT