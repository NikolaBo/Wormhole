FROM mcr.microsoft.com/cbl-mariner/base/nodejs:16

WORKDIR /usr/app

RUN npm install express
ADD app.js /usr/app/app.js
ENTRYPOINT ["node", "app.js"]