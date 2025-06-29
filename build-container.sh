#!/bin/bash

# Build script for the full-stack container

echo "Building container with both frontend and backend..."

# Build the Docker image
docker build -t miltechserver-fullstack .

echo "Container built successfully!"
echo ""
echo "To run the container:"
echo "docker run -p 8080:8080 miltechserver-fullstack"
echo ""
echo "Then access:"
echo "- Frontend: http://localhost:8080"
echo "- API: http://localhost:8080/api/v1/" 