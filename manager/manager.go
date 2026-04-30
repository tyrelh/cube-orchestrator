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
	fmt.Println("I will update tasks")
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
