FROM golang:1.21.6-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Start a new stage from scratch
FROM alpine:latest

# Create the /app directory
RUN mkdir /app

# Set the working directory
WORKDIR /app

# Copy the built binary from the previous stage
COPY --from=builder /app/main .

# Copy the frontend directory from the previous stage
COPY --from=builder /app/frontend ./frontend

EXPOSE 8080

CMD ["./main"]