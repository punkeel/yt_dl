FROM golang:alpine
ENV EDGE_REPOSITORY "http://dl-4.alpinelinux.org/alpine/edge/main/"
RUN apk update --repository $EDGE_REPOSITORY \
	&& apk add py-pip ca-certificates \
	&& apk add ffmpeg --repository $EDGE_REPOSITORY \
	&& rm -rf /var/cache/apk/*
RUN pip install youtube-dl==2015.08.28

EXPOSE 80
ENV PORT 80

ENTRYPOINT ["go run yt_dl.go"]
