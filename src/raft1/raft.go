package raft

// The file raftapi/raft.go defines the interface that raft must
// expose to servers (or the tester), but see comments below for each
// of these functions for more details.
//
// Make() creates a new raft peer that implements the raft interface.

import (
	"bytes"
	"log"
	"raft/src/labgob"

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
	//to have conditional run of the applier whenever there is a change in the commit index.
	Cond *sync.Cond
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
	Term          int
	Success       bool
	ConflictIndex int
	ConflictTerm  int
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

type DummyArgs struct {
	Me int
}
type DummyReply struct {
	CommitIndex int
}

// return CurrentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	utils.RecoverWithStackTrace("GetState", rf.me)
	var term int
	var isleader bool
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
	//to save to the disk.
	// Your code here (3C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// raftstate := w.Bytes()
	// rf.persister.Save(raftstate, nil)
	writeBuffer := new(bytes.Buffer)
	encoder := labgob.NewEncoder(writeBuffer)
	err := encoder.Encode(rf.CurrentTerm)
	if err != nil {
		log.Printf("error encoding current term: %v", err)
		return
	}
	err = encoder.Encode(rf.VotedFor)
	if err != nil {
		log.Printf("error encoding voted for: %v", err)
		return
	}
	err = encoder.Encode(rf.EventLogs)
	if err != nil {
		log.Printf("error encoding event logs: %v", err)
		return
	}
	raftState := writeBuffer.Bytes()
	rf.persister.Save(raftState, nil)
	//log.Printf("Successfully persisted all 3 state variables")
}

// restore previously persisted ServerState.
func (rf *Raft) readPersist(data []byte) {
	//to read from the disk.
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
	readBuffer := bytes.NewBuffer(data)
	decoder := labgob.NewDecoder(readBuffer)
	var CurrentTerm int
	var VotedFor int
	var EventLogs []LogEntry
	if decoder.Decode(&CurrentTerm) != nil || decoder.Decode(&VotedFor) != nil || decoder.Decode(&EventLogs) != nil {
		log.Printf("readPersist failed while decoding")
	} else {
		rf.mu.Lock()
		rf.CurrentTerm = CurrentTerm
		rf.VotedFor = VotedFor
		rf.EventLogs = EventLogs
		rf.mu.Unlock()
	}
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

func (rf *Raft) ReplicateLogsToFollower(server int, term int) {
	for !rf.killed() {
		rf.mu.Lock()
		if rf.CurrentTerm != term || rf.ServerState != Leader {
			//log.Printf("SendEventLogs failed: term %v, command %v", rf.CurrentTerm, eventCommand)
			rf.VoteCount = 0
			rf.VotedFor = -1
			rf.ServerState = Follower
			rf.persist()
			rf.mu.Unlock()
			return
		}
		isLeader := rf.ServerState == Leader
		me := rf.me
		commitIndex := rf.CommitIndex

		if isLeader {
			nextindex := rf.NextIndex[server] //from where we have to send the logs to this server
			if nextindex < 1 {
				nextindex = 1
			}
			//log.Printf("SendEventLogs peers[%v]: nextindex: %v", server, nextindex)
			prevlogindex := nextindex - 1
			prevlogterm := 0
			if prevlogindex >= 0 && prevlogindex < len(rf.EventLogs) {
				prevlogterm = rf.EventLogs[prevlogindex].Term
			}
			sendEntries := append([]LogEntry(nil), rf.EventLogs[nextindex:]...)
			//log.Printf("SendEventLogs for server %v : %v", me, sendEntries)
			request := AppendEntriesArgs{
				Term:         term,
				LeaderId:     me,
				PrevLogIndex: prevlogindex,
				PrevLogTerm:  prevlogterm,
				Entries:      sendEntries,
				LeaderCommit: commitIndex,
			}
			rf.mu.Unlock()
			var reply AppendEntriesReply
			ok := rf.peers[server].Call("Raft.AppendEntries", &request, &reply)
			if !ok {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			rf.mu.Lock()
			if reply.Term > term || rf.ServerState != Leader {
				log.Printf("No longer in same term or no longer a leader.")
				rf.mu.Unlock()
				return
			}
			if reply.Term > rf.CurrentTerm {
				rf.CurrentTerm = reply.Term
				rf.VotedFor = -1
				rf.VoteCount = 0
				rf.ServerState = Follower
				rf.persist()
				rf.mu.Unlock()
				return
			}
			if reply.Success {
				log.Printf("Successfully appended logs to server %v", server)
				match := request.PrevLogIndex + len(request.Entries)
				if match > rf.MatchIndex[server] {
					rf.MatchIndex[server] = match
				}
				if rf.NextIndex[server] < match+1 {
					rf.NextIndex[server] = match + 1
				}

				for N := rf.CommitIndex + 1; N < len(rf.EventLogs); N++ {
					if rf.EventLogs[N].Term != rf.CurrentTerm {
						continue
					}
					cnt := 1 // include leader
					for i := range rf.peers {
						if i != rf.me && rf.MatchIndex[i] >= N {
							cnt++
						}
					}
					if cnt > len(rf.peers)/2 {
						rf.CommitIndex = N
						rf.Cond.Signal()
					}
				}
				rf.mu.Unlock()
				ms := 50 + (rand.Int63() % 300)
				time.Sleep(time.Duration(ms) * time.Millisecond)
				continue
			} else {
				//backoff logic. skip over all the same terms, to reduce the number of RPC calls.
				newNextIndex := rf.NextIndex[server] - 1
				conflictTerm := reply.ConflictTerm
				conflictIndex := reply.ConflictIndex
				if conflictTerm != -1 {
					lastConflictIndex := -1
					for i := len(rf.EventLogs) - 1; i >= 0; i-- {
						if rf.EventLogs[i].Term == conflictTerm {
							lastConflictIndex = i
							break
						}
					}
					if lastConflictIndex != -1 {
						newNextIndex = lastConflictIndex + 1
					} else if conflictIndex >= 1 {
						newNextIndex = conflictIndex
					}
				} else if conflictIndex != -1 {
					newNextIndex = conflictIndex
				}
				if newNextIndex < 1 {
					newNextIndex = 1
				}

				rf.NextIndex[server] = newNextIndex
				rf.mu.Unlock()
				ms := 50 + (rand.Int63() % 300)
				time.Sleep(time.Duration(ms) * time.Millisecond)
			}
		}
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
	index := -1 //where we plan to insert the command sent by the applicn.
	term := rf.CurrentTerm
	isLeader := rf.ServerState == Leader
	if isLeader {
		index = len(rf.EventLogs)
		rf.EventLogs = append(rf.EventLogs, LogEntry{Term: term, Command: command})
		rf.persist()
		//go rf.SendEventLogs(term, command)
	} else {
		return -1, term, false
	}
	return index, term, isLeader

}

// AppendEntries , this is on the server that is on the receiving end of the RPC.
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer utils.RecoverWithStackTrace("AppendEntries", rf.me)
	defer rf.mu.Unlock()

	reply.Term = rf.CurrentTerm
	reply.Success = false
	reply.ConflictTerm = -1
	reply.ConflictIndex = -1
	// Term checks
	if args.Term < rf.CurrentTerm {
		return
	}
	if args.Term > rf.CurrentTerm {
		rf.CurrentTerm = args.Term
		rf.VotedFor = -1
		rf.ServerState = Follower
		rf.VoteCount = 0
		rf.persist()
	}

	// CONSISTENCY CHECK (applies to heartbeats too)
	// PrevLogIndex must exist and match PrevLogTerm
	if args.PrevLogIndex >= len(rf.EventLogs) {
		// the prevlogindex is still larger than the size of the followers logs.
		reply.ConflictIndex = len(rf.EventLogs)
		return
	}

	if args.PrevLogIndex >= 0 && rf.EventLogs[args.PrevLogIndex].Term != args.PrevLogTerm {
		//term mismatch; find the term that matches and accordingly set conflict index and conflict term
		conflictTerm := rf.EventLogs[args.PrevLogIndex].Term
		reply.ConflictTerm = conflictTerm
		i := args.PrevLogIndex
		for i >= 0 && rf.EventLogs[i].Term == conflictTerm {
			i--
		}
		reply.ConflictIndex = i + 1
		return
	}
	// Append / overwrite any new entries
	insert := args.PrevLogIndex + 1
	if len(args.Entries) > 0 {
		// find first conflict; truncate once then append rest
		for i := 0; i < len(args.Entries); i++ {
			if insert+i < len(rf.EventLogs) {
				if rf.EventLogs[insert+i].Term != args.Entries[i].Term {
					rf.EventLogs = rf.EventLogs[:insert+i]
					rf.EventLogs = append(rf.EventLogs, args.Entries[i:]...)
					rf.persist()
					break
				}
			} else {
				rf.EventLogs = append(rf.EventLogs, args.Entries[i:]...)
				rf.persist()
				break
			}
		}
	}

	// Update follower's commit index, but never beyond lastNewIndex (the last index the follower actually has)
	lastNewIndex := args.PrevLogIndex + len(args.Entries)
	if args.LeaderCommit > rf.CommitIndex {
		if lastNewIndex < args.LeaderCommit {
			rf.CommitIndex = lastNewIndex
		} else {
			rf.CommitIndex = args.LeaderCommit
		}
		rf.Cond.Signal()
	}

	// heartbeat / reset election timer
	rf.LastHeartBeat = time.Now()
	rf.ServerState = Follower
	reply.Success = true
	reply.Term = rf.CurrentTerm
}

func (rf *Raft) SendHeartBeatToPeers(server int, term int, leaderId int) {
	rf.mu.Lock()
	defer utils.RecoverWithStackTrace("SendHeartBeatToPeers", rf.me)
	if rf.ServerState != Leader || rf.CurrentTerm != term { //in case it's been modified by some other node.
		rf.mu.Unlock()
		return
	}
	prevlogindex := len(rf.EventLogs) - 1
	prevlogterm := rf.EventLogs[prevlogindex].Term
	request := &AppendEntriesArgs{
		Term:         term,
		LeaderId:     leaderId,
		PrevLogIndex: prevlogindex,
		PrevLogTerm:  prevlogterm,
		Entries:      nil,
		LeaderCommit: rf.CommitIndex,
	}
	rf.mu.Unlock()
	reply := &AppendEntriesReply{}
	//log.Printf("Sending AppendEntries to %v", server)
	//log.Printf("peers: %v", rf.peers[server])
	ok := rf.peers[server].Call("Raft.AppendEntries", request, reply)
	if !ok {
		//log.Printf("SendHeartBeatToPeers failed for server %v", server)
	}
}

func (rf *Raft) SendHeartbeatImmediate() {
	rf.mu.Lock()
	//log.Printf("SendHeartbeatImmediate")
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
			//log.Printf("Leader inside sendheartbeatimmediate is: %v", leaderId)
			continue
		}
		//log.Printf("Sending heartbeat to %v", i)
		go rf.SendHeartBeatToPeers(i, term, leaderId) //send concrrently to increase speed.
	}

}

func (rf *Raft) PeriodicHeartbeats() {
	defer utils.RecoverWithStackTrace("PeriodicHeartbeats", rf.me)
	heartbeatInterval := 125 * time.Millisecond //to try and reduce RPC count.
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	//log.Printf("Heartbeat periodic every %v", heartbeatInterval)
	//when just became a leader send it right away.
	//rf.SendHeartbeatImmediate()
	for {
		select {
		// if not the leader. cant send out shit.
		case <-ticker.C:
			rf.mu.Lock()
			if rf.ServerState != Leader {
				//log.Printf("%v is no longer a leader and cant send out heartbeats", rf.me)
				rf.mu.Unlock()
				return
			}
			rf.mu.Unlock()
			//log.Printf("Sending out heartbeats now")
			rf.SendHeartbeatImmediate()
		}
	}
}

// HandleVoteReplies handles the votes that a server receives.
func (rf *Raft) HandleVoteReplies(reply *RequestVoteReply) {
	rf.mu.Lock()
	defer utils.RecoverWithStackTrace("HandleVoteReplies", rf.me)
	defer rf.mu.Unlock()

	if reply.Term > rf.CurrentTerm {
		rf.CurrentTerm = reply.Term
		rf.VotedFor = -1
		rf.VoteCount = 0
		rf.ServerState = Follower
		rf.persist()
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
				if i == rf.me {
					rf.MatchIndex[i] = len(rf.EventLogs) - 1
					continue
				}
				go rf.ReplicateLogsToFollower(i, rf.CurrentTerm)
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
		//log.Printf("Term of candidate %d cannot be less than current Term %d\n", args.Term, rf.CurrentTerm)
		return
		//return nil
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
	//the log with the later(greater term number) Term is more up-to-date. If the logs
	//end with the same Term, then whichever log is longer is
	//more up-to-date
	nodeLastLogIndex := len(rf.EventLogs) - 1
	nodeLastLogTerm := 0
	if nodeLastLogIndex >= 0 {
		nodeLastLogTerm = rf.EventLogs[nodeLastLogIndex].Term
	}

	upToDate := false
	if args.LastLogTerm > nodeLastLogTerm ||
		(args.LastLogTerm == nodeLastLogTerm && args.LastLogIndex >= nodeLastLogIndex) {
		upToDate = true
	}
	if (rf.VotedFor == -1 || rf.VotedFor == args.CandidateId) && upToDate {
		rf.VotedFor = args.CandidateId
		reply.VoteGranted = true
		// reset election timeout.
		rf.LastHeartBeat = time.Now()
		rf.persist()
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
	if rf.killed() {
		rf.mu.Unlock()
		return
	}
	rf.CurrentTerm++
	term := rf.CurrentTerm
	rf.ServerState = Candidate
	rf.VotedFor = rf.me
	rf.VoteCount = 1
	lastLogIndex := len(rf.EventLogs) - 1
	lastLogTerm := 0
	rf.persist()
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
				log.Printf("RequestVote Failed")
			} else {
				rf.HandleVoteReplies(reply)
			}
		}(i)
	}
}
func (rf *Raft) applier() {
	for !rf.killed() {
		rf.mu.Lock()
		for rf.LastApplied >= rf.CommitIndex && !rf.killed() {
			rf.Cond.Wait()
		}
		if rf.killed() {
			rf.mu.Unlock()
			return
		}
		for rf.LastApplied < rf.CommitIndex {
			rf.LastApplied++
			idx := rf.LastApplied
			cmd := rf.EventLogs[idx].Command
			msg := raftapi.ApplyMsg{CommandValid: true, Command: cmd, CommandIndex: idx}
			rf.mu.Unlock()
			rf.ApplicationChanel <- msg
			rf.mu.Lock()
		}
		rf.mu.Unlock()
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
	rf.EventLogs = make([]LogEntry, 1)
	rf.EventLogs[0] = LogEntry{Term: 0, Command: nil}
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
	//log.Printf("len of nextIndex: %v", len(rf.NextIndex))
	rf.LastHeartBeat = time.Now()
	rf.StopHeartBeat = make(chan bool, 1)
	// initialize from ServerState persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	rf.Cond = sync.NewCond(&rf.mu)
	// start ticker goroutine to start elections
	go rf.ticker()
	go rf.applier()
	//go rf.SendHeartbeats()
	//log.Printf("Methods on *Raft: %+v", reflect.TypeOf(rf))
	//for i := 0; i < reflect.TypeOf(rf).NumMethod(); i++ {
	//	log.Printf("Method: %s", reflect.TypeOf(rf).Method(i).Name)
	//}
	return rf
}
