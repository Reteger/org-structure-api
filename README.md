# Organizational Structure API

REST API для управления организационной структурой компании (подразделения и сотрудники).
Стек: Go, net/http, GORM, PostgreSQL, Goose, Docker, Docker Compose.

[![Go Version](https://img.shields.io/badge/Go-1.25.7%2B-brightgreen.svg)](https://golang.org)

##  Содержание
- [Быстрый запуск](#быстрый-запуск)
- [Обзор API](#Обзор-API)
- [Примеры запросов](#Примеры-запросов)


## Быстрый запуск

###  Способ 1: Docker (рекомендуется)
```
git clone https://github.com/Reteger/org-structure-api.git
cd org-structure-api
docker compose up --build
```
Приложение доступно на: http://localhost:8080

##  Обзор API
Подразделения
| Метод | Путь | Описание |
|-------|------|----------|
| POST  | `/departments/` | Создать подразделение |
| GET   | `/departments/{id}?depth=N&include_employees=bool` | Получить дерево |
| PATCH | `/departments/{id}` | Обновить/переместить |
| DELETE| `/departments/{id}?mode=cascade|reassign` | Удалить |
| POST  | `/departments/{id}/employees/` | Добавить сотрудника |

Сотрудники
| Метод | Путь | Описание |
|-------|------|----------|
|POST|/departments/{id}/employees/|Создать сотрудника |

### Параметры
- depth: 1–5 (по умолчанию 1) — глубина вложенности дерева
- include_employees: true/false (по умолчанию true) — включать ли сотрудников в ответ
- mode: cascade или reassign (обязательно для DELETE)
- reassign_to_department_id: ID подразделения для переноса сотрудников (только для mode=reassign)

## Примеры запросов
Создать корневое подразделение
~~~
$body = '{"name": "Engineering", "parent_id": null}'
Invoke-RestMethod -Uri "http://localhost:8080/departments/" `
  -Method POST -ContentType "application/json" -Body $body
~~~
Ответ:
~~~
{
  "id": 1,
  "name": "Engineering",
  "parent_id": null,
  "created_at": "2026-05-30T15:00:00Z"
}
~~~
## Создать дочернее подразделение
~~~
$body = '{"name": "Backend Team", "parent_id": 1}'
Invoke-RestMethod -Uri "http://localhost:8080/departments/" `
  -Method POST -ContentType "application/json" -Body $body
~~~
## Получить дерево подразделений
~~~
Invoke-RestMethod -Uri "http://localhost:8080/departments/1?depth=3&include_employees=true"
~~~
Ответ:
~~~~
{
  "id": 1,
  "name": "Engineering",
  "parent_id": null,
  "created_at": "2026-05-30T15:00:00Z",
  "employees": [...],
  "children": [
    {
      "id": 2,
      "name": "Backend Team",
      "parent_id": 1,
      "employees": [],
      "children": []
    }
  ]
}
~~~~
## Создать сотрудника
~~~
$body = '{
  "full_name": "Ivan Ivanov",
  "position": "Senior Developer",
  "hired_at": "2024-01-15T00:00:00Z"
}'
Invoke-RestMethod -Uri "http://localhost:8080/departments/1/employees/" `
  -Method POST -ContentType "application/json" -Body $body
~~~
## Обновить подразделение
~~~
$body = '{"name": "Backend Engineering"}'
Invoke-RestMethod -Uri "http://localhost:8080/departments/2" `
  -Method PATCH -ContentType "application/json" -Body $body
~~~
## Удалить подразделение cascade
~~~
Invoke-RestMethod -Uri "http://localhost:8080/departments/2?mode=cascade" -Method DELETE
~~~
Ответ: 204 No Content
