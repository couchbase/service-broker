FROM scratch

ADD build/bin/broker /usr/local/bin/broker

ENTRYPOINT ["/usr/local/bin/broker"]
