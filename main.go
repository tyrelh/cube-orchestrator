package main

import (
	"cube/manager"
	"cube/node"
	"cube/task"
	"cube/worker"
	"fmt"
	"os"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/moby/moby/client"
)

func main() {
	t := task.Task{
		ID:     uuid.New(),
		Name:   "Task-1",
		State:  task.Pending,
		Image:  "Image-1",
		Memory: 1024,
		Disk:   1,
	}

	te := task.TaskEvent{
		ID:        uuid.New(),
		State:     task.Pending,
		Timestamp: time.Now(),
		Task:      t,
	}

	fmt.Printf("task: %v\n", t)
	fmt.Printf("task event: %v\n", te)

	w := worker.Worker{
		Name:  "worker-1",
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}
	fmt.Printf("worker: %v\n", w)
	w.CollectStats()
	w.RunTask()
	w.StartTask()
	w.StopTask()

	m := manager.Manager{
		Pending: *queue.New(),
		TaskDb:  make(map[string][]*task.Task),
		EventDb: make(map[string][]*task.TaskEvent),
		Workers: []string{w.Name},
	}

	fmt.Printf("manager: %v\n", m)
	m.SelectWorker()
	m.UpdateTasks()
	m.SendWork()

	n := node.Node{
		Name:   "Node-1",
		Ip:     "192.168.1.1",
		Cores:  4,
		Memory: 1024,
		Disk:   25,
		Role:   "worker",
	}
	fmt.Printf("node: %v\n", n)

	// testing starting and stopping a container

	fmt.Print("create a test container\n")
	dockerTask, createResult := createContainer()
	if createResult.Error != nil {
		fmt.Printf("%v", createResult.Error)
		os.Exit(1)
	}
	time.Sleep(time.Second * 5)
	fmt.Printf("stopping container %s\n", createResult.ContainerId)
	_ = stopContainer(dockerTask, createResult.ContainerId)
}

func createContainer() (*task.Docker, *task.DockerResult) {
	config := task.Config{
		Name:  "test-container-1",
		Image: "postgres:18",
		Env: []string{
			"POSTGRES_USER=cube",
			"POSTGRES_PASSWORD=secret",
		},
	}

	// book note: needed to add client.WithAPIVersionNegotiation()
	dockerClient, _ := client.New(client.FromEnv, client.WithAPIVersionNegotiation())
	dockerTask := task.Docker{
		Client: dockerClient,
		Config: config,
	}

	result := dockerTask.Run()
	if result.Error != nil {
		fmt.Printf("%v\n", result.Error)
		return nil, &result
	}
	return &dockerTask, &result
}

func stopContainer(dockerTask *task.Docker, id string) *task.DockerResult {
	result := dockerTask.Stop(id)
	if result.Error != nil {
		fmt.Printf("%v\n", result.Error)
		return nil
	}
	fmt.Printf("Container %s has been stopped and removed\n", result.ContainerId)
	return &result
}
