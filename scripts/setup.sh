# DISPATCHER
# sudo ./setup.sh dispatcher --env /path/to/.env --config /path/to/config.yaml

# Copy ajdispatcher
#     ↓
# /usr/bin/ajdispatcher

# Copy config
#     ↓
# /etc/ajdispatcher/config.yaml

# Copy .env
#     ↓
# /etc/ajdispatcher/.env

# Copy systemd service
#     ↓
# /etc/systemd/system/ajdispatcher.service

# systemctl daemon-reload
#     ↓
# systemctl enable ajdispatcher
#     ↓
# systemctl start ajdispatcher


# RUNNER 
# sudo ./setup.sh runner --id runner-001 --env /path/to/.env

# Copy ajrunner
#     ↓
# /usr/bin/ajrunner

# Generate runner config
#     ↓
# runner_id: runner-001

# Copy config
#     ↓
# /etc/ajrunner/config.yaml

# Copy .env
#     ↓
# /etc/ajrunner/.env

# Copy systemd service
#     ↓
# /etc/systemd/system/ajrunner.service

# systemctl daemon-reload
#     ↓
# systemctl enable ajrunner
#     ↓
# systemctl start ajrunner