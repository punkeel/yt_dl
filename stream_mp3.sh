#!/bin/sh
# $1 = youtubeID
# $2 = Video Title (Song name)
# $3 = Uploader name (Artist)

# First we grab the stream URL (youtube-dl)
# We stream it to stdout
# And avconv converts it to mp3, adding the correct metadata

curl $(youtube-dl -x --simulate --get-url $1) \
| avconv -i pipe:0 -c:a libmp3lame -f mp3 -metadata title="$2" -metadata artist="$3" pipe:1