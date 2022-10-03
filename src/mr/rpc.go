package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import "os"
import "strconv"

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

// JobCondition condition of job
type JobCondition int

// JobType type of job
type JobType int

const (
	JobWorking = iota
	JobWaiting
	JobDone
)

type Args struct{}

type Reply struct {
	TaskType TaskType

	TaskNum int

	ReduceTasks int

	MapFile string

	MapTasks int
}

type TaskType int

const (
	Map    TaskType = 1
	Reduce TaskType = 2
	Done   TaskType = 3
)

//condition of coordinator
type Condition int

const (
	MapPhase = iota
	ReducePhase
	AllDone
)

// Add your RPC definitions here.

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/824-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
