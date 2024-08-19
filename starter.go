package async_task_starter

import (
	"github.com/kordar/gotask"
	"github.com/spf13/cast"
)

type AsyncTaskStarter struct {
}

func (m AsyncTaskStarter) Name() string {
	return "async_task_starter"
}

func (m AsyncTaskStarter) Load(value interface{}) {
	cfg := cast.ToStringMap(value)
	workpoolsize := 3
	if cfg["work_size"] != "" {
		workpoolsize = cast.ToInt(cfg["work_size"])
	}

	workpoolbuflen := 200
	if cfg["work_buff_len"] != "" {
		workpoolbuflen = cast.ToInt(cfg["work_buff_len"])
	}

	gotask.InitTaskHandle(workpoolsize, workpoolbuflen)
}

func (m AsyncTaskStarter) Close() {
}
