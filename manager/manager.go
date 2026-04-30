package manager

import (
	"bytes"
	"cube/task"
	"cube/worker"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

type Manager struct {
	Pending       queue.Queue
	TaskDb        map[string]*task.Task
	EventDb       map[string]*task.TaskEvent
	Workers       []string // format: "hostname:port"
	WorkerTaskMap map[string][]uuid.UUID
	TaskWorkerMap map[uuid.UUID]string
	LastWorker    int
}

func (m *Manager) AddTask(te task.TaskEvent) {
	m.Pending.Enqueue(te)
}

func (m *Manager) SelectWorker() string {
	m.LastWorker = (m.LastWorker + 1) % len(m.Workers)
	return m.Workers[m.LastWorker]
}

func (m *Manager) updateTasks() {
	for _, worker := range m.Workers {
		log.Printf("[Manager] Checking worker %s for task updates", worker)
		url := fmt.Sprintf("http://%s/tasks", worker)
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("[Manager] Error connecting to %s: %v\n", worker, err)
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("[Manager] Error sending request: %d\n", resp.StatusCode)
		}

		d := json.NewDecoder(resp.Body)
		var tasks []*task.Task
		err = d.Decode(&tasks)
		if err != nil {
			log.Printf("[Manager] Error unmarshalling tasks: %v\n", err)
		}

		for _, t := range tasks {
			log.Printf("[Manager] Attempting to update task %v\n", t.ID)
			_, found := m.TaskDb[t.ID.String()]
			if !found {
				log.Printf("[Manager] Task with ID %s not found\n", t.ID.String())
				return
			}

			m.TaskDb[t.ID.String()].State = t.State
			m.TaskDb[t.ID.String()].StartTime = t.StartTime
			m.TaskDb[t.ID.String()].FinishTime = t.FinishTime
			m.TaskDb[t.ID.String()].ContainerID = t.ContainerID

		}
	}
}

func (m *Manager) UpdateTasks(t time.Duration) {
	for {
		log.Println("[Manager] Checking for task updates from workers")
		m.updateTasks()
		log.Println("[Manager] Task updates completed")
		log.Println("[Manager] Sleeping for 15 seconds")
		time.Sleep(t)
	}
}

func (m *Manager) SendWork() {
	if m.Pending.Len() > 0 {
		w := m.SelectWorker()
		e := m.Pending.Dequeue().(task.TaskEvent)
		t := e.Task
		log.Printf("[Manager] Pulled %v off pending queue\n", t)

		m.EventDb[e.ID.String()] = &e
		m.WorkerTaskMap[w] = append(m.WorkerTaskMap[w], e.Task.ID)
		m.TaskWorkerMap[t.ID] = w

		t.State = task.Scheduled
		m.TaskDb[t.ID.String()] = &t

		data, err := json.Marshal(e)
		if err != nil {
			log.Printf("[Manager] Unable to marshal task object: %v\n", t)
		}

		url := fmt.Sprintf("http://%s/tasks", w)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			log.Printf("[Manager] Error connecting to %s: %v\n", w, err)
			m.Pending.Enqueue(e)
			return
		}
		d := json.NewDecoder(resp.Body)
		if resp.StatusCode != http.StatusCreated {
			errorResponse := worker.ErrResponse{}
			err := d.Decode(&errorResponse)
			if err != nil {
				log.Printf("[Manager] Error decoding response: %s\n", err.Error)
				return
			}
			log.Printf("[Manager] Response error (%d): %v\n", errorResponse.HTTPStatusCode, errorResponse.Message)
			return
		}

		t = task.Task{}
		err = d.Decode(&t)
		if err != nil {
			log.Printf("[Manager] Error decoding response: %s\n", err.Error)
			return
		}
		// log.Printf("%#v\n", t)
	} else {
		log.Println("[Manager] No work in the queue")
	}
}

func (m *Manager) ProcessTasks(t time.Duration) {
	for {
		log.Println("[Manager] Processing any tasks in the queue")
		m.SendWork()
		log.Println("[Manager] Sleeping for 10 seconds")
		time.Sleep(t)
	}
}

func (m *Manager) GetTasks() []*task.Task {
	tasks := []*task.Task{}
	for _, t := range m.TaskDb {
		tasks = append(tasks, t)
	}
	return tasks
}

func New(workers []string) *Manager {
	taskDb := make(map[string]*task.Task)
	eventDb := make(map[string]*task.TaskEvent)
	workerTaskMap := make(map[string][]uuid.UUID)
	taskWorkerMap := make(map[uuid.UUID]string)
	for worker := range workers {
		workerTaskMap[workers[worker]] = []uuid.UUID{}
	}
	return &Manager{
		Pending:       *queue.New(),
		Workers:       workers,
		TaskDb:        taskDb,
		EventDb:       eventDb,
		WorkerTaskMap: workerTaskMap,
		TaskWorkerMap: taskWorkerMap,
	}
}
