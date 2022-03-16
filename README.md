# Video Gallery Generator
The command line application generates thumbnails for videos and run the builtin WebServer to serve the HTML gallery and play the videos.
![](img/1.png)

- Check the [releases](https://github.com/gadelkareem/video-gallery-generator/releases) page for executable versions.
- By default, the application will only run the server on http://0.0.0.0:8282/. To generate thumbnails for all videos in a directory, use the `-g` option.

# Prerequisites
- FFmpeg installed on your system. Download [here](https://ffmpeg.org/download.html).
- Python 2.7 installed (only in case of using spatial media flag `-s`). More info [here](https://github.com/google/spatial-media).

# Usage
```bash
# Run the server using current directory as the root directory
./vgg-darwin-arm64 
# Run the server using current directory as the root directory and generate thumbnails for all videos in the current directory
./vgg-darwin-arm64 -g
# Run the server using a specific path as the root directory and different port
./vgg-darwin-arm64 -d /path/to/videos -p 8080
# Control the maximum number of generators to run concurrently
./vgg-darwin-arm64 -g -c 10
# Add spatial media metadata to videos. This will rename the videos and add '_180x180_3dh' suffix to the video file name. Only left-right 180 is currently supported. 
./vgg-darwin-arm64 -s
```

# Build for all platforms
```shell
./build.sh
```