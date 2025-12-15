
Содержание README
- Архитектура и компоненты
- Схема БД и init-файлы
- Конфигурация и переменные окружения
- Запуск (локально / Docker Compose)
- HTTP API — примеры запросов/ответов
- Структура проекта


# 1. Архитектура и компоненты
---------------------------
- storage — сервис для хранения метаданных работ (cmd/storage, internal/storage). Работает с PostgreSQL (база `antiplag_storage`).
- analysis — сервис для хранения отчётов анализа (cmd/analysis, internal/analysis). Работает с PostgreSQL (база `antiplag_analysis`).
- gateway — фасад (cmd/gateway, internal/gateway): принимает запросы от клиента, вызывает storage и analysis, возвращает комбинированные ответы.

Каждый сервис имеет собственный HTTP API, использует конфиг из `config/local.yaml` и логирует через `slog`.

# 2. Схема БД и init SQL
----------------------
В папке `init/` находятся SQL-скрипты, которые инициализируют БД при старте контейнера Postgres (docker-compose). Наличие и содержание:
- `init/001_create_databses.sql` — создаёт базы `antiplag_storage` и `antiplag_analysis`.
- `init/002_init_create_works.sql` — создаёт таблицу `works`.
- `init/003_init_create_reports.sql` - создает таблицу `reports`

# 3. Конфигурация и переменные окружения
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

# 4. Запуск
---------

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

# 5. HTTP API — примеры
---------------------
5.1 Storage
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

 Analysis
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

5.3 Gateway
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



# 6. Структура проекта
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



