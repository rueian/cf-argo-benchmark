FROM grafana/grafana:6.5.1

ADD ./grafana.db /grafana.db
ADD ./init.sh /init.sh

ENTRYPOINT ["/init.sh"]