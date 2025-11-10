# Build stage
FROM golang:1.25-alpine3.22 AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the Lambda binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bootstrap ./cmd/lambda

# Runtime stage - use AWS Lambda base image for custom runtimes
FROM public.ecr.aws/lambda/provided:al2023

# Copy the binary from builder
COPY --from=builder /app/bootstrap ${LAMBDA_RUNTIME_DIR}/bootstrap

# Set the CMD to your handler
CMD [ "bootstrap" ]
