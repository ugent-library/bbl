FROM node:22-alpine
WORKDIR /app
RUN apk add --no-cache git \
    && git clone --recurse-submodules --depth 1 https://github.com/zotero/citeproc-js-server.git . \
    && npm install --production \
    && apk del git
EXPOSE 8085
CMD ["node", "lib/citeServer.js"]
