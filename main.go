package main

import (
	"cube/manager"
	"cube/task"
	"cube/worker"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

func main() {

	fmt.Println("Starting Cube worker(s)")
	host := os.Getenv("CUBE_HOST")
	port, _ := strconv.Atoi(os.Getenv("CUBE_PORT"))
	w := worker.Worker{
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}
	api := worker.Api{
		Address: host,
		Port:    port,
		Worker:  &w,
	}
	workers := []string{fmt.Sprintf("%s:%d", host, port)}

	go runTasks(&w, 10*time.Second)
	go w.CollectStats(15 * time.Second)
	go api.Start()

	fmt.Println("Starting Cube manager")
	m := manager.New(workers)

	fmt.Println("Creating tasks")
	for i := range 3 {
		t := task.Task{
			ID:    uuid.New(),
			Name:  fmt.Sprintf("test-container-%d", i),
			State: task.Scheduled,
			Image: "strm/helloworld-http",
		}
		te := task.TaskEvent{
			ID:    uuid.New(),
			State: task.Running,
			Task:  t,
		}
		m.AddTask(te)
		m.SendWork()
	}

	go func() {
		for {
			log.Printf("[Manager] Updating tasks from %d workers\n", len(m.Workers))
			m.UpdateTasks()
			time.Sleep(15 * time.Second)
		}
	}()

	for {
		for _, t := range m.TaskDb {
			log.Printf("[Manager] Task: id: %s, state: %d\n", t.ID.String(), t.State)
			time.Sleep(10 * time.Second)
		}
	}
}

func runTasks(w *worker.Worker, d time.Duration) {
	for {
		if w.Queue.Len() != 0 {
			result := w.RunTask()
			if result.Error != nil {
				log.Printf("Error running task: %v\n", result.Error)
			}
		} else {
			log.Println("No tasks to proccess currently")
		}
		log.Printf("Sleeping for %v\n", d)
		time.Sleep(d)
	}
}
