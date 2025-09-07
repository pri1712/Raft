# Raft Consensus Algorithm

This repository contains my implementation of the **Raft consensus algorithm** in Go.  
The project focuses on building a fault-tolerant, strongly consistent replicated state machine that remains robust under concurrency, unreliable communication, and crash-recovery scenarios.

## Features Implemented
- **Leader Election** : Safe, concurrent election process ensuring a single valid leader even under failures.
- **Log Replication** : Commands replicated across all servers with strong consistency guarantees.
- **Persistence** : Raft state durably stored on disk, allowing recovery after crashes or restarts.
- **Log Compaction (Snapshotting)** : Old log entries are compacted into snapshots, reducing storage overhead and improving recovery speed.

## Usecases
Raft is the industry standard for pretty much any distributed systems consensus due to its ease of usage as compared to Paxos as well as correctness in the face of various types of failures as mentioned above.
## Next Steps
- Optimize snapshot installation and recovery.
- Add support for client-facing applications (e.g., a distributed key-value store).
- Explore extensions such as joint consensus and dynamic cluster reconfiguration.  