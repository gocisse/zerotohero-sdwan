# VeloCloud Academy

A personal, local-only interactive course for learning Arista VeloCloud SD-WAN from zero to hero: lessons, quizzes, a
teaching-model simulator, labs, a capstone and a self-check. No login, no accounts; one implicit local profile.
See `CLAUDE.md` for conventions and `docs/DECISIONS.md` for the decision log.

## What you need

- Docker with `docker-compose` (the Compose plugin or the standalone binary) and `make`.
- To run the checks yourself: Go (the version in `backend/go.mod`) and Node 22 or newer (`make test`, `make lint`, `make e2e`). Running the app needs only Docker.

## Run

```sh
git clone <this repository> velocloud-academy
cd velocloud-academy
make up        # creates .env from .env.example if missing, builds and starts everything
```

Open http://localhost:5173 (the port is `FRONTEND_PORT` in `.env`). The Home page should show Backend OK and Database OK.
The API answers at http://localhost:8090/api/health.

- `make down` stops, `make logs` tails the logs, `make help` lists every target.
- `make test` (backend tests against a throwaway Postgres database, content validation, type-check), `make lint`, `make fmt`.
- `make e2e` runs the browser regression suite in its own isolated stack (separate containers, database volume and ports 5174 / 8091 / 5433), so it never touches your progress.
- `make reset-db` deletes your progress (asks for confirmation).

## Where things are

- `content/` is the course: lessons, quizzes, diagrams, glossary, simulator topologies and labs, plus the research notes in `content/reference/`.
- `docs/OBSERVATION-PRIORITIES.md` is your lab checklist: what to check in a real Orchestrator first. `docs/HOW-TO-ADD-AN-OBSERVATION.md` says what to do afterwards.
- `docs/COURSE-MAP.md` shows which lessons, quizzes, diagrams, labs and simulator screens support each of the four goals.
