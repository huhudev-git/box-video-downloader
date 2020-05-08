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

Copy your `box.com` cookies from Chrome or something else write into `cookies` file

**Important**: `box.com` cookies not Letus Cookies

```sh
go run main.go -i <URL>
```

Example URL: `https://tus.app.box.com/s/xxxxxxxxxxxxxxxxxxxxxxxx`

## How to copy cookies

- Hit `F12` or Right-click page and Inspect to open the Developer Tools
- Repeat the request with the Network tab open.
- Right-click the relevant request (in the list on the left-hand side).
- Choose Copy as cURL.
- Extract the cookie from the generated Cookie header option.
