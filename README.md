# Raft Consensus Algorithm

This repository contains my implementation of the **Raft consensus algorithm** in Go.  
Leader election and log replication have been implemented and are sufficiently passing tests even with concurrent data accesses and unreliable networks.

### Implemented Features
- Leader election with safety under concurrency and unreliable communication
- Log replication across Raft replicas, ensuring consistency of client commands

### Next Steps
1. Implement snapshotting for efficient state transfer and log compaction
2. Implement persistence to disk to survive process crashes and restarts