FROM nginx:1.26.0

COPY ./api_gateway.conf /etc/nginx/conf.d/api_gateway.conf
COPY ./Certificate/cert.crt /etc/nginx/certs/cert.crt
COPY ./Certificate/cert.key /etc/nginx/certs/cert.key
RUN rm /etc/nginx/conf.d/default.conf

RUN apt-get update && apt-get install -y unzip wget && \
    cd /opt && \
    wget https://github.com/open-telemetry/opentelemetry-cpp-contrib/releases/download/webserver%2Fv1.1.0/opentelemetry-webserver-sdk-x64-linux.tgz && \
    tar xvfz opentelemetry-webserver-sdk-x64-linux.tgz && \
    cd /opt/opentelemetry-webserver-sdk && \
    ./install.sh

ARG LD_LIBRARY_PATH_INIT
ENV LD_LIBRARY_PATH=/opt/opentelemetry-webserver-sdk/sdk_lib/lib:${LD_LIBRARY_PATH_INIT:-}

RUN echo "load_module /opt/opentelemetry-webserver-sdk/WebServerModule/Nginx/1.26.0/ngx_http_opentelemetry_module.so;\n$(cat /etc/nginx/nginx.conf)" > /etc/nginx/nginx.conf

COPY opentelemetry_module.conf /etc/nginx/conf.d