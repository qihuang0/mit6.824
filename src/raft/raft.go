package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	//	"bytes"
	"sync"
	"sync/atomic"

	//	"6.824/labgob"
	"6.824/labrpc"
)

// Status 节点的角色
type Status int

// VoteState 投票的状态 2A
type VoteState int

// AppendEntriesState 追加日志的状态 2A 2B
type AppendEntriesState int

// HeartBeatTimeout 定义一个全局心跳超时时间
var HeartBeatTimeout = 120 * time.Millisecond

const (
	Follower Status = iota
	Candidate
	Leader
)

const (
	Normal VoteState = iota //投票过程正常
	Killed                  //Raft节点已终止
	Expire                  //投票(消息\竞选者）过期
	Voted                   //本Term内已经投过票

)

const (
	AppNormal    AppendEntriesState = iota // 追加正常
	AppOutOfDate                           // 追加过时
	AppKilled                              // Raft程序终止
	AppRepeat                              // 追加重复 (2B
	AppCommited                            // 追加的日志已经提交 (2B
	Mismatch                               // 追加不匹配 (2B

)

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
type ApplyMsg struct {
	Index        int
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()
	leaderId  int                 // leader id
	isLeader  bool                // is leader
	// Your data here (2A, 2B, 2C).
	commitIndex int
	lastApplied int

	nextIndex  []int
	matchIndex []int

	status      Status
	currentTerm int
	votedFor    int
	log         []Log

	// volatile state on all servers
	volatileState VolatileState

	// volatile state on leaders
	volatileLeaderState VolatileLeaderState

	overtime time.Duration
	timer    *time.Ticker

	// apply channel
	applyCh chan ApplyMsg

	// close goroutine
	killIt chan bool

	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	rf.mu.Lock()
	defer rf.mu.Unlock()
	var term int
	var isleader bool
	// Your code here (2A).
	term = rf.currentTerm
	fmt.Println("the peer[", rf.me, "] state is:", rf.status)
	if rf.status == Leader {
		isleader = true
	} else {
		isleader = false
	}
	return term, isleader
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

// A service wants to switch to snapshot.  Only do so if Raft hasn't
// have more recent info since it communicate the snapshot on applyCh.
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool {

	// Your code here (2D).

	return true
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).

}

// resetElectionTimer reset the election timer
func (rf *Raft) resetElectionTimer() {
	rf.overtime = time.Duration(rand.Intn(150)+300) * time.Millisecond

}

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
	// Your data here (2A, 2B).
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	Term        int
	VoteGranted bool
	VoteState   VoteState
	Ok          bool
	ArgsTerm    int
	// Your data here (2A).
}

// applyStateMachine apply state machine
func (rf *Raft) applyStateMachine() { // has lock already
	for rf.lastApplied < rf.commitIndex {
		rf.lastApplied += 1
		var applyMsg ApplyMsg
		applyMsg.Index = rf.lastApplied
		applyMsg.Command = rf.log[rf.lastApplied-rf.log[0].Index].Command
		rf.applyCh <- applyMsg
	}

}

func lastLog(Logs []Log) Log {
	return Logs[len(Logs)-1]
}

// sendAppendEntries send AppendEntries RPC to server
func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply, appendNums *int) bool {
	if rf.killed() {
		return false
	}

	// paper中5.3节第一段末尾提到，如果append失败应该不断的retries ,直到这个log成功的被store
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	for !ok {

		if rf.killed() {
			return false
		}
		ok = rf.peers[server].Call("Raft.AppendEntries", args, reply)

	}

	// 必须在加在这里否则加载前面retry时进入时，RPC也需要一个锁，但是又获取不到，因为锁已经被加上了
	rf.mu.Lock()
	defer rf.mu.Unlock()

	// 对reply的返回状态进行分支
	switch reply.AppendState {
	// 目标节点crash
	case AppKilled:
		{
			return false
		}

	// 目标节点正常返回
	case AppNormal:
		{
			// 2A的test目的是让Leader能不能连续任期，所以2A只需要对节点初始化然后返回就好
			return true
		}

	//If AppendEntries RPC received from new leader: convert to follower(paper - 5.2)
	//reason: 出现网络分区，该Leader已经OutOfDate(过时）
	case AppOutOfDate:

		// 该节点变成追随者,并重置rf状态
		rf.status = Follower
		rf.votedFor = -1
		rf.timer.Reset(rf.overtime)
		rf.currentTerm = reply.Term

	}
	return ok
}

// updateCommitIndex()
func (rf *Raft) updateCommitIndex() {
	for i := len(rf.log) - 1; i > rf.commitIndex; i-- {
		count := 1
		for j := 0; j < len(rf.peers); j++ {
			if j != rf.me && rf.matchIndex[j] >= i && rf.log[i].Term == rf.currentTerm {
				count++
			}
		}
		if count > len(rf.peers)/2 {
			rf.commitIndex = i
			rf.applyCh <- ApplyMsg{
				CommandValid: true,
				Command:      rf.log[i].Command,
				CommandIndex: i,
			}
			break
		}
	}
}

// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply, voteNums *int) bool {
	if rf.killed() {
		return false
	}
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	for !ok {
		if rf.killed() {
			return false
		}
		ok = rf.peers[server].Call("Raft.RequestVote", args, reply)
	}
	rf.mu.Lock()
	defer rf.mu.Unlock()
	fmt.Printf("[	sendRequestVote(%v) ] : send a election to %v\n", rf.me, server)
	// 由于网络分区，请求投票的人的term的比自己的还小，不给予投票
	if args.Term < rf.currentTerm {
		return false
	}
	// 对reply的返回情况进行分支处理
	switch reply.VoteState {
	// 消息过期有两种情况:
	// 1.是本身的term过期了比节点的还小
	// 2.是节点日志的条目落后于节点了
	case Expire:
		rf.status = Follower
		rf.timer.Reset(rf.overtime)
		if reply.Term > rf.currentTerm {
			rf.currentTerm = reply.Term
			rf.votedFor = -1
		}
	case Normal, Voted:
		//根据是否同意投票，收集选票数量
		if reply.VoteGranted && reply.Term == rf.currentTerm && *voteNums <= (len(rf.peers)/2) {
			*voteNums++
		}
		// 票数超过一半
		if *voteNums >= (len(rf.peers)/2)+1 {
			*voteNums = 0
			// 本身就是leader在网络分区中更具有话语权的leader
			if rf.status == Leader {
				return ok
			}
			// 本身不是leader，那么需要初始化nextIndex数组
			/*
				领导人（服务器）上的易失性状态 (选举后已经重新初始化)
				参数       	    解释
				nextIndex[]	    对于每一台服务器，发送到该服务器的下一个日志条目的索引（初始值为领导人最后的日志条目的索引+1）
				matchIndex[]	对于每一台服务器，已知的已经复制到该服务器的最高日志条目的索引（初始值为0，单调递增）
			*/
			rf.status = Leader
			rf.nextIndex = make([]int, len(rf.peers))
			for i, _ := range rf.nextIndex {
				rf.nextIndex[i] = len(rf.log) + 1
			}

		}
	}
	return ok
}

// example RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).

	defer fmt.Printf("[	    func-RequestVote-rf(%+v)		] : return %v\n", rf.me, reply)

	rf.mu.Lock()
	defer rf.mu.Unlock()

	// 当前节点crash
	if rf.killed() {
		reply.VoteState = Killed
		reply.Term = -1
		reply.VoteGranted = false
		return
	}

	//reason: 出现网络分区，该竞选者已经OutOfDate(过时）
	if args.Term < rf.currentTerm {
		reply.VoteState = Expire
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return
	}

	if args.Term > rf.currentTerm {
		// 重置自身的状态
		rf.status = Follower
		rf.currentTerm = args.Term
		rf.votedFor = -1
	}

	fmt.Printf("[	    func-RequestVote-rf(%+v)		] : rf.voted: %v, request.term: %v, rf.currentTerm: %v\n", rf.me, rf.votedFor, args.Term, rf.currentTerm)
	// 此时比自己任期小的都已经把票还原
	if rf.votedFor == -1 {

		currentLogIndex := len(rf.log) - 1
		currentLogTerm := 0
		// 如果currentLogIndex下标不是-1就把term赋值过来
		if currentLogIndex >= 0 {
			currentLogTerm = rf.log[currentLogIndex].Term
		}

		//  If votedFor is null or candidateId, and candidate’s log is at least as up-to-date as receiver’s log, grant vote (§5.2, §5.4)
		// 论文里的第二个匹配条件，当前peer要符合arg两个参数的预期
		if args.LastLogIndex < currentLogIndex || args.LastLogTerm < currentLogTerm {
			reply.VoteState = Expire
			reply.VoteGranted = false
			reply.Term = rf.currentTerm
			return
		}

		// 给票数，并且返回true
		rf.votedFor = args.CandidateId

		reply.VoteState = Normal
		reply.Term = rf.currentTerm
		reply.VoteGranted = true

		rf.timer.Reset(rf.overtime)

		fmt.Printf("[	    func-RequestVote-rf(%v)		] : voted rf[%v]\n", rf.me, rf.votedFor)

	} else { // 只剩下任期相同，但是票已经给了，此时存在两种情况

		reply.VoteState = Voted
		reply.VoteGranted = false

		// 1、当前的节点是来自同一轮，不同竞选者的，但是票数已经给了(又或者它本身自己就是竞选者）
		if rf.votedFor != args.CandidateId {
			// 告诉reply票已经没了返回false
			return
		} else { // 2. 当前的节点票已经给了同一个人了，但是由于sleep等网络原因，又发送了一次请求
			// 重置自身状态
			rf.status = Follower
		}

		rf.timer.Reset(rf.overtime)

	}

	return
}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.killed() == true {
		return -1, -1, false
	}
	if rf.status != Leader {
		return -1, -1, false
	} else {
		index := rf.getLastIndex() + 1
		term := rf.currentTerm
		return index, term, true
	}
}

// getLastIndex()
func (rf *Raft) getLastIndex() int {
	return rf.log[len(rf.log)-1].Index
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.

	if rf.killed() {
		log.Fatalf("kill client id %d", rf.me)
	}
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)

	return z == 1
}

// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.
func (rf *Raft) ticker() {
	for rf.killed() == false {

		// Your code here to check if a leader election should
		// be started and to randomize sleeping time using

		// 当定时器结束进行超时选举
		select {

		case <-rf.timer.C:
			if rf.killed() {
				return
			}
			rf.mu.Lock()
			// 根据自身的status进行一次ticker
			switch rf.status {

			// follower变成竞选者
			case Follower:
				rf.status = Candidate
				fallthrough
			case Candidate:

				// 初始化自身的任期、并把票投给自己
				rf.currentTerm += 1
				rf.votedFor = rf.me
				votedNums := 1 // 统计自身的票数

				// 每轮选举开始时，重新设置选举超时
				rf.overtime = time.Duration(150+rand.Intn(200)) * time.Millisecond // 随机产生200-400ms
				rf.timer.Reset(rf.overtime)

				// 对自身以外的节点进行选举
				for i := 0; i < len(rf.peers); i++ {
					if i == rf.me {
						continue
					}

					voteArgs := RequestVoteArgs{
						Term:         rf.currentTerm,
						CandidateId:  rf.me,
						LastLogIndex: len(rf.log),
						LastLogTerm:  0,
					}
					if len(rf.log) > 0 {
						voteArgs.LastLogTerm = rf.log[len(rf.log)-1].Term
					}

					voteReply := RequestVoteReply{}

					go rf.sendRequestVote(i, &voteArgs, &voteReply, &votedNums)
				}
			case Leader:
				// 进行心跳/日志同步
				appendNums := 1 // 对于正确返回的节点数量
				rf.timer.Reset(rf.overtime)
				// 构造msg
				for i := 0; i < len(rf.peers); i++ {
					if i == rf.me {
						continue
					}
					args := AppendEntriesArgs{
						Term:         rf.currentTerm,
						LeaderId:     rf.me,
						PrevLogIndex: 0,
						PrevLogTerm:  0,
						Entries:      nil,
						LeaderCommit: rf.commitIndex, // commitIndex为大多数log所认可的commitIndex
					}

					reply := AppendEntriesReply{}
					//
					//// 如果nextIndex[i]长度不等于rf.logs,代表与leader的log entries不一致，需要附带过去
					//
					//args.Entries = rf.log[rf.nextIndex[i]-1:]
					//
					//// 代表已经不是初始值0
					//if rf.nextIndex[i] > 0 {
					//	args.PrevLogIndex = rf.nextIndex[i] - 1
					//}
					//
					//if args.PrevLogIndex > 0 {
					//	//fmt.Println("len(rf.log):", len(rf.logs), "PrevLogIndex):", args.PrevLogIndex, "rf.nextIndex[i]", rf.nextIndex[i])
					//	args.PrevLogTerm = rf.logs[args.PrevLogIndex-1].Term
					//}

					fmt.Printf("[	ticker(%v) ] : send a election to %v\n", rf.me, i)
					go rf.sendAppendEntries(i, &args, &reply, &appendNums)
				}
			}

			rf.mu.Unlock()
		}

	}
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me
	rf.applyCh = applyCh
	rf.votedFor = -1
	rf.commitIndex = 0
	rf.lastApplied = 0
	rf.nextIndex = make([]int, len(peers))
	rf.matchIndex = make([]int, len(peers))
	rf.overtime = time.Duration(150+rand.Intn(200)) * time.Millisecond // 随机产生150-350ms
	rf.timer = time.NewTicker(rf.overtime)
	// Your initialization code here (2A, 2B, 2C).

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()

	return rf
}

type Log struct {
	Term    int
	Command interface{}
	Index   int
}

type AppendEntriesArgs struct {
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []Log
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term        int
	Success     bool
	AppendState AppendEntriesState
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	// 节点crash
	if rf.killed() {
		reply.AppendState = AppKilled
		reply.Term = -1
		reply.Success = false
		return
	}

	// 出现网络分区，args的任期，比当前raft的任期还小，说明args之前所在的分区已经OutOfDate
	if args.Term < rf.currentTerm {
		reply.AppendState = AppOutOfDate
		reply.Term = rf.currentTerm
		reply.Success = false
		return
	}

	// 对当前的rf进行ticker重置
	rf.currentTerm = args.Term
	rf.votedFor = args.LeaderId
	rf.status = Follower
	rf.timer.Reset(rf.overtime)

	// 对返回的reply进行赋值
	reply.AppendState = AppNormal
	reply.Term = rf.currentTerm
	reply.Success = true
	return

}

// volatile state on all servers
type VolatileState struct {
	commitIndex int
	lastApplied int
}

// volatile state on leaders
type VolatileLeaderState struct {
	nextIndex  []int
	matchIndex []int
}
