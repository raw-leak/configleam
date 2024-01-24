# Start from a base image, e.g., Alpine with Golang
FROM golang:alpine

# Copy the Go application
COPY . /app

# Set working directory
WORKDIR /app

# Build the application
RUN go build -o configleam .

# Expose port 57752
EXPOSE 57752

# Command to run the executable
CMD ["./configleam"]
