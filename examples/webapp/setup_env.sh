#!/bin/bash

# Sets WEBAPP_* variables for the Synthra webapp example.
# Usage: source examples/webapp/setup_env.sh

echo "Setting WEBAPP_* variables."

export WEBAPP_SERVER_HOST=0.0.0.0
export WEBAPP_SERVER_PORT=8080
export WEBAPP_SERVER_TLS_ENABLED=false

export WEBAPP_DATABASE_PRIMARY_HOST=localhost
export WEBAPP_DATABASE_PRIMARY_PORT=5432
export WEBAPP_DATABASE_PRIMARY_DATABASE=myapp
export WEBAPP_DATABASE_POOL_MAX_OPEN=25
export WEBAPP_DATABASE_POOL_MAX_IDLE=5

export WEBAPP_AUTH_JWT_SECRET=your-super-secret-jwt-key-here
export WEBAPP_AUTH_TOKEN_DURATION=24h

export WEBAPP_FEATURES_DEBUG_MODE=false

echo "Done. Run: cd examples/webapp && go run ."
