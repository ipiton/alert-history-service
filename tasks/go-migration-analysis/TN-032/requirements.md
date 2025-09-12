# TN-032: AlertStorage Interface & PostgreSQL

## 1. Обоснование
Интерфейс для работы с хранилищем алертов и его PostgreSQL реализация.

## 2. Сценарий
Сервисы используют AlertStorage для сохранения и поиска алертов.

## 3. Требования
- AlertStorage интерфейс с CRUD операциями
- PostgreSQL реализация с pgx
- Поддержка фильтрации и pagination
- Оптимизированные индексы
- Connection pooling

## 4. Критерии приёмки
- [ ] Интерфейс определён
- [ ] PostgreSQL adapter реализован
- [ ] Pagination работает
- [ ] Индексы созданы
- [ ] Unit и integration тесты
