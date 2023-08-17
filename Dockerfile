#######################
##### BUILD STAGE #####
#######################
FROM golang:1.20-alpine AS build_stage

# This can't be `/workspace` or kaniko in cloud build breaks
WORKDIR /app

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .
RUN go mod download

RUN mkdir cmd
RUN mkdir cmd/llproxy
COPY cmd/llproxy/ cmd/llproxy/

# Build the Go app
RUN GOOS=linux GOARCH=amd64 go build ./cmd/llproxy
RUN adduser --disabled-password --shell /bin/ash llp
USER llp

# Unit tests
COPY test.sh .
RUN ./test.sh

#########################
##### RUNTIME STAGE #####
#########################
FROM alpine:3.18

# Create user so we don't run as root
RUN addgroup --gid 1000 llp && \
  adduser --disabled-password --uid 1000 --ingroup llp --shell /bin/ash --home /home/llp llp
USER llp

WORKDIR /home/llp/

# Copy binary from build stage
COPY --chown=1000:1000 --from=build_stage /app/llproxy .
COPY config.json .

EXPOSE 8080
EXPOSE 8081

CMD [ "./llproxy" ]