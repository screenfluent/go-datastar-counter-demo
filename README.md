# Go Datastar Postgres Realtime Counter

Minimalne demo stacku do budowania lekkich SaaS-ow w Go z Postgres jako jedynym zrodlem stanu:

- Echo v5.1 jako prosty HTTP router.
- templ jako typowane komponenty HTML kompilowane do Go.
- Datastar jako cienka warstwa realtime przez SSE i HTML patches.
- Zog jako walidacja wejscia po stronie serwera.
- Postgres 18 + pgx + sqlc + goose jako trwala warstwa danych.

Demo celowo nie ma Reacta, bundlera frontendu, JSON API ani Firebase SDK. Stan jest w Postgresie, a UI jest tylko jego projekcja.

## Quick Start

```bash
docker compose up --build
```

Otworz `http://localhost:8080` w dwoch kartach lub dwoch przegladarkach. Klikniecie `+` albo `-` w jednej karcie aktualizuje wszystkie pozostale przez Server-Sent Events.

`docker-compose.yml` uruchamia dwie uslugi:

```text
+----------------------+        +----------------------+
| app                  |        | postgres             |
| Go + Echo + Datastar | -----> | Postgres 18          |
| :8080                |        | persistent volume    |
+----------------------+        +----------------------+
```

## Co Demo Pokazuje

To jest maly, konkretny odpowiednik "Firebase realtime counter", ale bez vendor lock-in i bez przenoszenia logiki biznesowej do przegladarki.

```text
+----------------+        +---------------------------+        +----------------+
| Browser A      |        | Go process                |        | Browser B      |
|                |        |                           |        |                |
| click "+"      +------->| POST /counter/increment   |        | open SSE       |
|                |        | Echo handler              |        | GET /events    |
| open SSE       |        | Zog validation            |        |                |
| GET /events    |        | pgx/sqlc Postgres update  |        |                |
|                |        | Hub broadcast             |        |                |
| DOM patch      |<-------+ templ renders #counter    +------->| DOM patch      |
+----------------+        | Datastar SSE event        |        +----------------+
                          +---------------------------+
```

Najwazniejsza idea:

```text
+---------+-------------------------------+
| state   | Postgres                      |
| UI      | rendered HTML                 |
| change  | POST action                   |
| sync    | SSE patch to every client     |
+---------+-------------------------------+
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

Po kliknieciu handler Echo waliduje akcje przez Zog, aktualizuje licznik w Postgresie przez sqlc/pgx, a potem `Hub` rozsyla nowy `Snapshot` do wszystkich kanalow SSE. Datastar odbiera gotowy fragment HTML i podmienia element `#counter-card` w kazdej przegladarce.

## Architektura

```text
.
|-- cmd/
|   `-- server/
|       `-- main.go          HTTP routes, startup, required DATABASE_URL
|-- internal/
|   |-- counter/
|   |   |-- counter.go       shared Snapshot model
|   |   `-- hub.go           subscriptions, broadcast, realtime fan-out
|   |-- db/
|   |   |-- counter.sql.go   sqlc-generated typed counter queries
|   |   |-- db.go            sqlc DBTX/Queries plumbing
|   |   `-- models.go        sqlc-generated models
|   |-- store/
|   |   `-- postgres.go      pgx/sqlc-backed counter store
|   `-- validate/
|       `-- counter.go       Zog validation for counter actions
|-- migrations/
|   `-- *.sql                goose migrations applied at startup
|-- queries/
|   `-- counter.sql          source SQL used by sqlc
|-- views/
|   |-- page.templ           templ source components
|   `-- page_templ.go        generated Go renderer
|-- static/
|   `-- app.css              visual layer
|-- Dockerfile               app image only
|-- docker-compose.yml       app + Postgres 18
|-- Makefile
|-- go.mod
|-- go.sum
`-- sqlc.yaml
```

Warstwy:

```text
+-------------+
| HTTP action |
+------+------+
       |
       v
+-------------+
| Echo route  |
+------+------+
       |
       v
+-------------+
| Zog         |
| validation  |
+------+------+
       |
       v
+-------------+
| sqlc + pgx  |
| Postgres    |
+------+------+
       |
       v
+-------------+
| Hub         |
| broadcast   |
+------+------+
       |
       v
+-------------+
| templ       |
| render      |
+------+------+
       |
       v
+-------------+
| Datastar    |
| SSE patch   |
+-------------+
```

## Dlaczego Dwa Kontenery

Postgresa nie warto wciskac do tego samego obrazu co aplikacje.

```text
+----------------------+      +-------------------------+
| app container        |      | postgres container      |
| one Go binary        | ---> | official postgres:18    |
| stateless runtime    |      | volume-backed database  |
+----------------------+      +-------------------------+
```

To jest czystsze, bo:

- app image zawiera tylko aplikacje,
- Postgres uzywa oficjalnego obrazu i wlasnego volume,
- `depends_on.healthcheck` pilnuje startu aplikacji po gotowosci bazy,
- produkcyjnie taki podzial mapuje sie bezposrednio na Compose, Fly, Railway, Kubernetes albo zwykly VPS.

## Baza i Migracje

Przy starcie aplikacja wykonuje:

```text
+--------------------+
| connect DATABASE_URL |
+----------+---------+
           |
           v
+--------------------+
| goose.Up(migrations) |
+----------+---------+
           |
           v
+--------------------+
| db.New(pgx pool)     |
+--------------------+
```

Tabela licznika jest tworzona przez `migrations/20260427160000_create_counter.sql`.

Zapytania z `queries/counter.sql` generuja typowany kod w `internal/db`:

```bash
go tool sqlc generate
```

## Dlaczego Ten Stack Jest Ciekawy

Dla osoby przyzwyczajonej do Next.js/Firebase najwieksza roznica to miejsce trzymania stanu i ilosc warstw.

```text
+--------------------+     +------------+     +---------------------+     +-------------------------+
| component state    | --> | client SDK | --> | JSON/data snapshots | --> | frontend reconciliation |
+--------------------+     +------------+     +---------------------+     +-------------------------+

+----------------+     +----------------------+     +-----------+     +------------+
| Postgres state | --> | typed HTML component | --> | SSE patch | --> | DOM update |
+----------------+     +----------------------+     +-----------+     +------------+
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
- Jeden proces Go dla aplikacji.
- Oficjalny Postgres 18 jako trwala baza.
- Typy od zapytania SQL do komponentu templ.
- Realtime przez standardowe SSE.

## Firebase/Next.js Mapowanie

| Problem | Firebase/Next.js | Ten stack |
| --- | --- | --- |
| Realtime UI | Firestore listeners / client SDK | Datastar SSE patches |
| Routing | Next routes/API routes | Echo handlers |
| UI | React components + hydration | templ server components |
| Validation | Zod + Firebase Rules | Zog in Go |
| Database | Firestore documents | Postgres 18 through pgx |
| Data access | SDK/ORM/query builder | sqlc-generated typed SQL |
| Schema changes | implicit document shape | goose migrations in Git |
| Deployment | Node/serverless/platform runtime | Go app container + Postgres container |

## Lokalny Development

```bash
docker compose up --build
```

W drugim terminalu:

```bash
go tool templ generate
go test ./...
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

1. Odpal `docker compose up --build`.
2. Otworz dwie karty.
3. Kliknij `+` w jednej karcie, pokaz ze druga aktualizuje sie sama.
4. Zrestartuj sam kontener app i pokaz, ze licznik zostaje w Postgresie.
5. Pokaz `views/page.templ`: przyciski to tylko `data-on:click`.
6. Pokaz `queries/counter.sql`: SQL jest zrodlem prawdy.
7. Pokaz `internal/db`: sqlc generuje typowany kod bez recznego `Scan`.

## Wersje Sprawdzone 2026-04-27

- Go 1.26.2
- Echo v5.1.0
- templ v0.3.1001
- Datastar Go SDK v1.2.1
- Zog v0.22.2
- pgx v5.9.2
- sqlc v1.31.1
- goose v3.27.1
- Postgres 18
