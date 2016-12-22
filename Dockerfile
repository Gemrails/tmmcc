FROM debian:8
MAINTAINER barnett<zengqg@goodrain.com>
# COPY . /go/src/tcm
# RUN apk update && apk add libpcap0.8-dev && rm -rf /var/cache/apk/* && \
#     cd /go/src/tcm && go get && go build -o tcm ./cmd && \
#     mv tcm /bin/tcm && rm -rf /go && chmod 755 /bin/tcm
RUN apt-get update && apt-get install -y libpcap0.8-dev
COPY tcm /bin

RUN chmod 755 /bin/tcm
#默认敏感端口
ENV PORT=5000

ENTRYPOINT ["/bin/tcm"]

CMD ["-i","eth0"]

