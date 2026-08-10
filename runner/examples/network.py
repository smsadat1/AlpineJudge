import socket
import sys

try:
    s = socket.create_connection(("8.8.8.8", 53), timeout=5)
    s.close()
    print('Connection was successful')
    # If connection succeeds then sandbox has network connection (shouldn't happen)
    sys.exit(0)
except Exception as e:
    print(f'Connection failed: {e}')
    sys.exit(1) # Runtime Error