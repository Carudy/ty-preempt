#!/bin/bash

# Basic Blockchain Shard Network Runner
# Sets num_shards and num_nodes variables and starts all nodes
./kill_blockexe.sh  # Kill any existing blockexe processes
./clean.sh
# Configuration variables - edit these as needed
num_shards=4      # Number of shards (S0, S1, S2, S3)
num_nodes=2       # Number of nodes per shard (N0, N1)
malicious_num=1   # Maximum malicious nodes per shard
test_file="20W.csv"  # Test file

echo "=== Starting Blockchain Shard Network ==="
echo "Shards: $num_shards"
echo "Nodes per shard: $num_nodes"
echo "Malicious nodes per shard: $malicious_num"
echo "Test file: $test_file"
echo ""

# Check if blockexe exists
if [ ! -f "./blockexe" ]; then
    echo "ERROR: blockexe executable not found!"
    echo "Please build the project first: go build -o blockexe"
    exit 1
fi

# Check if test file exists
if [ ! -f "$test_file" ]; then
    echo "ERROR: Test file '$test_file' not found!"
    exit 1
fi

# Make blockexe executable if needed
if [ ! -x "./blockexe" ]; then
    chmod +x ./blockexe
    echo "Made blockexe executable"
fi

# Create log directory
log_dir="./process_logs"
mkdir -p "$log_dir"
echo "Logs will be saved to: $log_dir/"
echo ""

# Array to store PIDs
node_pids=()

echo "Starting nodes..."
for ((s=0; s<num_shards; s++)); do
    shard_id="S$s"
    for ((n=0; n<num_nodes; n++)); do
        node_id="N$n"
        log_file="$log_dir/${shard_id}_${node_id}.log"

        echo "Starting $shard_id $node_id..."

        # Start node with exact command format from requirements
        ./blockexe -S $num_shards -N $num_nodes -s $shard_id -n $node_id -t $test_file > "$log_file" 2>&1 &

        pid=$!
        node_pids+=($pid)

        # Brief pause between node starts
        sleep 0.2

        echo "  Started (PID: $pid, Log: $log_file)"
    done
done

echo ""
echo "All nodes started!"
echo ""

# Start client/supervisor
echo "Starting client/supervisor..."
client_log="$log_dir/client.log"

# Start client with -c flag as requested
./blockexe -c -S $num_shards -N $num_nodes -t $test_file  > "$client_log" 2>&1 &

client_pid=$!
node_pids+=($client_pid)

sleep 1
echo "Client started (PID: $client_pid, Log: $client_log)"
echo ""

echo "=== Network is running ==="
echo "Total processes: ${#node_pids[@]}"
echo "All output is being logged to: $log_dir/"
echo ""
echo "To view logs:"
echo "  tail -f $log_dir/*.log"
