FROM centos:7
LABEL maintainer="droslean@gmail.com"

ADD events-reporter /usr/bin/events-reporter
ENTRYPOINT ["/usr/bin/events-reporter"]
