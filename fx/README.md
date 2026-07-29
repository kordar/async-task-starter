# async-task-starter/fx

基于 `go.uber.org/fx` 的异步任务 starter，实现 `gocfg-load-module/fx` 的 `GoCfgModule` 接口，按配置创建多个命名 `gotask.TaskHandle` 实例并注入到 Fx 容器。

## 快速接入

### 1. 注册 starter

```go
import (
    gocfgmodulefx "github.com/kordar/gocfg-load-module/fx/v2"
    asynctaskstarter "github.com/kordar/async-task-starter/fx/v2"
)

gocfgmodulefx.Register(asynctaskstarter.StarterModule("async_task"))
```

### 2. 注入 TaskHandle

```go
type MyService struct {
    fx.In
    SysTask *gotask.TaskHandle `name:"async-task.sys"`
}
```

命名规则：`name:"async-task.<id>"`

### 3. 全局 API

```go
handle := asynctaskstarter.Get("sys")
asynctaskstarter.AddTasks("sys", MyTask{})
asynctaskstarter.Send("sys", MyBody{})
```

---

## 配置方式

### 单实例

```ini
[async_task]
id = "default"
work_size = 5
work_buff_len = 30
```

### 多实例

```ini
[async_task.sys]
work_size = 10
work_buff_len = 50

[async_task.file]
work_size = 2
work_buff_len = 100
```

---

## 配置项说明

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `id` | string | — | 实例标识（单实例模式必填；多实例模式默认取段名） |
| `work_size` | int | `3` | Worker 数量 |
| `work_buff_len` | int | `200` | 每个 Worker 的任务队列缓冲长度 |

---

## 注入与使用

```go
fx.Invoke(fx.Annotate(
    func(handle *gotask.TaskHandle) {
        handle.AddTask(MyTask{})
        handle.SendToTaskQueue(MyBody{})
    },
    fx.ParamTags(`name:"async-task.sys"`),
))
```

也可在 `StarterModule` 上挂载加载回调（兼容 dig starter 的 `load` 钩子）：

```go
asynctaskstarter.StarterModule("async_task",
    asynctaskstarter.WithOnLoaded(func(moduleName, itemId string, item map[string]string) {
        // 在此注册任务等
    }),
    asynctaskstarter.WithIndex(20),
)
```
