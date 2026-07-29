package async_task_starter

import (
	"fmt"
	"log/slog"
	"sync"

	gocfgmodulefx "github.com/kordar/gocfg-load-module/fx/v2"
	"github.com/kordar/gotask"
	"github.com/spf13/cast"
	"go.uber.org/fx"
)

// InstanceConfig 单个异步任务实例配置。
type InstanceConfig struct {
	ID          string
	WorkSize    int
	WorkBuffLen int
}

// ModuleConfig 异步任务模块配置。
type ModuleConfig struct {
	Instances []InstanceConfig
	OnLoaded  func(moduleName string, itemId string, item map[string]string)
}

type cfgModule struct {
	name     string
	index    int
	onLoaded func(moduleName string, itemId string, item map[string]string)
}

var _ gocfgmodulefx.GoCfgModule = cfgModule{}
var _ gocfgmodulefx.GoCfgIndex = cfgModule{}

type Option func(*cfgModule)

// WithIndex 设置加载优先级。
func WithIndex(index int) Option {
	return func(s *cfgModule) {
		s.index = index
	}
}

// WithOnLoaded 设置实例加载完成后的回调（兼容 dig starter 的 load 钩子）。
func WithOnLoaded(fn func(moduleName string, itemId string, item map[string]string)) Option {
	return func(s *cfgModule) {
		s.onLoaded = fn
	}
}

// StarterModule 返回可注册到 gocfg-load-module/fx 的异步任务模块适配器。
// name 对应配置段名称，例如 "async_task" → [async_task] / [async_task.xxx]。
func StarterModule(name string, opts ...Option) gocfgmodulefx.GoCfgModule {
	c := &cfgModule{name: name}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (m cfgModule) Name() string {
	return m.name
}

func (m cfgModule) Index() int {
	return m.index
}

func (m cfgModule) Load(data any) []fx.Option {
	slog.Info("Module Load Complete", "module", "async-task-starter(fx)")
	cfg := buildModuleConfig(data)
	cfg.OnLoaded = m.onLoaded
	return []fx.Option{
		Module(cfg),
	}
}

var (
	handlesMu sync.RWMutex
	handles   = make(map[string]*gotask.TaskHandle)
)

// Module 返回一个 fx.Option，按配置初始化并注册所有异步任务实例。
// 每个实例以 `name:"async-task.<id>"` 的命名标签注册到 fx 容器。
func Module(config ModuleConfig) fx.Option {
	providers := make([]any, 0, len(config.Instances))
	for _, ic := range config.Instances {
		cfg := ic
		providers = append(providers,
			fx.Annotate(
				func() (*gotask.TaskHandle, error) {
					return provideTaskHandle(config, cfg)
				},
				fx.ResultTags(fmt.Sprintf(`name:"async-task.%s"`, cfg.ID)),
			),
		)
	}

	return fx.Module("async-task-starter",
		fx.Supply(config),
		fx.Provide(providers...),
	)
}

// ---------- 配置解析 ----------

func buildModuleConfig(data any) ModuleConfig {
	cfg := ModuleConfig{
		Instances: make([]InstanceConfig, 0),
	}

	section := cast.ToStringMap(data)
	if len(section) == 0 {
		return cfg
	}

	// 单实例模式：[async_task] id = "xxx"
	if section["id"] != nil {
		id := cast.ToString(section["id"])
		if id != "" {
			cfg.Instances = append(cfg.Instances, InstanceConfig{
				ID:          id,
				WorkSize:    cast.ToInt(section["work_size"]),
				WorkBuffLen: cast.ToInt(section["work_buff_len"]),
			})
		}
		return cfg
	}

	// 多实例模式：[async_task.xxx]
	for id, raw := range section {
		v := cast.ToStringMap(raw)
		instID := id
		if explicit := cast.ToString(v["id"]); explicit != "" {
			instID = explicit
		}
		cfg.Instances = append(cfg.Instances, InstanceConfig{
			ID:          instID,
			WorkSize:    cast.ToInt(v["work_size"]),
			WorkBuffLen: cast.ToInt(v["work_buff_len"]),
		})
	}

	return cfg
}

// ---------- 实例初始化 ----------

func provideTaskHandle(moduleCfg ModuleConfig, cfg InstanceConfig) (*gotask.TaskHandle, error) {
	if cfg.ID == "" {
		return nil, fmt.Errorf("async-task-starter: instance id cannot be empty")
	}

	if existing := Get(cfg.ID); existing != nil {
		slog.Info("async-task instance already initialized, reuse", "id", cfg.ID)
		return existing, nil
	}

	cfg = normalizeInstanceConfig(cfg)

	handle := gotask.NewTaskHandleWithName(cfg.ID, cfg.WorkSize, cfg.WorkBuffLen)
	handle.StartWorkerPool()
	Provide(cfg.ID, handle)

	if moduleCfg.OnLoaded != nil {
		item := map[string]string{
			"id":            cfg.ID,
			"work_size":     cast.ToString(cfg.WorkSize),
			"work_buff_len": cast.ToString(cfg.WorkBuffLen),
		}
		moduleCfg.OnLoaded("async-task-starter", cfg.ID, item)
		slog.Debug("triggering custom loader completion", "module", "async-task-starter", "id", cfg.ID)
	}

	slog.Info("async-task instance initialized",
		"id", cfg.ID,
		"work_size", cfg.WorkSize,
		"work_buff_len", cfg.WorkBuffLen,
	)
	return handle, nil
}

func normalizeInstanceConfig(cfg InstanceConfig) InstanceConfig {
	if cfg.WorkSize <= 0 {
		cfg.WorkSize = 3
	}
	if cfg.WorkBuffLen <= 0 {
		cfg.WorkBuffLen = 200
	}
	return cfg
}

// ---------- 全局注册表 ----------

// Provide 将 TaskHandle 注册到本地命名注册表。
func Provide(id string, handle *gotask.TaskHandle) {
	handlesMu.Lock()
	defer handlesMu.Unlock()
	handles[id] = handle
}

// Get 按 id 获取已初始化的 TaskHandle。
func Get(id string) *gotask.TaskHandle {
	handlesMu.RLock()
	defer handlesMu.RUnlock()
	return handles[id]
}

// Has 检查指定 id 的实例是否已初始化。
func Has(id string) bool {
	return Get(id) != nil
}

// Send 向指定实例投递任务。
func Send(id string, body gotask.IBody) {
	if handle := Get(id); handle != nil {
		handle.SendToTaskQueue(body)
	}
}

// AddTasks 向指定实例注册任务处理器。
func AddTasks(id string, tasks ...gotask.ITask) {
	handle := Get(id)
	if handle == nil {
		return
	}
	for i := range tasks {
		handle.AddTask(tasks[i])
	}
}
