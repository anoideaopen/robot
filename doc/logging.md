# Logging

## Уровни логирования

### Батч отработал без ошибок, но внутри батча транзакции выполнились с ошибками

**WARNING** неотработанные транзакции (`batch was executed in hlf`)

### Информация об ошибках в роботе во время обработки транзакций

**ERROR** Информация при ошибках выполнения батча (после `batch was executed in hlf, hlfTxID`)

**WARNING** мисматч (`endorsement mismatch, err:`)

**ERROR** проблемы соединения (в sdk если пир/ордерер выключены) (`robot for [%s] channel finished after cancel context %+v`, `robot for [%s] channel finished with: %+v, repeat after delay`)

**ERROR** таймаут от клиента, от пира (`timeout`)

**WARNING** сплиты батча (`executeBatch: request size exceeded and txs more than 1, will split batch`)

**WARNING** ретраи батча, включая кол-во попыток, кол-во транзакций, свопов, мультисвопов (`retrying execute, attempt: %d, err: %s, batch: %s` и далее)

### Config info

**INFO** Конфигурация робота (`Robot config:`)

**DEBUG** Запустить сбор блоков с канала (`robot for [%s] channel started`)

**DEBUG** Запустить обработку batch / swap на канале (`start collect from %d block in channel %s`)

### collect batch

**DEBUG** Как собирается батч (`collect batch`, `batch collected`)

**DEBUG** Какие транзакции в батче (`batch collected, batch for exec:`)

### batchExecute

**DEBUG** На какие пиры какие отправлены запросы (`targetPeers:`)

**DEBUG** Какой ответ получен с пиров и с ордерера (`response`, `response.Responses[0].ProposalRespons`)

**DEBUG** Статусы выполнения транзакций внутри батча (`tx was executed`)
