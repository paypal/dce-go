#Mesos build instructions from https://mesos.apache.org/gettingstarted/
FROM ubuntu:trusty

RUN apt-get update
ENV DEBIAN_FRONTEND noninteractive
RUN apt-get install -y tar wget git
RUN apt-get install -y openjdk-7-jdk
RUN apt-get install -y autoconf libtool
RUN apt-get install -y build-essential python-dev libcurl4-nss-dev libsasl2-dev libsasl2-modules maven libapr1-dev libsvn-dev
RUN apt-get install -y libcurl3 libcurl3-nss

ARG MESOS_VERSION=master
ARG MESOS_REPO=https://git-wip-us.apache.org/repos/asf/mesos.git
ENV MESOS_VERSION=${MESOS_VERSION}
ENV MESOS_REPO=${MESOS_REPO}

RUN git clone -b ${MESOS_VERSION} ${MESOS_REPO} /mesos
WORKDIR /mesos
RUN ./bootstrap
RUN mkdir build

WORKDIR /mesos/build

RUN ../configure --enable-optimize --enable-install-module-dependencies --disable-java --disable-optimize && make -j2 V=0 && make install

COPY mesos-modules /mesos-modules
WORKDIR /mesos-modules
RUN ./bootstrap && \
    rm -Rf build && \
    mkdir build
WORKDIR /mesos-modules/build
RUN PATH=/usr/local/lib/mesos/3rdparty/bin:$PATH ../configure --with-mesos-root=/usr/local/lib/mesos  --with-mesos-build-dir=/usr/local/lib/mesos/3rdparty && make

VOLUME ["/output"]

CMD "DESTDIR=/output make install"