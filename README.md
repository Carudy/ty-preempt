# TY: Preempt
- Using auto assigned port numbers for all nodes, can set addr and start_port in params/config.go
- Default is 127.0.0.1, 32000 for client, 32{C}{N} for nodes in shard C and node N, e.g. 32101 for shard S0 node N0.

## Simulation
```
go build -o blockexe main.go

./run.sh
```


### Helper
- clean_db.sh: remove all tmp DB files
- kill_block.sh: kill all block processes
- run.sh: run batch blockchain nodes
  - num_shards and num_nodes are set in the script, will generate num_shards * num_nodes nodes
