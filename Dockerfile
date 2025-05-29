# Start with a base golang image
FROM golang:1.23-alpine

# Install required packages
RUN apk add --no-cache git build-base pkgconfig czmq-dev libzmq libsodium-dev

# Install sqlc
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Install go-migrate
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Install go-task
RUN go install github.com/go-task/task/v3/cmd/task@latest

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to the working directory
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the entire source code to the working directory
COPY . .

# Run task commands: sqlc generate and go build
RUN task sqlc
RUN task build

# Add build directory to PATH
ENV PATH="/app/build:${PATH}"

# Expose the required port
EXPOSE 5556

# Run the executable
CMD ["./build/proxy"]

