# Go Datastar Realtime Counter

Minimalne demo stacku do budowania lekkich SaaS-ow w Go:

- Echo v5.1 jako prosty HTTP router.
- templ jako typowane komponenty HTML kompilowane do Go.
- Datastar jako cienka warstwa realtime przez SSE i HTML patches.
- Zog jako walidacja wejscia po stronie serwera.
- pgx + sqlc + goose jako opcjonalna sciezka Postgres: typowany SQL i migracje.

Demo celowo nie ma Reacta, bundlera frontendu, JSON API ani Firebase SDK. Stan jest na serwerze, a UI jest tylko jego projekcja.

## Quick Start

```bash
docker build -t go-datastar-counter-demo .
docker run --rm -p 8080:8080 go-datastar-counter-demo
```

Otworz `http://localhost:8080` w dwoch kartach lub dwoch przegladarkach. Klikniecie `+` albo `-` w jednej karcie aktualizuje wszystkie pozostale przez Server-Sent Events.

Domyslnie aplikacja uzywa pamieci procesu, zeby jeden obraz Dockerowy dzialal bez dodatkowych uslug.

## Co Demo Pokazuje

To jest maly, konkretny odpowiednik "Firebase realtime counter", ale bez vendor lock-in i bez przenoszenia logiki biznesowej do przegladarki.

```text
Browser A                 Go process                         Browser B
---------                 ----------                         ---------
click "+"  -- POST -->  Echo handler
                         Zog validation
                         Memory/Postgres store
                         Hub broadcast

open SSE  <-- patch --   templ renders HTML   -- patch -->   open SSE
DOM update                Datastar SSE event                 DOM update
```

Najwazniejsza idea:

```text
state = server
UI = rendered HTML
change = POST action
sync = SSE patch to every connected client
```

## Jak Dziala Realtime

Kazda karta po zaladowaniu strony odpala stale polaczenie:

```html
data-init="@get('/events', {openWhenHidden: true})"
```

Przyciski nie wywoluja lokalnego `useState` ani `fetch().then(json)`. One tylko wysylaja intencje do serwera:

```html
data-on:click="@post('/counter/increment')"
data-on:click="@post('/counter/decrement')"
```

Serwer trzyma liste subskrybentow w `Hub`:

```go
subs map[chan Snapshot]struct{}
```

Kazdy klient SSE dostaje swoj kanal. Po zmianie licznika `Hub` rozsyla nowy `Snapshot` do wszystkich kanalow, renderuje `views.CounterCard(snapshot)` i Datastar podmienia element `#counter-card` w kazdej przegladarce.

## Architektura

```text
cmd/server
  main.go              HTTP routes, startup, store selection

internal/counter
  hub.go               subscriptions, broadcast, realtime fan-out
  counter.go           shared Snapshot model

internal/store
  memory.go            default in-memory counter store
  postgres.go          optional pgx/sqlc-backed store

internal/validate
  counter.go           Zog validation for counter actions

internal/db
  *.go                 sqlc-generated typed query layer

views
  page.templ           templ components for full page and counter card

migrations
  *.sql                goose migrations

queries
  counter.sql          source SQL used by sqlc
```

Warstwy:

```text
HTTP action
  -> Echo route
  -> Zog validation
  -> Store interface
  -> Hub broadcast
  -> templ render
  -> Datastar SSE patch
```

## Pamiec Procesu vs Postgres

Domyslnie licznik siedzi w RAM procesu Go:

```go
type Memory struct {
    mu    sync.Mutex
    value int32
}
```

To wystarcza do demo, bo wszystkie przegladarki lacza sie z tym samym procesem. `sync.Mutex` chroni licznik przed jednoczesnymi kliknieciami.

Ograniczenie: restart kontenera zeruje licznik.

Jesli ustawisz `DATABASE_URL`, aplikacja wybierze store Postgres:

```text
DATABASE_URL set     -> Postgres store
DATABASE_URL missing -> Memory store
```

Wtedy:

- goose zaklada tabele przy starcie,
- sqlc generuje typowane funkcje z `queries/counter.sql`,
- pgx wykonuje zapytania bez ORM-a,
- stan przetrwa restart aplikacji.

## Tryb z Postgres

```bash
docker run --rm --name counter-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=counter \
  -p 5432:5432 \
  postgres:17-alpine

DATABASE_URL='postgres://postgres:postgres@localhost:5432/counter?sslmode=disable' go run ./cmd/server
```

Przy starcie aplikacja wykonuje migracje z `migrations/` przez goose. Kod w `internal/db/` jest wygenerowany przez sqlc z pliku `queries/counter.sql`.

## Dlaczego Ten Stack Jest Ciekawy

Dla osoby przyzwyczajonej do Next.js/Firebase najwieksza roznica to miejsce trzymania stanu i ilosc warstw.

```text
Next/Firebase style:
component state -> client SDK -> JSON/data snapshots -> frontend reconciliation

Go/Datastar style:
server state -> typed HTML component -> SSE patch -> DOM update
```

Co odpada:

- Brak hydracji Reacta dla prostych interakcji.
- Brak osobnego JSON API dla widokow, ktore i tak koncza jako HTML.
- Brak Firebase Rules jako osobnego jezyka polityk poza aplikacja.
- Brak kosztow zaleznosci od liczby document reads/writes.
- Brak ORM-a ukrywajacego SQL.

Co zostaje:

- Normalny HTTP.
- Normalny SQL.
- Jeden proces Go.
- Jedna binarka w Dockerze.
- Typy od zapytania SQL do komponentu templ.
- Realtime przez standardowe SSE.

## Firebase/Next.js Mapowanie

| Problem | Firebase/Next.js | Ten stack |
| --- | --- | --- |
| Realtime UI | Firestore listeners / client SDK | Datastar SSE patches |
| Routing | Next routes/API routes | Echo handlers |
| UI | React components + hydration | templ server components |
| Validation | Zod + Firebase Rules | Zog in Go |
| Database | Firestore documents | Postgres through pgx |
| Data access | SDK/ORM/query builder | sqlc-generated typed SQL |
| Schema changes | implicit document shape | goose migrations in Git |
| Deployment | Node/serverless/platform runtime | one Go binary/container |

## Lokalny Development

```bash
go mod tidy
go tool templ generate
go test ./...
go run ./cmd/server
```

Po zmianach w `views/*.templ`:

```bash
go tool templ generate
```

Po zmianach SQL:

```bash
go tool sqlc generate
```

## Jak To Pokazac W 2 Minuty

1. Odpal kontener i otworz dwie karty.
2. Kliknij `+` w jednej karcie, pokaz ze druga aktualizuje sie sama.
3. Pokaz `views/page.templ`: przyciski to tylko `data-on:click`.
4. Pokaz `internal/counter/hub.go`: lista subskrybentow i broadcast.
5. Pokaz `queries/counter.sql`: SQL jest zrodlem prawdy dla Postgresa.
6. Pokaz `internal/db`: sqlc generuje typowany kod bez recznego `Scan`.

## Wersje Sprawdzone 2026-04-27

- Go 1.26.2
- Echo v5.1.0
- templ v0.3.1001
- Datastar Go SDK v1.2.1
- Zog v0.22.2
- pgx v5.9.2
- sqlc v1.31.1
- goose v3.27.1
