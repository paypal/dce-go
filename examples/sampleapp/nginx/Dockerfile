FROM tutum/nginx
RUN rm /etc/nginx/sites-enabled/default
ADD sites-enabled/ /etc/nginx/sites-enabled
RUN mkdir -p /etc/nginx/ssl/certs
RUN mkdir -p /etc/nginx/ssl/private

COPY server.key /etc/nginx/ssl/private/
COPY server.crt /etc/nginx/ssl/certs/