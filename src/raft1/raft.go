package raft

// The file raftapi/raft.go defines the interface that raft must
// expose to servers (or the tester), but see comments below for each
// of these functions for more details.
//
// Make() creates a new raft peer that implements the raft interface.

import (
	"log"
	//	"bytes"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	_ "raft/src/labgob"
	"raft/src/labrpc"
	"raft/src/raftapi"
	"raft/src/tester1"
)

const (
	Follower State = iota
	Candidate
	Leader
)

const MinTime int = 600
const MaxTime int = 900

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *tester.Persister   // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()
	//persist to disk
	eventLogs   []logEntry
	votedFor    int
	currentTerm int
	//volatile information
	commitIndex int
	lastApplied int
	//for leader only.
	nextIndex  []int
	matchIndex []int
	//state information
	state State
	//election information
	electionTimeout time.Duration
	LastHeartBeat   time.Time
	VoteCount       int
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
}

type AppendEntriesArgs struct {
}

type AppendEntriesReply struct {
}
type State int

type logEntry struct {
	term    int
	Command interface{}
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here (3A).
	term = rf.currentTerm
	if rf.state == Leader {
		isleader = true
	}
	return term, isleader
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
// before you've implemented snapshots, you should pass nil as the
// second argument to persister.Save().
// after you've implemented snapshots, pass the current snapshot
// (or nil if there's not yet a snapshot).
func (rf *Raft) persist() {
	// Your code here (3C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// raftstate := w.Bytes()
	// rf.persister.Save(raftstate, nil)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (3C).
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

// how many bytes in Raft's persisted log?
func (rf *Raft) PersistBytes() int {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return rf.persister.RaftStateSize()
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (3D).

}

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (3A, 3B).
	Term         int //current term on candidate server.
	CandidateId  int //id of the candidate server.
	LastLogIndex int //last index the server has filled up in its log.
	LastLogTerm  int //term of the item at the last log index.
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (3A).
	Term        int  //currentTerm, for candidate to update itself
	VoteGranted bool //if vote was given or not to the current candidate.
}

// example RequestVote RPC handler.
func (rf *Raft) RequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (3A, 3B).
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	if !ok {
		log.Println("RequestVote Failed")
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
// handler function on the server side does not return.Thus, there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
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
	index := -1
	term := -1
	isLeader := true

	// Your code here (3B).

	return index, term, isLeader
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
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func (rf *Raft) StartElection() {
	//vote for self, increment the term and send requestvote rpc
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.state = Candidate
	rf.currentTerm++
	rf.votedFor = rf.me
	rf.VoteCount++
	lastLogIndex := 0
	lastLogTerm := 0
	if len(rf.eventLogs) > 0 {
		lastLogIndex = len(rf.eventLogs) - 1
		lastLogTerm = rf.eventLogs[len(rf.eventLogs)-1].term
	}
	for i := range rf.peers {
		if i == rf.me {
			continue
		}
		go func(server int) {
			request := &RequestVoteArgs{rf.currentTerm, rf.me, lastLogIndex, lastLogTerm}
			reply := &RequestVoteReply{}
			rf.RequestVote(server, request, reply)
			if reply.VoteGranted {
				log.Printf("Vote Granted from %v", rf.peers[i])
				rf.VoteCount++
				if rf.VoteCount > len(rf.peers)/2 {
					rf.state = Leader
					rf.LastHeartBeat = time.Now()
				}
			} else if reply.Term > rf.currentTerm {
				rf.state = Follower
				rf.VoteCount = 0
				rf.votedFor = -1
				rf.currentTerm = reply.Term
			}
		}(i)
	}
}
func (rf *Raft) ticker() {
	//code necessary for 3A
	for rf.killed() == false {
		rf.mu.Lock()
		timeElapsed := time.Since(rf.LastHeartBeat)
		if rf.state != Leader && timeElapsed > rf.electionTimeout {
			rf.StartElection()
			//started a leader election process.
			rf.electionTimeout = time.Duration(rand.Intn(MaxTime-MinTime+1)+MinTime) * time.Millisecond
			rf.LastHeartBeat = time.Now()
		}
		rf.mu.Unlock()
		// pause for a random amount of time between 50 and 350
		// milliseconds.
		ms := 50 + (rand.Int63() % 300)
		time.Sleep(time.Duration(ms) * time.Millisecond)
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
	persister *tester.Persister, applyCh chan raftapi.ApplyMsg) raftapi.Raft {
	// Your initialization code here (3A, 3B, 3C).
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me
	rf.votedFor = -1
	rf.currentTerm = 0
	rf.state = Follower
	rf.electionTimeout = time.Duration(rand.Intn(MaxTime-MinTime+1)+MinTime) * time.Millisecond
	rf.commitIndex = 0
	rf.lastApplied = 0
	rf.nextIndex = make([]int, 0, len(rf.peers))
	rf.matchIndex = make([]int, 0, len(rf.peers))
	rf.LastHeartBeat = time.Now()
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()

	return rf
}
