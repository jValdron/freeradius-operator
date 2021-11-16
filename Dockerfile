FROM node:16-alpine

WORKDIR /app
COPY package.json .

RUN npm install
COPY . .
RUN ./node_modules/.bin/tsc

CMD ["node", "./dist/app.js"]