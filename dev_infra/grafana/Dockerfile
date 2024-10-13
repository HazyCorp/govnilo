FROM grafana/grafana:10.4.2-ubuntu as grafana

COPY grafana.ini /etc/grafana/grafana.ini

RUN grafana cli --pluginUrl https://github.com/VictoriaMetrics/victoriametrics-datasource/releases/download/v0.8.2/victoriametrics-datasource-v0.8.2.zip --pluginsDir ${GF_PATHS_PLUGINS} plugins install victoriametrics-datasource

RUN grafana cli --pluginUrl  https://github.com/VictoriaMetrics/victorialogs-datasource/releases/download/v0.2.1/victorialogs-datasource-v0.2.1.zip  --pluginsDir ${GF_PATHS_PLUGINS} plugins install victorialogs-datasource

