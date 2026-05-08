package worker

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"

	"cube/task"
)

type Worker struct {
	Name      string
	Queue     queue.Queue
	Db        map[uuid.UUID]*task.Task
	TaskCount int
	Stats     *Stats
}

func (w *Worker) CollectStats(d time.Duration) {
	for {
		log.Println("[Worker] Collecting stats")
		w.Stats = GetStats()
		w.Stats.TaskCount = w.TaskCount
		time.Sleep(d)
	}
}

func (w *Worker) GetTasks() []*task.Task {
	tasks := []*task.Task{}
	for _, t := range w.Db {
		tasks = append(tasks, t)
	}
	return tasks
}

func (w *Worker) AddTask(t task.Task) {
	w.Queue.Enqueue(t)
}

func (w *Worker) runTask() task.DockerResult {
	t := w.Queue.Dequeue()
	if t == nil {
		log.Println("[Worker] No tasks in the queue")
		return task.DockerResult{Error: nil}
	}
	taskQueued := t.(task.Task)
	taskPersisted := w.Db[taskQueued.ID]
	if taskPersisted == nil {
		taskPersisted = &taskQueued
		w.Db[taskQueued.ID] = &taskQueued
	}

	var result task.DockerResult
	if task.ValidStateTransition(taskPersisted.State, taskQueued.State) {
		switch taskQueued.State {
		case task.Scheduled:
			result = w.StartTask(taskQueued)
		case task.Completed:
			result = w.StopTask(taskQueued)
		default:
			result.Error = errors.New("[Worker] We should not get here")
		}
	} else {
		result.Error = fmt.Errorf("[Worker] Invalid transition from %v to %v", taskPersisted.State, taskQueued.State)
	}
	return result
}

func (w *Worker) StartTask(t task.Task) task.DockerResult {
	t.StartTime = time.Now().UTC()
	config := task.NewConfig(&t)
	docker := task.NewDocker(config)
	result := docker.Run()
	if result.Error != nil {
		log.Printf("[Worker] Error running task %v: %v\n", t.ID, result.Error)
		t.State = task.Failed
		w.Db[t.ID] = &t
		return result
	}
	t.ContainerID = result.ContainerId
	t.State = task.Running
	w.Db[t.ID] = &t
	return result
}

func (w *Worker) StopTask(t task.Task) task.DockerResult {
	config := task.NewConfig(&t)
	docker := task.NewDocker(config)

	result := docker.Stop(t.ContainerID)
	if result.Error != nil {
		log.Printf("[Worker] Error stopping container %s: %v\n", t.ContainerID, result.Error)
		// book note: added missing return
		return task.DockerResult{Error: result.Error}
	}
	t.FinishTime = time.Now().UTC()
	t.State = task.Completed
	w.Db[t.ID] = &t
	log.Printf("[Worker] Stopped and removed container %s for task %v\n", t.ContainerID, t.ID)
	return result
}

func (w *Worker) RunTasks(d time.Duration) {
	for {
		if w.Queue.Len() != 0 {
			result := w.runTask()
			if result.Error != nil {
				log.Printf("[Worker] Error running task: %v\n", result.Error)
			}
		} else {
			log.Println("[Worker] No tasks to proccess currently")
		}
		log.Printf("[Worker] Sleeping for %v\n", d)
		time.Sleep(d)
	}
}

func (w *Worker) InspectTask(t task.Task) task.DockerInspectResponse {
	config := task.NewConfig(&t)
	d := task.NewDocker(config)
	return d.Inspect(t.ContainerID)
}

func (w *Worker) UpdateTasks(duration time.Duration) {
	for {
		log.Println("[Worker] Checking status of tasks")
		w.updateTasks()
		log.Println("[Worker] Task updates completed")
		log.Printf("[Worker] Sleeping for %v seconds\n", duration)
		time.Sleep(duration)
	}
}

func (w *Worker) updateTasks() {
	for id, t := range w.Db {
		if t.State == task.Running {
			resp := w.InspectTask(*t)
			if resp.Error != nil {
				log.Printf("[Worker] ERROR: %v\n", resp.Error)
			}

			if resp.Container == nil {
				log.Printf("[Worker] No container for running task %s\n", id)
				w.Db[id].State = task.Failed
			}

			if resp.Container.State.Status == "exited" {
				log.Printf("Container for task %s is in non-running state %s", id, resp.Container.State.Status)
				w.Db[id].State = task.Failed
			}

			// Error: cannot use resp.Container.NetworkSettings.Ports (variable of map type network.PortMap) as network.PortSet value in assignment
			// w.Db[id].ExposedPorts = resp.Container.NetworkSettings.Ports
		}
	}
}
