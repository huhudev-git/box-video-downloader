# Box Video Downloader

**Personal use**: only available in Letus

Download the video on box shared inside Letus

## Requirement

- Go
- FFmpeg(to merge video and audio)

## Getting Started

```sh
git clone https://github.com/huhugiter/box-video-downloader.git
```

Copy your `box.com` cookies from Chrome into `cookies` file

**Important**: `box.com` cookies not `letus.tus.ac.jp` cookies

```sh
go run main.go -i <URL>
```

Example URL: `https://tus.app.box.com/s/xxxxxxxxxxxxxxxxxxxxxxxx`
Use `,` to split url, Example: `-i <URL>,<URL>,<URL>`

temp videos will be downloaded into temp/ folder, and complete video will output repo root folder, after finished temp videos will be deleted.

> Not download multi videos at the same time, the box.com has limit that will loss data

### FFmpeg

If you perfer to use docker, use command below

```sh
go run main.go -d -i <URL>
```

it will use `jrottenberg/ffmpeg:4.0-scratch` to merge video and audio.

## How to copy cookies

- Hit `F12` or Right-click page and Inspect to open the Developer Tools
- Repeat the request with the Network tab open.
- Right-click the relevant request (in the list on the left-hand side).
- Choose Copy as cURL.
- Extract the cookie from the generated Cookie header option.
