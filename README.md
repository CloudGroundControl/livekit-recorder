# Disclaimer

This project has been stable for our use at Cloud Ground Control. However, we will be moving to `livekit-egress` when it comes out. The good news is I will be contributing to that project, so for any suggestions, you can file it here or talk to us on Slack!

# LiveKit Recorder

Service to record each participant in LiveKit room without transcoding. Prioritises minimum resource consumption for scalability. For composite recording (multiple participants in one layout), check out https://github.com/livekit/livekit-recorder.

## Features

- [x] Single track recording
- [x] Track composite recording (video + audio)
- [x] Docker support
- [x] Upload to S3
- [x] Structured logging
- [ ] Custom file name
- [ ] Job queue

## Motivation

The original https://github.com/livekit/livekit-recorder was intended mainly for compositing participants, but consumes significant resources due to Chrome. We only want to record each participant, and that recorder was too expensive to scale. Our solution is to record each RTP stream via LiveKit's `server-sdk-go`.

## How it works

The project has a service which is responsible for managing <strong>recordbots</strong>. The bots use selective subscription so we can have multiple bots in the room without duplicated recording, making it scalable through a Load Balancer. To stop the recording, either send a POST request to stop, or disconnect the participant from the room. We then use `ffmpeg` to containerise the output files.

## Prerequisite

Make sure you have `ffmpeg` installed (https://ffmpeg.org/download.html)

## Quickstart

First, start the server

```
LIVEKIT_URL=ws://... \
LIVEKIT_API_KEY=... \
LIVEKIT_API_SECRET=... \
APP_PORT=8000 \
LOG_LEVEL=debug \
go run main.go
```

Then, join the LiveKit room `my-room` as `my-participant`.

Next, create a POST request to `/recordings/start` with the body:

```
{
    "room": "my-room",
    "participant": "my-participant",
}
```

After recording for some time, to stop, either disconnect from the room, or create a POST request to `/recordings/stop` with the body:

```
{
    "room": "my-room",
    "participant: "my-participant"
}
```

You should have a file in the `recordings/` folder.

## Triggers

There are 2 ways to perform recording.

#### Manual

Use POST endpoints `/recordings/start` and `/recordings/stop` with the payload specified in quickstart.

#### Webhooks

The endpoint `/recordings/webhooks` will readily receive LiveKit's webhooks. Recording will start when receiving the event `participant_joined` and stop automatically on the event `participant_left`.

## Environment Variables

#### Required

| Flag               | Description                                        |
| ------------------ | -------------------------------------------------- |
| APP_PORT           | Exposed port for the service                       |
| LOG_LEVEL          | One of `debug`, `info`, `warn`, `error`            |
| LIVEKIT_URL        | Use `ws` format, will convert to `http` internally |
| LIVEKIT_API_KEY    | Your LiveKit API key                               |
| LIVEKIT_API_SECRET | Your LiveKit API secret                            |

#### S3 upload

To enable S3 upload, make sure to set the following variables. Both region and bucket variables are required to create an S3 uploader.

| Flag         | Description          |
| ------------ | -------------------- |
| S3_REGION    | AWS region           |
| S3_BUCKET    | Name of S3 bucket    |
| S3_DIRECTORY | Optional, read below |

For our use case, we have one bucket for different environments. If we specify `S3_DIRECTORY=livekit` and a file named `my-file.mp4`, the resulting file will be saved as `livekit/my-file.mp4` on S3.

## Deployment

We have shipped a Dockerfile which can be built locally. Unfortunately we don't have plans to have a DockerHub account, so you'll need to clone the repository, build the image, and push to container registry of your choice (ECR, etc.).

## Issues

For any bug reports or the service not working as expected, either file an issue here or contact me in LiveKit's Slack. You can also reach out to me via email at [dennis.wirya@advancednavigation.com](mailto:dennis.wirya@advancednavigation.com) .
