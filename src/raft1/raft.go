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
	"raft/src/utils"
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
	mu        sync.Mutex          // Lock to protect shared access to this peer's ServerState
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *tester.Persister   // Object to hold this peer's persisted ServerState
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()
	//persist to disk
	EventLogs   []LogEntry
	VotedFor    int
	CurrentTerm int
	//volatile information
	CommitIndex int
	LastApplied int
	//for leader only.
	NextIndex  []int
	MatchIndex []int
	//ServerState information
	ServerState State
	//election information
	ElectionTimeout time.Duration
	LastHeartBeat   time.Time
	VoteCount       int
	StopHeartBeat   chan bool
	// Look at the paper's Figure 2 for a description of what
	// ServerState a Raft server must maintain.
	//to talk to the application
	ApplicationChanel chan raftapi.ApplyMsg
}

type AppendEntriesArgs struct {
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}
type State int

type LogEntry struct {
	Term    int
	Command interface{}
}

// RequestVoteArgs structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	Term         int //current Term on candidate server.
	CandidateId  int //id of the candidate server.
	LastLogIndex int //last index the server has filled up in its log.
	LastLogTerm  int //Term of the item at the last log index.
}

// RequestVoteReply  structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (3A).
	Term        int  //CurrentTerm, for candidate to update itself
	VoteGranted bool //if vote was given or not to the current candidate.
}

// return CurrentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	utils.RecoverWithStackTrace("GetState", rf.me)
	var term int
	var isleader bool
	rf.mu.Lock()
	defer rf.mu.Unlock()
	// Your code here (3A).
	term = rf.CurrentTerm
	if rf.ServerState == Leader {
		isleader = true
	}
	return term, isleader
}

// save Raft's persistent ServerState to stable storage,
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

// restore previously persisted ServerState.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any ServerState?
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

func (rf *Raft) SendEventLogs(eventTerm int, eventCommand interface{}) {
	term := eventTerm
	command := eventCommand
	rf.mu.Lock()
	isLeader := rf.ServerState == Leader
	prevlogindex := len(rf.EventLogs) - 1
	prevlogterm := 0
	if prevlogindex >= 0 {
		prevlogterm = rf.EventLogs[prevlogindex].Term
	}
	request := AppendEntriesArgs{
		Term:         term,
		LeaderId:     rf.me,
		PrevLogIndex: prevlogindex,
		PrevLogTerm:  prevlogterm,
		Entries:      []LogEntry{{Term: term, Command: command}},
		LeaderCommit: rf.CommitIndex,
	}
	rf.mu.Unlock()
	if isLeader {
		//send out append entry RPC
		rf.mu.Lock()
		for i := range rf.peers {
			if i == rf.me {
				continue
			}
			go func(server int, localRequest AppendEntriesArgs) {
				localReply := AppendEntriesReply{}
				ok := rf.peers[server].Call("AppendEntries", &localRequest, &localReply)
				if !ok {
					log.Printf("Append entries failed in server %v", server)
				} else {
					if localReply.Term > rf.CurrentTerm {
						rf.CurrentTerm = localReply.Term
						rf.ServerState = Follower
						rf.VotedFor = -1
						rf.VoteCount = 0
					} else if !localReply.Success {
						//gotta do log correction now for the server that has mismatched logs.
					}
				}
			}(i, request)
		}
		rf.mu.Unlock()
	}
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
// Term. the third return value is true if this server believes it is
// the leader.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	index := len(rf.EventLogs)
	term := rf.CurrentTerm
	isLeader := rf.ServerState == Leader
	if isLeader {
		go rf.SendEventLogs(term, command)
		if rf.CommitConsensus > len(rf.peers)/2 {
			rf.EventLogs = append(rf.EventLogs, LogEntry{Term: term, Command: command})
		}
	}
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

//func (rf *Raft) HandleAppendEntries(reply *AppendEntriesReply) {
//	rf.mu.Lock()
//	term := rf.CurrentTerm
//	isLeader := rf.ServerState == Leader
//	if reply.Term > term || !isLeader {
//		rf.CurrentTerm = reply.Term
//		rf.VotedFor = -1
//		rf.ServerState = Follower
//		rf.VoteCount = 0
//	} else {
//		//if the success is set to true, its fine. just updates logs and stuff.
//		//if success is false, got to fid the previous matching Term in the log of this server and the
//		//leader.
//		return
//	}
//
//}

// AppendEntries , this is on the server that is on the receiving end of the RPC.
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	//log.Printf("In appendentries")
	rf.mu.Lock()
	defer utils.RecoverWithStackTrace("AppendEntries", rf.me)
	defer rf.mu.Unlock()
	reply.Success = false
	//log.Printf("AppendEntries for server in func AppendEntries %d", rf.me)
	//if the heartbeat server has a lower term than this server, step down.
	if args.Term < rf.CurrentTerm {
		reply.Term = rf.CurrentTerm
		return
	} else if args.Term > rf.CurrentTerm {
		rf.CurrentTerm = args.Term
		rf.VotedFor = -1
		rf.ServerState = Follower
	}
	if args.Entries == nil {
		//heartbeat
		rf.LastHeartBeat = time.Now()
		rf.ServerState = Follower
		reply.Term = rf.CurrentTerm
		reply.Success = true
	} else {
		//compare logs here.
		if rf.EventLogs[args.PrevLogIndex].Term != args.PrevLogTerm {
			//logs are not matching at prev log index, have to keep sending older logs till we get replicated logs.
			reply.Success = false
			return
		} else {
			//loop through the entries field and append all to logs.
			for i := range args.Entries {
				rf.EventLogs = append(rf.EventLogs, args.Entries[i])
			}
			reply.Success = true
			rf.LastHeartBeat = time.Now()
			rf.ServerState = Follower
			reply.Term = rf.CurrentTerm
		}
	}
}

func (rf *Raft) SendHeartBeatToPeers(server int, term int, leaderId int) {
	rf.mu.Lock()
	defer utils.RecoverWithStackTrace("SendHeartBeatToPeers", rf.me)
	if rf.ServerState != Leader || rf.CurrentTerm != term { //in case it's been modified by some other node.
		rf.mu.Unlock()
		return
	}
	request := &AppendEntriesArgs{
		Term:         term,
		LeaderId:     leaderId,
		PrevLogIndex: 0,
		PrevLogTerm:  0,
		Entries:      nil,
		LeaderCommit: 0,
	}
	rf.mu.Unlock()
	reply := &AppendEntriesReply{}
	//log.Printf("Sending AppendEntries to %v", server)
	//log.Printf("peers: %v", rf.peers[server])
	ok := rf.peers[server].Call("Raft.AppendEntries", request, reply)
	if !ok {
		log.Printf("AppendEntries failed for server %d", server)
	} else {
		//log.Printf("AppendEntries for server %d", server)
	}
}

func (rf *Raft) SendHeartbeatImmediate() {
	rf.mu.Lock()
	defer utils.RecoverWithStackTrace("SendHeartbeatImmediate", rf.me)
	if rf.ServerState != Leader {
		rf.mu.Unlock()
		return
	}
	term := rf.CurrentTerm
	leaderId := rf.me
	rf.mu.Unlock()
	for i, _ := range rf.peers {
		if i == leaderId {
			continue
		}
		//log.Printf("Sending heartbeat to %v", i)
		go rf.SendHeartBeatToPeers(i, term, leaderId) //send concurrently to increase speed.
	}

}

func (rf *Raft) PeriodicHeartbeats() {
	defer utils.RecoverWithStackTrace("PeriodicHeartbeats", rf.me)
	heartbeatInterval := 100 * time.Millisecond
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	//log.Printf("Heartbeat periodic every %v", heartbeatInterval)
	//when just became a leader send it right away.
	rf.SendHeartbeatImmediate()
	for {
		select {
		// if not the leader. cant send out shit.
		case <-ticker.C:
			rf.mu.Lock()
			if rf.ServerState != Leader {
				log.Printf("%v is no longer a leader and cant send out heartbeats", rf.me)
				rf.mu.Unlock()
				return
			}
			rf.mu.Unlock()
			//log.Printf("Sending out heartbeats now")
			rf.SendHeartbeatImmediate()
		}
	}
}

func (rf *Raft) HandleVoteReplies(reply *RequestVoteReply) {
	rf.mu.Lock()
	defer utils.RecoverWithStackTrace("HandleVoteReplies", rf.me)
	defer rf.mu.Unlock()

	if reply.Term > rf.CurrentTerm {
		rf.CurrentTerm = reply.Term
		rf.VotedFor = -1
		rf.VoteCount = 0
		rf.ServerState = Follower
		return
	}

	if reply.VoteGranted {
		//vote counting.
		rf.VoteCount++
		if rf.VoteCount > len(rf.peers)/2 {
			//become leader and send out rpc to all the other peers.
			rf.ServerState = Leader
			for i, _ := range rf.peers {
				//log.Printf("len of eventLogs: %v", len(rf.EventLogs))
				rf.NextIndex[i] = len(rf.EventLogs)
				rf.MatchIndex[i] = 0
			}
			log.Printf("Server %d became leader for term %d", rf.me, rf.CurrentTerm)
			go rf.PeriodicHeartbeats()
		}
	}
}

// RequestVote example RequestVote RPC handler.This is implemented on the servers receiving the RPC
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	//log.Printf("In requesting vote")
	rf.mu.Lock()
	defer utils.RecoverWithStackTrace("RequestVote", rf.me)
	defer rf.mu.Unlock()
	reply.VoteGranted = false
	if args.Term < rf.CurrentTerm {
		reply.Term = rf.CurrentTerm
		log.Printf("Term of candidate %d cannot be less than current Term %d\n", args.Term, rf.CurrentTerm)
		return
		//return nil
	} else if {
		//check for log conditions for getting a vote here.
	}
	if args.Term > rf.CurrentTerm {
		//log.Printf("Server %d updating term from %d to %d, becoming follower", rf.me, rf.CurrentTerm, args.Term)
		rf.CurrentTerm = args.Term
		rf.VotedFor = -1
		rf.ServerState = Follower
		rf.VoteCount = 0
	}
	reply.Term = rf.CurrentTerm
	//grant vote only if;
	//1. Candidate's Term >= CurrentTerm
	//2. didnt vote for anyone yet or voted for this candidate
	//3. candidates log entries are atleast as up to date as ours.
	//by that I mean, If the logs have last entries with different terms, then
	//the log with the later Term is more up-to-date. If the logs
	//end with the same Term, then whichever log is longer is
	//more up-to-date
	if rf.VotedFor == -1 || rf.VotedFor == args.CandidateId {
		reply.VoteGranted = true
		rf.VotedFor = args.CandidateId
		reply.Term = rf.CurrentTerm
		//make sure to add log checks here.
	}
	//log.Printf("Vote granted to %v", args.CandidateId)
	//return nil
}

func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	defer utils.RecoverWithStackTrace("sendRequestVote", rf.me)
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) StartElection() {
	//vote for self, increment the Term and send requestvote rpc
	//log.Printf("Start Election")
	rf.mu.Lock()
	defer utils.RecoverWithStackTrace("StartElection", rf.me)
	rf.CurrentTerm++
	term := rf.CurrentTerm
	rf.ServerState = Candidate
	rf.VotedFor = rf.me
	rf.VoteCount = 1
	lastLogIndex := len(rf.EventLogs) - 1
	lastLogTerm := 0
	if lastLogIndex >= 0 {
		lastLogTerm = rf.EventLogs[lastLogIndex].Term
	}
	rf.ElectionTimeout = time.Duration(rand.Intn(MaxTime-MinTime+1)+MinTime) * time.Millisecond
	rf.mu.Unlock()
	for i := range rf.peers {
		if i == rf.me {
			continue
		}
		go func(server int) {
			request := &RequestVoteArgs{term, rf.me, lastLogIndex, lastLogTerm}
			reply := &RequestVoteReply{}
			ok := rf.sendRequestVote(server, request, reply)
			if !ok {
				//log.Println("RequestVote Failed")
			} else {
				rf.HandleVoteReplies(reply)
			}
		}(i)
	}
}

func (rf *Raft) ticker() {
	//code necessary for 3A
	for rf.killed() == false {
		rf.mu.Lock()
		defer utils.RecoverWithStackTrace("ticker", rf.me)
		timeElapsed := time.Since(rf.LastHeartBeat)
		isLeader := rf.ServerState == Leader
		rf.mu.Unlock()
		if !isLeader && timeElapsed > rf.ElectionTimeout {
			rf.StartElection()
			//started a leader election process.
		}
		// pause for a random amount of time between 50 and 350
		// milliseconds.
		ms := 50 + (rand.Int63() % 300)
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
}

//func (rf *Raft) SendHeartbeats() {
//
//	if rf.ServerState == Leader {
//		//send heartbeat rpc to all peers.
//		for i, _ := range rf.peers {
//			if i == rf.me {
//				continue
//			}
//			server := i
//			request := &AppendEntriesArgs{
//				Term:         rf.CurrentTerm,
//				LeaderId:     rf.me,
//				PrevLogIndex: 0,
//				PrevLogTerm:  0,
//				Entries:      nil,
//				LeaderCommit: 0,
//			}
//			reply := &AppendEntriesReply{}
//			ok := rf.peers[server].Call("AppendEntries", request, reply)
//			if !ok {
//				log.Println("Heartbeat Failed")
//			} else {
//				if reply.Term > rf.CurrentTerm {
//					rf.CurrentTerm = reply.Term
//					rf.ServerState = Follower
//					rf.VotedFor = -1
//					rf.VoteCount = 0
//				}
//			}
//		}
//	} else {
//		log.Printf("Server %v not a Leader %v", rf.me, rf.ServerState)
//		return
//	}
//}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent ServerState, and also initially holds the most
// recent saved ServerState, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int,
	persister *tester.Persister, applyCh chan raftapi.ApplyMsg) *Raft {
	// Your initialization code here (3A, 3B, 3C).
	defer utils.RecoverWithStackTrace("Make", me)
	rf := &Raft{}
	//log.Printf("Starting a server")
	rf.peers = peers
	rf.ApplicationChanel = applyCh
	rf.persister = persister
	rf.me = me
	rf.EventLogs = make([]LogEntry, 0)
	rf.VotedFor = -1
	rf.CurrentTerm = 0
	rf.ServerState = Follower
	rf.ElectionTimeout = time.Duration(rand.Intn(MaxTime-MinTime+1)+MinTime) * time.Millisecond
	//log.Printf("Timeout is: %v", rf.ElectionTimeout)
	rf.CommitIndex = 0
	rf.LastApplied = 0
	//log.Printf("length of peers: %v", len(rf.peers))
	rf.NextIndex = make([]int, len(rf.peers))
	rf.MatchIndex = make([]int, len(rf.peers))
	log.Printf("len of nextIndex: %v", len(rf.NextIndex))
	rf.LastHeartBeat = time.Now()
	rf.StopHeartBeat = make(chan bool, 1)
	// initialize from ServerState persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	// start ticker goroutine to start elections
	go rf.ticker()
	//go rf.SendHeartbeats()
	//log.Printf("Methods on *Raft: %+v", reflect.TypeOf(rf))
	//for i := 0; i < reflect.TypeOf(rf).NumMethod(); i++ {
	//	log.Printf("Method: %s", reflect.TypeOf(rf).Method(i).Name)
	//}
	return rf
}
