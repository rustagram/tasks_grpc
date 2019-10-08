package main

import (
	"context"
	"encoding/json"
	pb "github.com/rustambek96/tasks_grpc/task-service/proto/task"
	"google.golang.org/grpc"
	"io/ioutil"
	"log"
	"os"
)

const (
	address			= "localhost:1234"
	defaultFilename	= "task.json"
)

func parseFile(file string) (*pb.Task, error) {
	var task *pb.Task
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &task)
	if err != nil {
		return nil, err
	}
	return task, err
}

func main() {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Did not connect %v", err)
	}

	defer conn.Close()

	client := pb.NewManagingServiceClient(conn)

	file := defaultFilename
	if len(os.Args) > 1 {
		file = os.Args[1]
	}

	task, err := parseFile(file)
	if err != nil {
		log.Fatalf("Could not parse file: %v", err)
	}
	r, err := client.CreateTask(context.Background(), task)
	if err != nil {
		log.Fatalf("Could not greet: %v", err)
	}
	log.Printf("Created: %t", r.Flag)

	getAll, err := client.GetAllTasks(context.Background(), &pb.GetAllRequest{})
	if err != nil {
		log.Fatalf("Could not list tasks: %v", err)
	}

	_, err = client.MakeLate(context.Background(), &pb.MakeLateRequest{})
	if err != nil {
		log.Fatalf("Could not make late tasks")
	}

	for _, v := range getAll.Tasks {
		log.Println(v)
	}
}