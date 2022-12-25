FROM scratch

WORKDIR /opt

COPY gateway ./gateway

ENTRYPOINT ["./gateway"]
CMD ["./gateway"]
