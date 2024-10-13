package async_task_starter

import (
	logger "github.com/kordar/gologger"
	"github.com/kordar/gotask"
	"github.com/spf13/cast"
)

type AsyncTaskStarter struct {
	name string
	load func(moduleName string, itemId string, item map[string]string)
}

func NewAsyncTaskStarter(name string, load func(moduleName string, itemId string, item map[string]string)) *AsyncTaskStarter {
	return &AsyncTaskStarter{name, load}
}

func (m AsyncTaskStarter) Name() string {
	return m.name
}

func (m AsyncTaskStarter) Load(value interface{}) {
	cfg := cast.ToStringMapString(value)
	id := cfg["id"]
	if id == "" {
		logger.Fatalf("[%s] the attribute id cannot be empty.", m.Name())
		return
	}

	workpoolsize := 3
	if cfg["work_size"] != "" {
		workpoolsize = cast.ToInt(cfg["work_size"])
	}

	workpoolbuflen := 200
	if cfg["work_buff_len"] != "" {
		workpoolbuflen = cast.ToInt(cfg["work_buff_len"])
	}

	gotask.InitTaskHandle(workpoolsize, workpoolbuflen)

	if m.load != nil {
		m.load(m.name, id, cfg)
		logger.Debugf("[%s] triggering custom loader completion", m.Name())
	}

	logger.Infof("[%s] loading module '%s' successfully", m.Name(), id)
}

func (m AsyncTaskStarter) Close() {
}
