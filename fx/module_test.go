package async_task_starter

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kordar/gotask"
	"go.uber.org/fx"
)

func TestBuildModuleConfigEmpty(t *testing.T) {
	cfg := buildModuleConfig(nil)
	if len(cfg.Instances) != 0 {
		t.Fatalf("expected 0 instances, got %d", len(cfg.Instances))
	}

	cfg = buildModuleConfig(map[string]any{})
	if len(cfg.Instances) != 0 {
		t.Fatalf("expected 0 instances, got %d", len(cfg.Instances))
	}
}

func TestBuildModuleConfigSingle(t *testing.T) {
	cfg := buildModuleConfig(map[string]any{
		"id":            "default",
		"work_size":     "5",
		"work_buff_len": "30",
	})

	if len(cfg.Instances) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(cfg.Instances))
	}

	inst := cfg.Instances[0]
	if inst.ID != "default" {
		t.Fatalf("expected id 'default', got %q", inst.ID)
	}
	if inst.WorkSize != 5 {
		t.Fatalf("expected 5 workers, got %d", inst.WorkSize)
	}
	if inst.WorkBuffLen != 30 {
		t.Fatalf("expected 30 queue, got %d", inst.WorkBuffLen)
	}
}

func TestBuildModuleConfigMulti(t *testing.T) {
	cfg := buildModuleConfig(map[string]any{
		"sys": map[string]any{
			"work_size":     "10",
			"work_buff_len": "50",
		},
		"file": map[string]any{
			"work_size": "2",
		},
	})

	if len(cfg.Instances) != 2 {
		t.Fatalf("expected 2 instances, got %d", len(cfg.Instances))
	}

	byID := make(map[string]InstanceConfig, len(cfg.Instances))
	for _, inst := range cfg.Instances {
		byID[inst.ID] = inst
	}

	sys, ok := byID["sys"]
	if !ok {
		t.Fatal("expected instance 'sys' not found")
	}
	if sys.WorkSize != 10 {
		t.Fatalf("expected 10 workers, got %d", sys.WorkSize)
	}
	if sys.WorkBuffLen != 50 {
		t.Fatalf("expected 50 queue, got %d", sys.WorkBuffLen)
	}

	file, ok := byID["file"]
	if !ok {
		t.Fatal("expected instance 'file' not found")
	}
	if file.WorkSize != 2 {
		t.Fatalf("expected 2 workers, got %d", file.WorkSize)
	}
}

func TestNormalizeInstanceConfigDefaults(t *testing.T) {
	cfg := normalizeInstanceConfig(InstanceConfig{ID: "test"})
	if cfg.WorkSize != 3 {
		t.Fatalf("expected default 3 workers, got %d", cfg.WorkSize)
	}
	if cfg.WorkBuffLen != 200 {
		t.Fatalf("expected default 200 queue, got %d", cfg.WorkBuffLen)
	}
}

func TestNormalizeInstanceConfigPreservesValues(t *testing.T) {
	cfg := normalizeInstanceConfig(InstanceConfig{
		ID:          "test",
		WorkSize:    7,
		WorkBuffLen: 50,
	})
	if cfg.WorkSize != 7 {
		t.Fatalf("expected 7 workers, got %d", cfg.WorkSize)
	}
	if cfg.WorkBuffLen != 50 {
		t.Fatalf("expected 50 queue, got %d", cfg.WorkBuffLen)
	}
}

type demoBody struct {
	n int
}

func (d demoBody) TaskId() string { return "demo" }

type demoTask struct {
	count *atomic.Int64
}

func (d demoTask) Id() string { return "demo" }

func (d demoTask) Execute(body gotask.IBody) {
	d.count.Add(1)
}

func TestModuleProvidesNamedTaskHandle(t *testing.T) {
	handlesMu.Lock()
	handles = make(map[string]*gotask.TaskHandle)
	handlesMu.Unlock()

	var got *gotask.TaskHandle
	var count atomic.Int64

	app := fx.New(
		Module(ModuleConfig{
			Instances: []InstanceConfig{
				{ID: "sys", WorkSize: 2, WorkBuffLen: 10},
			},
		}),
		fx.Invoke(fx.Annotate(
			func(handle *gotask.TaskHandle) {
				got = handle
				handle.AddTask(demoTask{count: &count})
			},
			fx.ParamTags(`name:"async-task.sys"`),
		)),
	)

	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if got == nil {
		t.Fatal("expected task handle to be injected")
	}
	if !Has("sys") {
		t.Fatal("expected registry to contain 'sys'")
	}
	if Get("sys") != got {
		t.Fatal("registry handle should match injected handle")
	}

	Send("sys", demoBody{n: 1})
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if count.Load() >= 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected task to execute, count=%d", count.Load())
}

func TestStarterModuleLoad(t *testing.T) {
	mod := StarterModule("async_task", WithIndex(10))
	if mod.Name() != "async_task" {
		t.Fatalf("unexpected name: %s", mod.Name())
	}
	if idx, ok := mod.(gocfgIndex); !ok || idx.Index() != 10 {
		t.Fatalf("expected index 10")
	}

	opts := mod.Load(map[string]any{
		"id":        "default",
		"work_size": "4",
	})
	if len(opts) != 1 {
		t.Fatalf("expected 1 fx option, got %d", len(opts))
	}
}

type gocfgIndex interface {
	Index() int
}
