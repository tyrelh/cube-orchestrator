package worker

import (
	"cube/task"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (a *Api) StartTaskHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	taskEvent := task.TaskEvent{}
	err := decoder.Decode(&taskEvent)
	if err != nil {
		msg := fmt.Sprintf("[Worker] Error unmarshalling body: %v", err)
		log.Println(msg)
		w.WriteHeader(400)
		errResponse := ErrResponse{
			HTTPStatusCode: 400,
			Message:        msg,
		}
		json.NewEncoder(w).Encode(errResponse)
		return
	}

	a.Worker.AddTask(taskEvent.Task)
	log.Printf("[Worker] Added task %v\n", taskEvent.Task.ID)
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(taskEvent.Task)
}

func (a *Api) GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(a.Worker.GetTasks())
}

func (a *Api) StopTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskIDParam := chi.URLParam(r, "taskID")
	if taskIDParam == "" {
		log.Println("[Worker] No taskID passed in request")
		w.WriteHeader(400)
	}

	taskID, _ := uuid.Parse(taskIDParam)
	_, found := a.Worker.Db[taskID]
	if !found {
		log.Printf("[Worker] No task with ID %v found", taskID)
		w.WriteHeader(404)
	}

	taskToStop := a.Worker.Db[taskID]
	taskCopy := *taskToStop
	taskCopy.State = task.Completed
	a.Worker.AddTask(taskCopy)

	log.Printf("[Worker] Added task %v to stop container %v\n", taskToStop.ID, taskToStop.ContainerID)
	w.WriteHeader(204)
}

func (a *Api) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(a.Worker.Stats)
}
