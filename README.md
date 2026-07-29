# async-task-starter

基于 [`github.com/kordar/gotask`](https://github.com/kordar/gotask) 的异步任务 Worker Pool starter。

## dig / 传统接入

```ini
[async_task]
id = default
# 队列并发数
work_size = 3
# 队列长度
work_buff_len = 200
```

```go
async_task_starter.NewAsyncTaskStarter("async_task", nil)
```

## Fx 接入

见 [`fx/`](./fx/README.md)，模块路径：`github.com/kordar/async-task-starter/fx/v2`。
