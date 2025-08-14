# Raft Consensus Algorithm

This repository contains my implementation of the **Raft consensus algorithm** in Go, leader election has been 
implemented and is sufficiently passing tests even with concurrent data accesses and unreliable network. 

### Further Steps:
1. Implement log replication across the raft replicas
2. Implement Snapshotting
3. Implement Persistence to disk Mechanism.