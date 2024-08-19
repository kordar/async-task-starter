package async_task

import (
	"github.com/kordar/gotask"
	"github.com/spf13/cast"
)

type AsyncTaskModule struct {
}

func (m AsyncTaskModule) Name() string {
	return "async_task"
}

func (m AsyncTaskModule) Load(value interface{}) {
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

func (m AsyncTaskModule) Close() {
}
