# Gojinn Cluster State Machine & Consensus

Gojinn operates as a formally modeled distributed system. By replacing external message brokers and databases with an embedded NATS JetStream and LibSQL topology, Gojinn enforces strict mathematical guarantees for state distribution.

## 1. The Consensus Engine (Raft)
Gojinn utilizes the **Raft Consensus Algorithm** embedded within NATS JetStream to manage the distributed Key-Value store and Message Queues.
- **Leader Election:** Deterministic and automatic. If a node fails, the remaining nodes hold an election. A new leader is established within milliseconds provided a quorum (N/2 + 1) is maintained.
- **Quorum Requirement:** Dictated by the `cluster_replicas` parameter in the Caddyfile.

## 2. Consistency vs. Availability (CAP Theorem)
Gojinn allows architects to define exactly how the system should behave during a Network Partition (Split-Brain) via namespace policies.

### CP Mode (Consistency / Partition Tolerance)
- **Use Case:** Financial transactions, Distributed Locks (Mutex), Inventory management.
- **Behavior:** The system enforces strict quorum. If a node loses connection to the majority, it **rejects** all read and write requests to prevent split-brain mutations (`stale_reads false`).

### AP Mode (Availability / Partition Tolerance)
- **Use Case:** User sessions, UI preferences, cache lookups.
- **Behavior:** The system prioritizes uptime. If a node is isolated, it accepts local reads even if they are outdated (`stale_reads true`).

## 3. Distributed Mutex State Machine
The `host_mutex_lock` function guarantees deterministic execution across the mesh.
1. **INIT:** WASM worker requests lock for `Key_A`.
2. **COMPARE-AND-SET:** NATS executes an atomic `Create` operation.
3. **STATE (LEADER):** If `Create` succeeds, the node enters `LEADER` state.
4. **STATE (FOLLOWER):** If `Create` fails (Key exists), the node enters `FOLLOWER` state and aborts critical path execution.
5. **RELEASE:** Leader executes `host_mutex_unlock`, deleting the key and resetting the state.

## 4. Deterministic Failover
When a node crashes unexpectedly:
- **Workers:** JetStream detects missing acknowledgments (`NakWithDelay`) and automatically redelivers the payload to the next available healthy worker in the `WORKERS_` queue group.
- **Data:** LibSQL embedded replicas seamlessly switch to read-only mode until a new primary heartbeat is established via the WebSocket tunnel.