FROM node:22 AS build-stage

WORKDIR /app
COPY package*.json ./

RUN --mount=type=secret,id=github_token,required=true \
    GIT_TOKEN="$(cat /run/secrets/github_token)" && \
    test -n "$GIT_TOKEN" && \
    git config --global \
        url."https://oauth2:${GIT_TOKEN}@github.com/".insteadOf \
        "https://github.com/" && \
    yarn install && \
    git config --global --unset-all \
        url."https://oauth2:${GIT_TOKEN}@github.com/".insteadOf

COPY .env.development .env
COPY . .

RUN NODE_OPTIONS="--max-old-space-size=4096" yarn build --mode deployment

FROM nginx:alpine
RUN apk add --no-cache bash curl nano

ARG NGINX_HTML_PATH=/usr/share/nginx/html/gr-web-container

RUN mkdir -p $NGINX_HTML_PATH
COPY certificate /certificate

COPY --from=build-stage /app/dist/ $NGINX_HTML_PATH/
COPY Deployment.config/Develop/nginx.conf /etc/nginx/nginx.conf
#COPY wms-uat.conf /etc/nginx/conf.d/wms-uat.conf

EXPOSE 80
EXPOSE 443

CMD ["nginx", "-g", "daemon off;"]
