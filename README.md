# Cube Orchestrator

This repo is me following along to the book [Build an Orchestrator in Go](https://www.manning.com/books/build-an-orchestrator-in-go-from-scratch) by Tim Boring.

It's a simplistic implementation of an orchestrator, like Kubernetes, written in Go using Docker.

## Setup

You'll need Go installed as well as Docker.

```bash
go mod install
```

## Run

To run the service and API:

```bash
CUBE_HOST=localhost CUBE_PORT=5555 go run .
```

## Sample API requests

List tasks
```bash
curl -v localhost:5555/tasks
```

Create task
```bash
curl -v --request POST \
    --header 'Content-Type: application/json' \
    --data '{"ID":"266592cd-960d-4091-981c-8c25c44b1018","State":2,"Task":{"State":1,"ID":"266592cd-960d-4091-981c-8c25c44b1018","Name":"test-chapter-5-1","Image":"strm/helloworld-http"}}' \
    localhost:5555/tasks
```

Delete task
```bash
curl -v --request DELETE "localhost:5555/tasks/266592cd-960d-4091-981c-8c25c44b1018"
```
