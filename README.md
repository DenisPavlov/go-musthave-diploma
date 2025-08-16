# Конфигурация приложения

Настройка приложения. Каждый следующий спосов более приоритетный предыдущего (сделать таблицу)

| конфигурация                            | env                    | флаг          | значение по умолчанию                                                     |
|-----------------------------------------|------------------------|---------------|---------------------------------------------------------------------------|
| хост:порт запуска приложения            | RUN_ADDRESS            | --a=host:port | localhost:8080                                                            |
| хост:порт системы расчета скидок        | ACCRUAL_SYSTEM_ADDRESS | --r=host:port | localhost:8081                                                            |
| окружение. Возможные значения prod, dev | ENV                    | --env         | dev                                                                       |
| адрес подключения к базе данных         | DATABASE_URI           | --d           | postgresql://postgres:postgres@localhost:55432/gophermart?sslmode=disable |


Для генерации моков используется mockery - https://vektra.github.io/mockery/latest/

- `go install github.com/vektra/mockery/v3@v3.5.1`
- `mockery`

# Запуск сервиса расчета скидок
`./cmd/accrual/accrual_linux_amd64 --a=localhost:8081`

# Запуск приложения
`go run cmd/gophermart/main.go`

# go-musthave-diploma-tpl

Шаблон репозитория для индивидуального дипломного проекта курса «Go-разработчик»

# Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без
   префикса `https://`) для создания модуля

# Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m master template https://github.com/yandex-praktikum/go-musthave-diploma-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/master .github
```

Затем добавьте полученные изменения в свой репозиторий.
