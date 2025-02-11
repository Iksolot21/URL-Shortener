# URL Shortener Service

Реализация сервиса, предоставляющего API по созданию сокращённых ссылок в рамках тестового задания для стажера-разработчика.

## Задача

Реализовать сервис, предоставляющий API по созданию сокращённых ссылок, соответствующих следующим требованиям:

*   Ссылка должна быть:
    *   Уникальной: на один оригинальный URL должна ссылаться только одна сокращенная ссылка.
    *   Длиной 10 символов.
    *   Из символов латинского алфавита в нижнем и верхнем регистре, цифр и символа \_ (подчеркивание).

*   Сервис должен быть написан на Go и предоставлять API через gRPC:
    *   `CreateShortURL`: Сохраняет оригинальный URL в базе данных и возвращает сокращённый URL.
    *   `GetOriginalURL`: Принимает сокращённый URL и возвращает оригинальный URL.

## Архитектура

Сервис имеет многослойную архитектуру, обеспечивающую разделение ответственности и гибкость:

*   **Транспортный слой (gRPC):** Обрабатывает gRPC-запросы, преобразует данные и вызывает сервисный слой.
*   **Сервисный слой (Бизнес-логика):** Реализует бизнес-логику приложения, включая генерацию коротких ссылок и взаимодействие с хранилищем.
*   **Слой хранилища:** Предоставляет интерфейс для доступа к данным, поддерживая in-memory и PostgreSQL хранилища.

## Структура проекта

```
url-shortener/
├── .github/
├── cmd/
│   └── url-shortener/
│       └── main.go         # Entry point
├── config/
│   ├── local.yaml          # Local config
│   └── prod.yaml           # Production config
├── internal/
│   ├── config/             # Configuration loading
│   ├── grpc/               # gRPC server
│   ├── lib/                # Shared libraries
│   │   └── random/         # Random string generator
│   ├── service/            # Business logic
│   └── storage/            # Storage interfaces and implementations
├── tests/                  # Unit tests
├── .gitignore
├── go.mod
├── go.sum
└── Makefile               # Build automation
```

*   **`cmd/url-shortener/main.go`:** Точка входа в приложение. Загружает конфигурацию, инициализирует хранилище и запускает gRPC-сервер.
*   **`config/`:** Файлы конфигурации в формате YAML.
*   **`internal/grpc/`:** Определение gRPC API (`.proto` файл) и реализация gRPC-сервера.
*   **`internal/lib/`:** Общие библиотеки и утилиты.
*   **`internal/service/`:** Реализация бизнес-логики приложения.
*   **`internal/storage/`:** Интерфейсы и реализации для различных типов хранилищ (in-memory, PostgreSQL).
*   **`tests/`:** Юнит-тесты.

## Зависимости

*   `github.com/joho/godotenv`: Для загрузки переменных окружения из файла `.env` (только для локальной разработки).
*   `google.golang.org/grpc`: Для работы с gRPC.
*   `google.golang.org/protobuf`: Для работы с protobuf-сообщениями.
*   `github.com/lib/pq`: Для работы с PostgreSQL.
*   `github.com/ilyakaznacheev/cleanenv`: Для загрузки переменных окружения в структуру

## Сборка и запуск

### Локально

1.  Установите Go ([https://go.dev/dl/](https://go.dev/dl/))
2.  Установите переменные окружения (см. раздел "Конфигурация").
3.  Соберите приложение:

    ```bash
    make build
    ```

4.  Запустите приложение:

    ```bash
    ./url-shortener
    ```

    Для использования PostgreSQL:

    ```bash
    DATABASE_URL="postgres://user:password@host:port/database?sslmode=disable" ./url-shortener
    ```

### С помощью Docker

1.  Установите Docker ([https://www.docker.com/](https://www.docker.com/))
2.  Соберите Docker-образ:

    ```bash
    make docker-build
    ```

3.  Запустите Docker-контейнер:

    ```bash
    make docker-run
    ```

    Для использования PostgreSQL:

    ```bash
    docker run -p 8082:8082 -e STORAGE_TYPE=postgres -e DATABASE_URL="postgres://user:password@host:port/database?sslmode=disable" url-shortener
    ```

## Конфигурация

Сервис использует переменные окружения для конфигурации:

*   `CONFIG_PATH`: Путь к файлу конфигурации (`config/local.yaml` или `config/prod.yaml`). Если не указан, используется `./config/local.yaml`.
*   `DATABASE_URL`: Строка подключения к базе данных PostgreSQL (пример: `"postgres://user:password@host:port/database?sslmode=disable"`). Используется, только если `STORAGE_TYPE` установлено в `postgres`.

В локальной разработке рекомендуется использовать файл `.env` для установки переменных окружения. Пример файла `.env`:

```
DATABASE_URL="postgres://myuser:mypassword@localhost:5432/url_shortener?sslmode=disable"
CONFIG_PATH="./config/local.yaml"
```

## Тестирование

Для запуска юнит-тестов выполните команду:

```bash
make test
```

Перед запуском тестов убедитесь, что:

*   Установлена переменная окружения `TEST_DATABASE_URL` с правильной строкой подключения к тестовой базе данных PostgreSQL.
*   Создана тестовая база данных PostgreSQL с именем, указанным в `TEST_DATABASE_URL`.

## Алгоритм генерации коротких ссылок

Сервис использует следующий алгоритм для генерации коротких ссылок:

1.  Генерация случайной строки длиной 10 символов, используя криптографически стойкий генератор случайных чисел (`crypto/rand`) и алфавит, содержащий символы латинского алфавита в нижнем и верхнем регистре, цифры и символ `_`.
2.  Проверка, не существует ли уже такая короткая ссылка в хранилище.
3.  Если короткая ссылка уже существует, генерируется новая случайная строка. Этот процесс повторяется до тех пор, пока не будет найдена уникальная короткая ссылка (максимальное количество попыток ограничено для предотвращения бесконечного цикла).
4.  Сохранение соответствия между оригинальным URL и короткой ссылкой в хранилище.

## Использование gRPC API

Для взаимодействия с сервисом можно использовать `grpcurl` или любой другой gRPC-клиент.

**Примеры использования `grpcurl`:**

*   **Создание короткой ссылки:**

    ```bash
    grpcurl -plaintext -d '{"original_url": "https://www.example.com"}' localhost:8082 url_shortener.URLShortener.CreateShortURL
    ```

*   **Получение оригинального URL:**

    ```bash
    grpcurl -plaintext -d '{"short_url": "aBcDeFgHiJ"}' localhost:8082 url_shortener.URLShortener.GetOriginalURL
    ```

    Замените `aBcDeFgHiJ` на фактическую короткую ссылку.

## Оценка масштабируемости и долговечности

*   **Масштабируемость:**
    *   Сервис можно масштабировать горизонтально, запуская несколько экземпляров gRPC-сервера за балансировщиком нагрузки.
    *   Для хранения данных рекомендуется использовать PostgreSQL, который обеспечивает хорошую масштабируемость и надежность.
    *   In-memory хранилище подходит только для небольших нагрузок и не рекомендуется для production.
*   **Долговечность:**
    *   При использовании PostgreSQL данные сохраняются на диск и не теряются при перезапуске сервиса.
    *   При использовании in-memory хранилища данные теряются при перезапуске сервиса.

## Общая чистота кода

При разработке сервиса были соблюдены следующие принципы:

*   Разделение ответственности (Separation of Concerns).
*   Принцип единственной ответственности (Single Responsibility Principle).
*   Dependency Inversion Principle.
*   Использовались meaningful названия переменных и функций
*   Стандартизация кода
