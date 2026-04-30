package manager

import (
	"bytes"
	"cube/task"
	"cube/worker"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

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

func (m *Manager) SelectWorker() string {
	m.LastWorker = (m.LastWorker + 1) % len(m.Workers)
	return m.Workers[m.LastWorker]
}

func (m *Manager) UpdateTasks() {
	for _, worker := range m.Workers {
		log.Printf("Checking worker %s for task updates", worker)
		url := fmt.Sprintf("http://%s/tasks", worker)
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("Error connecting to %s: %v\n", worker, err)
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("Error sending request: %d\n", resp.StatusCode)
		}

		d := json.NewDecoder(resp.Body)
		var tasks []*task.Task
		err = d.Decode(&tasks)
		if err != nil {
			log.Printf("Error unmarshalling tasks: %v\n", err)
		}

		for _, t := range tasks {
			log.Printf("Attempting to update task %v\n", t.ID)
			_, found := m.TaskDb[t.ID.String()]
			if !found {
				log.Printf("Task with ID %s not found\n", t.ID.String())
				return
			}

			m.TaskDb[t.ID.String()].State = t.State
			m.TaskDb[t.ID.String()].StartTime = t.StartTime
			m.TaskDb[t.ID.String()].FinishTime = t.FinishTime
			m.TaskDb[t.ID.String()].ContainerID = t.ContainerID

		}
	}
}

func (m *Manager) SendWork() {
	if m.Pending.Len() > 0 {
		w := m.SelectWorker()
		e := m.Pending.Dequeue().(task.TaskEvent)
		t := e.Task
		log.Printf("Pulled %v off pending queue\n", t)

		m.EventDb[e.ID.String()] = &e
		m.WorkerTaskMap[w] = append(m.WorkerTaskMap[w], e.Task.ID)
		m.TaskWorkerMap[t.ID] = w

		t.State = task.Scheduled
		m.TaskDb[t.ID.String()] = &t

		data, err := json.Marshal(e)
		if err != nil {
			log.Printf("Unable to marshal task object: %v\n", t)
		}

		url := fmt.Sprintf("http://%s/tasks", w)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			log.Printf("Error connecting to %s: %v\n", w, err)
			m.Pending.Enqueue(e)
			return
		}
		d := json.NewDecoder(resp.Body)
		if resp.StatusCode != http.StatusCreated {
			errorResponse := worker.ErrResponse{}
			err := d.Decode(&errorResponse)
			if err != nil {
				log.Printf("Error decoding response: %s\n", err.Error)
				return
			}
			log.Printf("Response error (%d): %v\n", errorResponse.HTTPStatusCode, errorResponse.Message)
			return
		}

		t = task.Task{}
		err = d.Decode(&t)
		if err != nil {
			log.Printf("Error decoding response: %s\n", err.Error)
			return
		}
		log.Printf("%#v\n", t)
	} else {
		log.Println("No work in the queue")
	}
}
