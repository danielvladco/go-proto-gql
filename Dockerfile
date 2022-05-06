FROM scratch

WORKDIR /opt

COPY gateway ./gateway
COPY ./cmd/gateway/test.yaml config.yaml

ENTRYPOINT ["./gateway", "--cfg", "./config.yaml"]

CMD ["./gateway", "--cfg", "./config.yaml"]
