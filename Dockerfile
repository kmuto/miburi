FROM debian:bookworm-slim
LABEL maintainer="kmuto@kmuto.jp"

ENV LANG en_US.UTF-8
ENV DEBIAN_FROMTEND noninteractive

RUN sed -i -e "s/Components: main/& non-free/" /etc/apt/sources.list.d/debian.sources && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
      libsnmp-base snmp-mibs-downloader && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

CMD ["bash", "-c", "mkdir -p /work/mibs && cp /usr/share/snmp/mibs/*MIB* /work/mibs && cp -r /var/lib/mibs/* /work/mibs"]
