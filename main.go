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
	wHost := os.Getenv("CUBE_WORKER_HOST")
	if wHost == "" {
		wHost = "localhost"
	}
	wPortStr := os.Getenv("CUBE_WORKER_PORT")
	if wPortStr == "" {
		wPortStr = "5555"
	}
	wPort, _ := strconv.Atoi(wPortStr)

	mHost := os.Getenv("CUBE_MANAGER_HOST")
	if mHost == "" {
		mHost = "localhost"
	}
	mPortStr := os.Getenv("CUBE_MANAGER_PORT")
	if mPortStr == "" {
		mPortStr = "5556"
	}
	mPort, _ := strconv.Atoi(mPortStr)

	log.Println("Starting Cube worker(s)")
	w := worker.Worker{
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}
	wApi := worker.Api{
		Address: wHost,
		Port:    wPort,
		Worker:  &w,
	}
	go w.RunTasks(15 * time.Second)
	go w.CollectStats(14 * time.Second)
	go w.UpdateTasks(13 * time.Second)
	go wApi.Start()

	log.Println("Starting Cube manager")
	workers := []string{fmt.Sprintf("%s:%d", wHost, wPort)}
	m := manager.New(workers)
	mApi := manager.Api{
		Address: mHost,
		Port:    mPort,
		Manager: m,
	}
	go m.ProcessTasks(13 * time.Second)
	go m.UpdateTasks(12 * time.Second)
	go m.DoHealthChecks(11 * time.Second)
	mApi.Start()
}
