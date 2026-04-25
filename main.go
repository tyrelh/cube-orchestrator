package main

import (
	"cube/task"
	"cube/worker"
	"fmt"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

func main() {
	db := make(map[uuid.UUID]*task.Task)
	w := worker.Worker{
		Queue: *queue.New(),
		Db:    db,
	}

	t := task.Task{
		ID:    uuid.New(),
		Name:  "test-container-1",
		State: task.Scheduled,
		Image: "strm/helloworld-http",
	}

	// first time the worker will see the task
	fmt.Printf("starting task %s", t.ID)
	w.AddTask(t)
	result := w.RunTask()
	if result.Error != nil {
		panic(result.Error)
	}

	t.ContainerID = result.ContainerId
	fmt.Printf("task %s is running in container %s\n", t.ID, t.ContainerID)
	fmt.Println("Sleep time")
	time.Sleep(time.Second * 30)

	fmt.Printf("stopping task %s\n", t.ID)
	t.State = task.Completed
	w.AddTask(t)
	result = w.RunTask()
	if result.Error != nil {
		panic(result.Error)
	}
}

// func createContainer() (*task.Docker, *task.DockerResult) {
// 	config := task.Config{
// 		Name:  "test-container-1",
// 		Image: "postgres:18",
// 		Env: []string{
// 			"POSTGRES_USER=cube",
// 			"POSTGRES_PASSWORD=secret",
// 		},
// 	}

// 	// book note: needed to add client.WithAPIVersionNegotiation()
// 	dockerClient, _ := client.New(client.FromEnv, client.WithAPIVersionNegotiation())
// 	dockerTask := task.Docker{
// 		Client: dockerClient,
// 		Config: config,
// 	}

// 	result := dockerTask.Run()
// 	if result.Error != nil {
// 		fmt.Printf("%v\n", result.Error)
// 		return nil, &result
// 	}
// 	return &dockerTask, &result
// }

// func stopContainer(dockerTask *task.Docker, id string) *task.DockerResult {
// 	result := dockerTask.Stop(id)
// 	if result.Error != nil {
// 		fmt.Printf("%v\n", result.Error)
// 		return nil
// 	}
// 	fmt.Printf("Container %s has been stopped and removed\n", result.ContainerId)
// 	return &result
// }
