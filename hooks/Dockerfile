FROM mcr.microsoft.com/cbl-mariner/base/nodejs:16

WORKDIR /usr/app

RUN npm install express
RUN npm install debug
ADD app.js /usr/app/app.js
ADD hooks /usr/app/hooks
ENTRYPOINT ["node", "app.js"]