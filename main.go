package main

import (
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
