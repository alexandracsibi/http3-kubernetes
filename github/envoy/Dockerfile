FROM envoyproxy/envoy:v1.28-latest

WORKDIR /etc/envoy

COPY envoy-kubernetes-config.yaml /etc/envoy/envoy.yaml
COPY servercert.pem /etc/envoy/servercert.pem
COPY serverkey.pem /etc/envoy/serverkey.pem
COPY ca.pem /etc/envoy/ca.pem

RUN mkdir -p /var/log/envoy && \
    chmod -R go+rw /var/log/envoy

RUN chmod go+r /etc/envoy/envoy.yaml \
    && chmod go+r /etc/envoy/servercert.pem \
    && chmod go+r /etc/envoy/serverkey.pem \
    && chmod go+r /etc/envoy/ca.pem

EXPOSE 7001 9901

CMD ["/usr/local/bin/envoy", "-c", "/etc/envoy/envoy.yaml", "--log-level", "debug"]
