import re
import os

with open("mairu/internal/contextsrv/server.go", "r") as f:
    content = f.read()

# We'll just do it manually via python or bash.
