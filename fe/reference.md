# Frontend Reference

The Frontend is a modern web application built using [Next.js](https://nextjs.org/) and standard React Hooks. It connects directly to the proxy server (`http://localhost:8000`) and serves two primary purposes:

1. A real-time Administrative Dashboard to manage users, quotas, and global rate limits seamlessly.
2. A Chat Simulator that mocks the experience of end-users integrating their own software with the proxy's API.

## Starting the Application

Inside the `fe/` directory, run the following commands to install dependencies and start the development server:

```bash
npm install
npm run dev
```

The application will be accessible at `http://localhost:3000`.

---

## Core Routes

The application utilizes Next.js App Router conventions:

| Route Path | Description                                                                                                                                                                                        | Access Level            |
| ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------- |
| `/`        | **Admin Dashboard:** Overview of user tokens, server limits, and quick-actions like suspending a user account or manipulating global `.env`-style API limits on the fly.                           | Admin Only              |
| `/login`   | **Authentication:** Login portal. Simulates an OAuth or DB-backed login procedure. Validates credentials directly against the Proxy endpoint.                                                      | Public                  |
| `/chat`    | **Chat Playground:** A beautiful chat interface integrating Server-Sent Events (SSE) natively. Supports auto-switching to Vision models upon detecting an image upload. Includes chat persistence. | Standard Users / Admins |
| `/usage`   | **My Usage:** Displays personal token consumption isolated by model. End-users can see exactly what they've consumed.                                                                              | Standard Users / Admins |

---

## Key Components

### 1. The Real-time Chat Experience (`/chat`)

Located at `src/app/chat/page.tsx`, this component implements a full streaming chat UI utilizing the OpenAI-compliant inference endpoint.

**Features include:**

- Zero-buffer Server-Sent Event (SSE) text decoding utilizing `TextDecoder` and standard `fetch` readers. Meaning tokens appear sequentially on the screen instantaneously as the inference engine pushes them out instead of lagging.
- Client-side history storage via `localStorage`, persisting conversation pairs between page reloads with intelligent length/byte cap truncations to prevent browser QuotaExceeded errors limit when interacting with high-res vision payloads.
- Integrated file-upload mechanics. Attaching an image automatically swaps the target `model` to `moondream` and packages the payload into OpenAI's multimodal vision JSON format.

### 2. The Admin Dashboard (`/`)

Located at `src/app/page.tsx`, this route acts as the command center.

**Features include:**

- A dynamically polling data table that hits `GET /admin/limits` to showcase active proxy settings and aggregated user consumption counts.
- Administrative form submissions updating critical performance variables dynamically (like Requests Per Second limits, or Max Tokens bounds). These requests `POST` straight back to the Proxy which atomically updates its live Go memory arrays without requiring a backend restart.
- An action-trigger pipeline to `POST /admin/suspend` directly clicking on a user's row, forcibly injecting a `403` ban state for that user's api-key effectively muting them.

### 3. Authentication Flow (`src/lib/auth.ts`)

A lightweight mocked authentication layer bridging the frontend UI with the deterministic user structure present in the proxy's memory pool.

- Exposes logic such as `login()`, `saveSession()`, and `clearSession()`.
- Identifies users as "Admins" allowing correct React routing and Dashboard rendering.

---

## API Communication Details

All upstream calls hit `http://localhost:8000`.

- **CORS:** The Next.js frontend runs natively on Port `3000` while making fetch requests directly to the Go backend on Port `8000`. The backend handles standard OPTIONS pre-flights via the Echo middleware to allow explicit cross-origin connections alongside `Authorization` Bearer headers.
- **SSE:** Because Next.js server actions / proxying often buffer streaming connections until the entire payload is finished (breaking ChatGPT-like typing UX), the frontend avoids SSR logic for the inference stream and instead fetches the data utilizing React Client Components directly connecting to the Proxy.
