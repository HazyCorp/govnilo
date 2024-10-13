FROM grafana/grafana:10.4.2-ubuntu as grafana

USER root

COPY grafana.ini /etc/grafana/grafana.ini

ENV GF_PATHS_PLUGINS=/usr/share/grafana/plugins/

RUN mkdir -p $GF_PATHS_PLUGINS

RUN grafana cli --pluginsDir ${GF_PATHS_PLUGINS} --pluginUrl https://github.com/VictoriaMetrics/victoriametrics-datasource/releases/download/v0.8.2/victoriametrics-datasource-v0.8.2.zip plugins install victoriametrics-datasource

