# Use multi-stage build as seen here: https://docs.docker.com/language/golang/build-images/#multi-stage-builds

# ----- Build stage -----
FROM golang:alpine AS build-env

# Copy all files to a temporary directory
COPY . /build

# cd to this directory
WORKDIR /build/
RUN go mod download

# Finally, build the Go app
RUN apk add git
RUN go build -o livekit-recorder .
# ----------

# ----- Final stage -----
# We use Alpine Linux as it's lightweight
FROM alpine

# Set environment variables
ENV APP_PORT 8000
ENV LIVEKIT_URL ""
ENV LIVEKIT_API_KEY ""
ENV LIVEKIT_API_SECRET ""
ENV S3_REGION ""
ENV S3_BUCKET ""
ENV S3_DIRECTORY ""
ENV WEBHOOK_URLS ""

# Install FFMPEG
RUN apk update && apk add ffmpeg

# Create final directory and the "copy" command does the trimming magic
RUN mkdir /exec
WORKDIR /exec
COPY --from=build-env /build/livekit-recorder /exec

# Don't forget to expose port
EXPOSE $APP_PORT

CMD ["./livekit-recorder"]
# ----------
