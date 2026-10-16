# Тесты

## Что проверяется

### Unit-тесты

`TestOrderServiceAddOrder` — табличный тест с `gomock`-моками репозиториев. Проверяет успешное создание заказа с расчётом цены, пустой адрес доставки и недоступное блюдо.

`TestOrderServiceChangeStatus` — табличный тест переходов статуса: принятие нового заказа, отмена принятого заказа и запрет передачи нового заказа в доставку.

Моки находятся в `tests/mocks`. После изменения интерфейсов репозиториев их можно пересоздать:

```powershell
go generate ./tests/mocks
```

### Integration-тесты

`TestOrderFlowCreatesOrderAndReservesStock` создаёт ресторан и блюда в тестовой PostgreSQL, оформляет заказ и проверяет сумму и уменьшение остатков.

`TestOrderFlowCancellationRestoresStock` создаёт заказ, принимает и отменяет его, затем проверяет статус `cancelled` и возврат остатков.

Данные для сценариев создаются самими тестами. Перед каждым тестом очищается только БД, имя которой содержит `test`.

## Запуск

Unit-тесты:

```powershell
go test ./tests/unit
```

Все обычные тесты проекта. Integration-тесты не входят в эту команду, так как требуют тег `integration` и PostgreSQL:

```powershell
go test ./...
```

Integration-тесты запускаются локально. Нужны локальная PostgreSQL и отдельная БД `mykitchen_test`. Миграции Goose тесты накатывают самостоятельно:

```powershell
$env:TEST_DATABASE_URL = "postgres://postgres:postgres@localhost:5432/mykitchen_test?sslmode=disable"
go test -tags=integration ./tests/integration
```

Не указывай рабочую БД в `TEST_DATABASE_URL`: integration-тесты очищают таблицы перед каждым сценарием.
