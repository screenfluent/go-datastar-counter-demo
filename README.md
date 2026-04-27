# Go Realtime Counter dla Kuby

Demo pokazuje maly, SaaS-owy stack Go jako alternatywe dla Firebase/Next.js:

- Echo v5.1: routing HTTP.
- templ: typowane komponenty HTML kompilowane do Go.
- Datastar: realtime UI przez SSE i fragmenty HTML, bez pisania React/useEffect.
- Zog: walidacja akcji licznika.
- pgx + sqlc + goose: opcjonalny Postgres, typowane SQL i migracje.

## Najszybsze odpalenie

```bash
docker build -t go-datastar-counter-demo .
docker run --rm -p 8080:8080 go-datastar-counter-demo
```

Wejdz na `http://localhost:8080`, otworz kilka kart i klikaj `+` albo `-`. Wszystkie karty aktualizuja sie w czasie rzeczywistym przez Server-Sent Events.

Domyslnie aplikacja uzywa pamieci procesu, zeby jeden obraz Dockerowy dzialal bez dodatkowych uslug.

## Tryb z Postgres

Jesli chcesz pokazac caly przeplyw pgx + goose + sqlc, odpal Postgresa i ustaw `DATABASE_URL`:

```bash
docker run --rm --name counter-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=counter \
  -p 5432:5432 \
  postgres:17-alpine

DATABASE_URL='postgres://postgres:postgres@localhost:5432/counter?sslmode=disable' go run ./cmd/server
```

Przy starcie aplikacja wykonuje migracje z `migrations/` przez goose. Kod w `internal/db/` jest wygenerowany przez sqlc z pliku `queries/counter.sql`.

## Lokalny development

```bash
go mod tidy
go tool templ generate
go test ./...
go run ./cmd/server
```

Przy zmianach w `views/*.templ` uruchom ponownie:

```bash
go tool templ generate
```

## Co warto pokazac

Klikniecie przycisku nie robi `fetch().then(json)` i nie aktualizuje lokalnego stanu Reacta. Przycisk ma tylko atrybut Datastar:

```html
data-init="@get('/events', {openWhenHidden: true})"
data-on:click="@post('/counter/increment')"
```

Serwer zmienia stan, renderuje komponent templ i wysyla go po SSE do wszystkich subskrybentow. To jest przeplyw:

```text
button -> Echo handler -> Zog validation -> store -> templ HTML -> Datastar SSE patch
```

W trybie Postgres dochodzi:

```text
goose migration -> sqlc query -> pgx pool -> typed Go struct
```

## Wersje sprawdzone 2026-04-27

- Go 1.26.2
- Echo v5.1.0
- templ v0.3.1001
- Datastar Go SDK v1.2.1
- Zog v0.22.2
- pgx v5.9.2
- sqlc v1.31.1
- goose v3.27.1
