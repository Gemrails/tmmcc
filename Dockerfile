FROM debian:jessie-20171210

MAINTAINER barnett<zengqg@goodrain.com>

# Timezone
RUN echo "Asia/Shanghai" > /etc/timezone;dpkg-reconfigure -f noninteractive tzdata

ADD ./sources.list  /etc/apt/sources.list

RUN echo 'APT::Install-Recommends 0;' >> /etc/apt/apt.conf.d/01norecommends \
 && echo 'APT::Install-Suggests 0;' >> /etc/apt/apt.conf.d/01norecommends \
 && apt-get update \
 && apt-get install -y vim.tiny wget net-tools libpcap0.8-dev\
 && apt-get autoremove \
 && apt-get clean all \
 && rm -rf /var/lib/apt/lists/* /usr/share/doc/* /usr/share/man/* 

COPY tcm /bin

RUN chmod 755 /bin/tcm
#默认敏感端口
ENV PORT=5000
ENV SERVICE_ID=system

ENTRYPOINT ["/bin/tcm"]


