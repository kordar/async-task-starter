package async_task_starter

import "github.com/kordar/gotask"

func GetTask() *gotask.TaskHandle {
	return gotask.GetAsyncTaskHandle()
}

func SendBody(body gotask.IBody) {
	gotask.SendAsyncTaskData(body)
}

func AddTask(tasks ...gotask.ITask) {
	for i := range tasks {
		gotask.RegAsyncTask(tasks[i])
	}
}
