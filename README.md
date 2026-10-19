# VeloCloud Academy

A personal, local-only interactive course for learning Arista VeloCloud SD-WAN from zero to hero.
See `CLAUDE.md` for conventions and `docs/DECISIONS.md` for the decision log.

## Run

```sh
make up        # creates .env from .env.example if missing, builds and starts everything
```

Open http://localhost:5173 — the Home page should show Backend OK and Database OK.
`make down` stops, `make logs` tails, `make test` / `make lint` / `make fmt` for checks, `make reset-db` wipes progress.
