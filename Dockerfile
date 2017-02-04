FROM busybox

WORKDIR /opt/dns-tool

ADD ./build/linux_amd64/dns-tool .

ENTRYPOINT ["/opt/dns-tool/dns-tool"]
