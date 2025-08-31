# Raft Consensus Algorithm

This repository contains my implementation of the **Raft consensus algorithm** in Go.  
Leader election and log replication have been implemented and are sufficiently passing tests even with concurrent data accesses and unreliable networks.

### Implemented Features
- Leader election with safety under concurrency and unreliable communication
- Log replication across Raft replicas, ensuring consistency of client commands
- Persistence of Raft state to disk to survive crashes and restarts

### Next Steps
1. Implement log compaction for efficient storage and retrieval of event logs for each server.