# Frontend (Vue 3 + Vite) — development image.
# Runs the Vite dev server; not a production build/serve setup.

FROM node:20-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
EXPOSE 5173
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]
