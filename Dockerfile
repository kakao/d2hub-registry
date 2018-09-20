FROM registry:2.6.0

RUN apk add --no-cache supervisor

COPY ./registry-proxy /bin/registry-proxy
COPY ./config.yml /etc/docker/registry/config.yml
COPY ./ssl/* /ssl/
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

ENV REGISTRY_URL http://localhost:5000

ENTRYPOINT []
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
