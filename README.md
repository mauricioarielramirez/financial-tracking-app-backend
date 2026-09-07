# financial-tracking-app-backend

API backend para seguimiento de finanzas personales. Reemplaza el
relevamiento manual que hoy se hace en una planilla de Google Sheets:
cuentas/activos, snapshots mensuales de saldos, cotizaciones, distribución
porcentual, posición consolidada e ingresos/gastos/ahorro con acumulado
histórico.

Ver `Requerimientos_Funcionales.md` y `Plan_Tecnico.md` (carpeta padre del
repo) para el detalle completo de alcance y decisiones de diseño.

## Estado actual

Primer artefacto ejecutable del proyecto: arquitectura en capas cableada de
punta a punta (handler → usecase → mapper/repository → SQLite) sobre el
recurso `accounts` (RF-01 a RF-04), más el esqueleto de configuración,
migraciones y clientes de cotización externa (RF-06, RF-10, RF-11) ya
funcionando. Pensado para recorrerse en el editor y probarse a mano con
curl/Postman antes de seguir con el resto de los recursos (`snapshots`,
`income-statements`, `reports/*`).

```
/cmd
  /api               → main.go, arranque del servidor HTTP
/internal
  /domain            → entidades y reglas de negocio puras (sin tags json/db)
  /usecase
    /account         → usecase.go (orquestador) + dto.go (Input/Output de la
                        API) + mapper.go (interfaz Mapper, domain ⇄ DTO)
    /quote           → usecase.go + dto.go (sin mapper: no hay domain de por
                        medio, las cotizaciones ya llegan como decimal)
  /service           → lógica de negocio pura sin I/O (vacío por ahora;
                        aparece con `snapshots`: % de distribución, posición
                        consolidada, ahorro acumulado)
  /repository
    repository.go     → interfaces (repository pattern)
    /sqlite             → implementación concreta
    /mocks              → generado con mockgen (ver más abajo)
  /http
    /handlers          → HTTP ⇄ DTO del usecase correspondiente; nunca ve domain
    /middleware        → logging, recover, CORS, API key opcional
  /quotes              → clientes DolarAPI / CoinGecko + interfaz QuoteProvider
  /config              → carga de variables de entorno (.env)
/migrations            → esquema versionado (NNNN_descripcion.up.sql / .down.sql)
```

Capas, de afuera hacia adentro: el **handler** sólo decodifica/codifica
JSON contra los DTO del usecase. El **usecase** (`account.UseCase`,
`quote.UseCase`) es el orquestador: recibe el DTO de entrada, usa el
**mapper** para convertirlo a `domain`, aplica las reglas de negocio
(`domain.Account.Validate()`, o un `service` de cálculo puro cuando
corresponda), llama al **repository**, y devuelve el DTO de salida armado
por el mapper. `domain` queda en el centro sin saber nada de HTTP, SQL ni
JSON — es la interfaz `Mapper` (una por agregado, en `usecase/<agregado>`)
la que traduce en los dos sentidos, y por ser interfaz se puede reemplazar
por un mock en los tests del usecase sin ejecutar mapeo real.

## Cómo correrlo

Requisitos: Go 1.22+ y un compilador de C (ver nota sobre SQLite más abajo).

```bash
cp .env.example .env      # ajustar si hace falta
go mod tidy                # descarga las dependencias (necesita red)
go run ./cmd/api
```

Al arrancar aplica automáticamente las migraciones pendientes contra el
archivo SQLite indicado en `DB_DSN` (por defecto `./finance.db`, se crea
solo si no existe).

Probar rápido:

```bash
curl http://localhost:8080/healthz

curl -X POST http://localhost:8080/api/v1/accounts \
  -H "Content-Type: application/json" \
  -d '{"name":"Cuenta ARS","type":"bank","currency":"ARS","is_invested":false}'

curl http://localhost:8080/api/v1/accounts

curl http://localhost:8080/api/v1/quotes/suggested
```

### Nota sobre el driver de SQLite (CGO)

Se usa `github.com/mattn/go-sqlite3`, que requiere CGO habilitado y un
compilador de C instalado (`CGO_ENABLED=1`, que es el default de Go cuando
detecta un compilador disponible):

- **Windows**: instalar un toolchain mingw-w64 (por ejemplo vía
  [MSYS2](https://www.msys2.org/) — `pacman -S mingw-w64-ucrt-x86_64-gcc` —
  o [TDM-GCC](https://jmeubank.github.io/tdm-gcc/)) y asegurarse de que
  `gcc` esté en el `PATH`.
- **macOS/Linux**: normalmente ya viene un compilador (Xcode Command Line
  Tools / `build-essential`).

Se eligió este driver (en vez de una alternativa 100% Go, como estaba
previsto originalmente en `Plan_Tecnico.md` con `modernc.org/sqlite`) porque
el entorno donde se generó este primer artefacto tenía acceso de red
restringido a `proxy.golang.org` y a los hosts de los que dependen los
drivers pure-Go habituales. Si se prefiere evitar CGO más adelante, alcanza
con reemplazar `internal/repository/sqlite` por una implementación
equivalente sobre otro driver: las interfaces de `internal/repository` no
cambian (ver comentario al inicio de `internal/repository/sqlite/db.go`).

### Migraciones

El runner de migraciones (`internal/repository/sqlite/db.go`) es una
implementación mínima propia (no `golang-migrate`), por la misma razón de
red mencionada arriba: lee los archivos `migrations/*.up.sql` en orden y
registra lo aplicado en una tabla `schema_migrations`. Sigue la misma
convención de nombres que `golang-migrate`, así que adoptarlo más adelante
(por ejemplo al migrar a Postgres) no requiere reescribir las migraciones.

### Tests y mocks (gomock)

`Mapper` (por usecase) y las interfaces de `internal/repository` están
pensadas para reemplazarse por mocks en los tests, generados con
[gomock](https://github.com/uber-go/mock). La librería de runtime
(`go.uber.org/mock`) ya está resuelta en `go.mod`/`go.sum` — incluye un
`replace` hacia `github.com/uber-go/mock`, que apunta al mismo código pero
evita el import vanity; es inofensivo, no hace falta tocarlo aunque tengas
red normal a `go.uber.org`.

Lo único que falta (no se pudo resolver en el entorno donde se generó este
artefacto) es instalar `mockgen`, el generador de código, una sola vez:

```bash
go install go.uber.org/mock/mockgen@v0.6.0
go generate ./...
```

Eso crea `internal/repository/mocks/repository_mock.go` (mocks de
`AccountRepository`, etc., compartidos por cualquier usecase que los
necesite) y `internal/usecase/account/mapper_mock_test.go` (mock de
`Mapper`, sólo para los tests de ese paquete). Hasta que corras esos dos
comandos, `internal/usecase/account/usecase_test.go` no compila — es
esperado, queda como referencia del diseño (ver el comentario al inicio del
archivo). El resto del proyecto compila y corre sin este paso.

Cada usecase nuevo que sume un `Mapper` propio repite el mismo patrón: la
directiva `//go:generate mockgen -source=mapper.go ...` va arriba del
`package` en su propio `mapper.go`.

## Próximos pasos

Siguiendo el mismo patrón `handler → usecase → service/repository` usado en
`accounts`:

1. `snapshots` + `account_balances` + `exchange_rates` (RF-05 a RF-16):
   el recurso central, con el cálculo de `balance_ars`, posición
   consolidada y distribución porcentual.
2. `income-statements` (RF-17 a RF-20), con el recálculo en cascada del
   ahorro acumulado.
3. `reports/*` (RF-21 a RF-25).
4. `cmd/import`: importación masiva desde la planilla de Google Sheets
   (RF-09, Plan Técnico sección 8).
