# 🚀 Deploy Connect Four to Render

## Quick Deploy Steps

### 1. Backend Service
1. Go to https://dashboard.render.com/
2. **New + → Web Service** → Connect GitHub repo
3. **Settings:**
   - Name: `connect-four-backend`
   - Environment: `Go`
   - Build Command: `go build -o main ./cmd/server`
   - Start Command: `./main`

### 2. Database
1. **New + → PostgreSQL**
2. Name: `connect-four-db`, Database: `connect_four`

### 3. Link & Configure
1. Backend service → **Environment** → **Link Database**
2. **Add variables:**
   ```
   GIN_MODE=release
   BOT_TIMEOUT_SECONDS=10
   CORS_ORIGINS=*
   ```

### 4. Frontend
1. **New + → Static Site** → Same GitHub repo
2. **Settings:**
   - Name: `connect-four-frontend`
   - Build Command: `cd web && npm ci && npm run build`
   - Publish Directory: `web/build`
3. **Add variables:**
   ```
   REACT_APP_API_URL=https://YOUR-BACKEND-URL/api
   REACT_APP_WS_URL=wss://YOUR-BACKEND-URL/ws
   ```

### 5. Update CORS
Update backend `CORS_ORIGINS` with your frontend URL.

## 🎯 Live URLs
- **Game:** `https://connect-four-frontend-xxxx.onrender.com`
- **API:** `https://connect-four-backend-xxxx.onrender.com`

## Test
- Health: `/health`
- Game: Visit frontend URL
- API: `/api/leaderboard`