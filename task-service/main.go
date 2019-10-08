package main

import (
	"context"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	pb "github.com/rustambek96/tasks_grpc/task-service/proto/task"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"time"
)

const (
	port = ":1234"
)

type Task struct {
	gorm.Model
	Assignee string `db:"not null"`
	Title    string `db:"not null"`
	Deadline time.Time `db:"not null, default true"`
	Done     bool   `db:"default false"`
	Overdue	 bool	`db:"default false"`
}

type TaskManagerI interface {
	Add(*pb.Task) error
	Update(uint64, *pb.Task) error
	Delete(id uint64) error
	GetAll() ([]*pb.Task, error)
	MakeDone(id uint64) error
	MakeLate() error
}

type TaskManager struct {
	db *gorm.DB
}

func NewTaskManager() (TaskManagerI, error){
	tm := TaskManager{}
	var err error
	tm.db, err = gorm.Open("postgres", "user=postgres password=123 dbname=tasks sslmode=disable")
	if err != nil {
		return nil, err
	}
	tm.db.LogMode(true)
	tm.db.AutoMigrate(Task{})
	return &tm, nil
}
func ProtoToStruct(tsk *pb.Task) Task{
	var task Task
	task.Assignee = tsk.Assignee
	task.Title = tsk.Title
	task.Deadline, _ = time.Parse("2006-01-02", tsk.Deadline)
	task.Done = tsk.Done
	return task
}
func (tm *TaskManager) Add(tsk *pb.Task) error{
	tm.db.LogMode(true)
	task := ProtoToStruct(tsk)
	if result := tm.db.Create(&task); result.Error != nil {
		return result.Error
	}
	return nil
}

func (tm *TaskManager) Update(id uint64, tsk *pb.Task) error {
	task := ProtoToStruct(tsk)
	if result := tm.db.Model(&task).Where("id = ?", id).Updates(&task); result.Error != nil {
		return result.Error
	}
	return nil
}

func (tm *TaskManager) Delete(id uint64) error {
	if result := tm.db.Where("id = ?", id).Delete(&Task{}); result.Error != nil {
		return result.Error
		}
	return nil
}

func (tm *TaskManager) GetAll() ([]*pb.Task, error) {
	var tasks []*pb.Task
	if result := tm.db.Find(&tasks); result.Error != nil {
		return nil, result.Error
	}
	return tasks, nil
}

func (tm *TaskManager) MakeDone(id uint64) error {
	if result := tm.db.Model(&Task{}).Where("id = ?", id).Update("done", true); result.Error != nil {
		return result.Error
	}
	return nil
}

func (tm *TaskManager) MakeLate() error {
	if result := tm.db.Model(&Task{}).Where("done = ? and deadline < ?", false, time.Now()).Update("overdue", true); result.Error != nil {
		return result.Error
	}
	return nil
}

type service struct {
	tmi TaskManagerI
}

func (s *service) CreateTask(ctx context.Context, req *pb.Task) (*pb.FlagResponse, error) {
	err := s.tmi.Add(req)
	if err != nil {
		return nil, err
	}
	return &pb.FlagResponse{Flag:true}, err
}

func (s *service) UpdateTask(ctx context.Context, req *pb.UpdateTaskRequest) (*pb.FlagResponse, error) {
	err := s.tmi.Update(req.Id, req.Task)
	if err != nil {
		return nil, err
	}
	return &pb.FlagResponse{Flag:true}, nil
}

func (s *service) MakeDone(ctx context.Context, req *pb.MakeDoneRequest) (*pb.FlagResponse, error) {
	err := s.tmi.MakeDone(req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.FlagResponse{Flag:true}, nil
}

func (s *service) MakeLate(ctx context.Context, req *pb.MakeLateRequest) (*pb.FlagResponse, error) {
	err := s.tmi.MakeLate()
	if err != nil {
		return &pb.FlagResponse{Flag:false}, err
	}
	return &pb.FlagResponse{Flag:true}, nil
}

func (s *service) DeleteTask(ctx context.Context, req *pb.DeleteTaskRequest) (*pb.FlagResponse, error) {
	err := s.tmi.Delete(req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.FlagResponse{Flag:true}, err
}

func (s *service) GetAllTasks(ctx context.Context, req *pb.GetAllRequest) (*pb.GetAllResponse, error) {
	tasks, err := s.tmi.GetAll()
	if err != nil {
		return nil, err
	}
	return &pb.GetAllResponse{Tasks:tasks}, nil
}

func main() {
	tm, err := NewTaskManager()
	if err != nil {
		log.Fatalf("Cannot connect to database: %v", err)
	}
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()

	pb.RegisterManagingServiceServer(s, &service{tm})

	reflection.Register(s)

	log.Println("Running on port:", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}