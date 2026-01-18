#!/bin/sh

# Default values
PORT=${PORT:-9999}
HEARTBEAT_INTERVAL=${HEARTBEAT_INTERVAL:-5s}
TIMEOUT=${TIMEOUT:-15s}
NODE_ID=${NODE_ID:-node-unknown}
SEED_NODE=${SEED_NODE:-}

# Build command
CMD="/app/pulsecheck --port $PORT --heartbeat-interval $HEARTBEAT_INTERVAL --timeout $TIMEOUT --node-id $NODE_ID"

# Add seed node if provided
if [ -n "$SEED_NODE" ]; then
    CMD="$CMD --seed-node $SEED_NODE"
fi

# Execute
exec $CMD
