# LiveKit Recorder

Service to record each participant in LiveKit room without transcoding. Prioritises minimum resource consumption for scalability. For composite recording (multiple participants in one layout), check out https://github.com/livekit/livekit-recorder.

## TODO

- [x] Single track recording
- [x] Add Dockerfile
- [x] Upload to S3
- [x] Mux video + audio tracks
- [x] Docker support
- [] Custom file name
- [] Job queue

## Motivation

The original https://github.com/livekit/livekit-recorder was intended mainly for compositing participants, but consumes significant resources as it runs a Chrome instance. We only want to record each participant, and running the composite recorder per participant was too expensive to scale. Our solution is to record each RTP track via LiveKit's `server-sdk-go` without using Chrome.

## How it works

The project has a service which is responsible for managing <strong>recordbots</strong>. The bots do not subscribe to all the participants; instead, we need to send a POST request specifying which room and participant to record, and the output name. This means we can have multiple bots in the room without duplicated recording, making it scalable through a Load Balancer. To stop the recording, either send a POST request to stop, or disconnect the participant from the room. We then use `ffmpeg` to containerise the output files.

## Prerequisite

Make sure you have `ffmpeg` installed (https://ffmpeg.org/download.html)

## Quickstart

First, start the server

```
LIVEKIT_URL=ws://... \
LIVEKIT_API_KEY=... \
LIVEKIT_API_SECRET=... \
APP_PORT=8000 \
APP_DEBUG=true \
go run main.go
```

Then, join the LiveKit room `my-room` as `my-participant`.

Next, create a POST request to `/recordings/start` with the body:

```
{
    "room": "my-room",
    "participant": "my-participant",
    "profile": "av / video / audio"
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

## S3 upload

To enable S3 upload, make sure to set the following variables:

```
S3_REGION=
S3_BUCKET=
S3_DIRECTORY=
```

The variable `S3_DIRECTORY` is optional. For our use case, our bucket stores recordings for different environments. For `S3_DIRECTORY=livekit` and a file named `my-file.mp4`, the resulting file will be saved as `livekit/my-file.mp4` on S3.

## Issues

For any bug reports or the service not working as expected, either file an issue here or contact me in LiveKit's Slack. You can also reach out to me via email at [dennis.wirya@advancednavigation.com](mailto:dennis.wirya@advancednavigation.com) .
