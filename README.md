# AntiPlag — мини‑платформа для хранения работ и анализа похожести

Это учебный проект (КПО‑ДЗ) — простой набор микросервисов, который демонстрирует архитектуру с независимыми сервисами, базами данных и шлюзом (gateway). Проект реализует приём метаданных работ (storage), запись результатов анализа (analysis) и фасадный HTTP API (gateway).

Содержание README
- Краткое описание и цель
- Архитектура и компоненты
- Схема БД и init-файлы
- Конфигурация и переменные окружения
- Запуск (локально / Docker Compose)
- HTTP API — примеры запросов/ответов
- Интеграция между сервисами
- Отладка и распространённые проблемы
- Структура проекта
- Предложения по улучшению
- Smoke tests (быстрая проверка)

1. Краткое описание и цель
--------------------------
Цель проекта — показать простую микросервисную систему для приёма работ, хранения и анализа результатов (проверка на плагиат). Работа соответствует заданию из `КПО‑ДЗ - 3.pdf` в корне репозитория: создать сервисы, таблицы в БД и обеспечить их взаимодействие через HTTP.

2. Архитектура и компоненты
---------------------------
- storage — сервис для хранения метаданных работ (cmd/storage, internal/storage). Работает с PostgreSQL (база `antiplag_storage`).
- analysis — сервис для хранения отчётов анализа (cmd/analysis, internal/analysis). Работает с PostgreSQL (база `antiplag_analysis`).
- gateway — фасад (cmd/gateway, internal/gateway): принимает запросы от клиента, вызывает storage и analysis, возвращает комбинированные ответы.

Каждый сервис имеет собственный HTTP API, использует конфиг из `config/local.yaml` и логирует через `slog`.

3. Схема БД и init SQL
----------------------
В папке `init/` находятся SQL-скрипты, которые инициализируют БД при старте контейнера Postgres (docker-compose). Наличие и содержание:
- `init/001_create_databses.sql` — создаёт базы `antiplag_storage` и `antiplag_analysis`.
- `init/002_init_create_works.sql` — создаёт таблицу `works`.

Если нужен файл для `reports`, добавьте в `init/` SQL типа:

```sql
\connect antiplag_analysis;

CREATE TABLE IF NOT EXISTS reports (
    id          SERIAL PRIMARY KEY,
    work_id     INTEGER NOT NULL REFERENCES works(id) ON DELETE CASCADE,
    status      TEXT NOT NULL,
    similarity  NUMERIC,
    details     TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);
```

4. Конфигурация и переменные окружения
--------------------------------------
Конфиг по умолчанию: `config/local.yaml`.
Ключевые поля (`internal/config/config.go`):
- Env — среда (local)
- StoragePath — путь для файлов
- HTTPServer / AnalysisServer — адрес и таймауты
- StorageDB.DSN / AnalysisDB.DSN — DSN для подключения к Postgres
- Gateway.StorageBaseURL / AnalysisBaseURL / Address — адреса для обращения между сервисами и порт gateway

Переменные окружения, которые могут переопределять конфиг:
- CONFIG_PATH — путь к YAML (по умолчанию ./config/local.yaml)
- STORAGE_DB_DSN, ANALYSIS_DB_DSN — альтернативные DSN
- STORAGE_BASE_URL, ANALYSIS_BASE_URL, GATEWAY_ADDRESS — адреса для gateway

В `docker-compose.yaml` сервисы используют DSN, где хост — `db` (имя контейнера). Для доступа с хоста проброшен порт `5440:5432`.

5. Запуск
---------
Требования: Go (рекомендуется 1.20+), Docker (если используете compose).

5.1 Локально (без Docker)

- Подготовьте PostgreSQL локально или используйте docker-compose DB (в этом случае используйте `host.docker.internal` или проброс портов).
- Запуск сервисов:

```zsh
# storage
go run ./cmd/storage

# analysis
go run ./cmd/analysis

# gateway
go run ./cmd/gateway
```

Или собрать бинарники:

```zsh
go build -o bin/storage ./cmd/storage
go build -o bin/analysis ./cmd/analysis
go build -o bin/gateway ./cmd/gateway

./bin/storage &
./bin/analysis &
./bin/gateway &
```

5.2 Через Docker Compose (рекомендуется)

```zsh
docker-compose up --build
# или в фоне
docker-compose up -d --build

# смотреть логи
docker-compose logs -f storage
docker-compose logs -f analysis
docker-compose logs -f gateway
```

Postgres в compose монтирует папку `./init` в `/docker-entrypoint-initdb.d`, поэтому SQL будет применён при первом старте.

6. HTTP API — примеры
---------------------
6.1 Storage
- POST /works — создать работу
  Request JSON:
  ```json
  {"student":"Ivan Ivanov","task":"Homework 1","file_path":"/files/1.pdf"}
  ```
  curl:
  ```zsh
  curl -v -X POST http://localhost:8081/works \
    -H "Content-Type: application/json" \
    -d '{"student":"Ivan","task":"t1","file_path":"/tmp/f1.pdf"}'
  ```

- GET /works/{id}
  ```zsh
  curl -v http://localhost:8081/works/1
  ```

6.2 Analysis
- POST /reports
  Request JSON:
  ```json
  {"work_id":1,"status":"done","similarity":12.5,"details":"Found similar fragments"}
  ```
  curl:
  ```zsh
  curl -v -X POST http://localhost:8069/reports \
    -H "Content-Type: application/json" \
    -d '{"work_id":1,"status":"done","similarity":12.5,"details":"..."}'
  ```

- GET /reports/{id}
- GET /reports/work/{work_id}

6.3 Gateway
- POST /works — создаёт work (storage) и report (analysis) и возвращает оба объекта
  curl:
  ```zsh
  curl -v -X POST http://localhost:8052/works \
    -H "Content-Type: application/json" \
    -d '{"student":"Ivan","task":"t1","file_path":"/tmp/f1.pdf"}'
  ```

- GET /works/{id} — возвращает work и, если есть, связанный report
  ```zsh
  curl -v http://localhost:8052/works/1
  ```

7. Интеграция между сервисами
-----------------------------
- Gateway делает HTTP-вызовы к storage и analysis. В Docker Compose это "http://storage:8081" и "http://analysis:8069".
- При локальном запуске используйте "http://localhost:8081" и "http://localhost:8069".

8. Отладка и частые проблемы
---------------------------
8.1 Сервисы не подключаются к БД
- Проверьте, что Postgres запущен и доступен:
  ```zsh
  pg_isready -h localhost -p 5440
  psql -h localhost -p 5440 -U gleboss -d antiplag_storage -c '\dt'
  ```
- В docker-compose DSN использует host `db`. Если сервисы запускаются на хосте, а БД в Docker — используйте `host.docker.internal` или проброс портов.
- Посмотрите логи контейнера db: `docker-compose logs -f db`.

8.2 Gateway не видит storage/analysis
- Убедитесь, что в `config/local.yaml` и/или переменных окружения заданы правильные `storage_base_url` и `analysis_base_url`.
- Для контейнеров внутри одной сети используйте имена сервисов (storage, analysis).

8.3 CORS и фронтенд
- При тестах через браузер проверьте CORS-заголовки — в коде используются наборы разрешённых origin и заголовков.

8.4 Логи
- Сервисы логируют в stdout через `slog`. Для контейнеров: `docker-compose logs -f <service>`.

9. Структура проекта
--------------------
```
cmd/                # точка входа для каждого сервиса (storage / analysis / gateway)
internal/           # реализация сервисов: storage, analysis, gateway, config, logger
config/local.yaml   # конфиг по умолчанию
init/               # init SQL для postgres
Dockerfile
docker-compose.yaml
go.mod
```

10. Идеи для улучшения
----------------------
- Асинхронный анализ: вставить очередь задач (RabbitMQ / Redis) для анализа, чтобы gateway не блокировался.
- Миграции вместо init SQL: использовать golang-migrate.
- Добавить unit/integration тесты и CI.
- Безопасность: аутентификация/авторизация, защита от перегрузки.

11. Быстрые smoke tests
-----------------------
1) Поднять через docker-compose:
```zsh
docker-compose up -d --build
```
2) Создать работу через gateway (создаст и report):
```zsh
curl -v -X POST http://localhost:8052/works \
  -H "Content-Type: application/json" \
  -d '{"student":"Ivan Ivanov","task":"Task 1","file_path":"/tmp/f1.pdf"}'
```
3) Проверить через storage напрямую:
```zsh
curl -v http://localhost:8081/works/1
```
4) Проверить отчет напрямую:
```zsh
curl -v http://localhost:8069/reports/work/1
```

12. Заключение
--------------
README даёт обзор текущей структуры и инструкции для быстрой проверки проекта локально и в Docker. Если хочешь, могу:
- создать скрипт `smoke_test.sh` с curl-командами;
- добавить отсутствующий SQL для `reports` в `init/`;
- автоматически закоммитить README в репозиторий.

---

Если нужно — добавлю все правки прямо в репозиторий (инициализацию SQL, smoke_test.sh и т.п.).

