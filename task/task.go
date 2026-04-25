package task

import (
	"context"
	"io"
	"log"
	"math"
	"os"
	"time"

	"github.com/google/uuid"

	// book note: needed to modify docker imports from github.com/docker/docker to github.com/moby/moby
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

type State int

const (
	Pending State = iota
	Scheduled
	Running
	Completed
	Failed
)

type Task struct {
	ID     uuid.UUID
	Name   string
	State  State
	Image  string
	Memory int
	Disk   int
	// book note: needed to modify nat.PortSet to network.PortSet
	ExposedPorts  network.PortSet
	PortBindings  map[string]string
	RestartPolicy string
	StartTime     time.Time
	FinishTime    time.Time
}

type TaskEvent struct {
	ID        uuid.UUID
	State     State
	Timestamp time.Time
	Task      Task
}

type Config struct {
	Name         string
	AttachStdin  bool
	AttachStdout bool
	AttachStderr bool
	// book note: needed to modify nat.PortSet to network.PortSet
	ExposedPorts network.PortSet
	Cmd          []string
	Image        string
	Cpu          float64
	Memory       int64
	Disk         int64
	Env          []string
	// book note: needed to modify RestartPolicy from a string to container.RestartPolicyMode
	RestartPolicy container.RestartPolicyMode
}

type Docker struct {
	Client *client.Client
	Config Config
}

type DockerResult struct {
	Error       error
	Action      string
	ContainerId string
	Result      string
}

func (d *Docker) Run() DockerResult {
	ctx := context.Background()
	// book note: needed to modify types.ImagePullOptions to client.ImagePullOptions
	reader, err := d.Client.ImagePull(ctx, d.Config.Image, client.ImagePullOptions{})
	if err != nil {
		log.Printf("Error pulling image %s: %v\n", d.Config.Image, err)
		return DockerResult{Error: err}
	}
	io.Copy(os.Stdout, reader)

	restartPolicy := container.RestartPolicy{
		Name: d.Config.RestartPolicy,
	}
	resources := container.Resources{
		Memory:   d.Config.Memory,
		NanoCPUs: int64(d.Config.Cpu * math.Pow(10, 9)),
	}
	containerConfig := container.Config{
		Image:        d.Config.Image,
		Tty:          false,
		Env:          d.Config.Env,
		ExposedPorts: d.Config.ExposedPorts,
	}
	hostConfig := container.HostConfig{
		RestartPolicy:   restartPolicy,
		Resources:       resources,
		PublishAllPorts: true, // automatically map internal ports to host ports
	}

	log.Printf("Creating container %s for %s", d.Config.Name, d.Config.Image)
	resp, err := d.Client.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config:           &containerConfig,
		HostConfig:       &hostConfig,
		NetworkingConfig: nil,
		Platform:         nil,
		Name:             d.Config.Name,
		// book note: removed Image: d.Config.Image since it's already set in the container.Config
	})
	if err != nil {
		log.Printf("Error creating container using image %s: %v\n", d.Config.Image, err)
		return DockerResult{Error: err}
	}
	log.Printf("Created container %s", resp.ID)

	log.Printf("Starting container %s", resp.ID)
	// book note: needed to ignore client.ContainerStartResult return. Maybe act on it?
	_, err = d.Client.ContainerStart(ctx, resp.ID, client.ContainerStartOptions{})
	if err != nil {
		log.Printf("Error starting container %s: %v\n", resp.ID, err)
		return DockerResult{Error: err}
	}
	log.Printf("Started container %s", resp.ID)

	// book note: commented this out for now as d.Config.Runtime doesn't exist
	// d.Config.Runtime.ContainerID = resp.ID

	out, err := d.Client.ContainerLogs(
		ctx,
		resp.ID,
		client.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
		},
	)
	if err != nil {
		log.Printf("Error getting logs for container %s: %v\n", resp.ID, err)
		return DockerResult{Error: err}
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	return DockerResult{
		ContainerId: resp.ID,
		Action:      "start",
		Result:      "success",
	}
}

func (d *Docker) Stop(id string) DockerResult {
	log.Printf("Attempting to stop container %s\n", id)
	ctx := context.Background()
	// book note: needed to ignore client.ContainerStopResult return. Maybe act on it?
	_, err := d.Client.ContainerStop(
		ctx,
		id,
		client.ContainerStopOptions{},
	)
	if err != nil {
		log.Printf("Error stopping container %s: %v\n", id, err)
		return DockerResult{Error: err}
	}

	// book note: needed to ignore client.ContainerRemoveResult return. Maybe act on it?
	_, err = d.Client.ContainerRemove(
		ctx,
		id,
		client.ContainerRemoveOptions{
			RemoveVolumes: true,
			RemoveLinks:   false,
			Force:         false,
		},
	)
	if err != nil {
		log.Printf("Error removing container %s: %v\n", id, err)
		return DockerResult{Error: err}
	}

	return DockerResult{
		Action: "stop",
		Result: "success",
		Error:  nil,
	}
}
