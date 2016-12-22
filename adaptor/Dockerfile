FROM barnettzqg/tcm:v1

RUN    apt-get install --no-cache  libc6-compat && \
       ln -s /lib /lib64 && \
       cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
       echo "Asia/Shanghai" >  /etc/timezone && \
       date
       
RUN sed -i -r 's/nofiles/rain/' /etc/group && \
    adduser -u 200 -D -S -G rain rain
    
RUN apk add --no-cache wget curl netcat-openbsd iputils net-tools bash su-exec && \
    sed -i -e "s/bin\/ash/bin\/bash/" /etc/passwd

RUN wget http://lang.goodrain.me/public/grproxy -O /tmp/grproxy

COPY docker-entrypoint.sh /tmp

RUN chmod +x /tmp/*

WORKDIR /tmp

ENTRYPOINT ["/tmp/docker-entrypoint.sh"]

CMD [""]