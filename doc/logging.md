# Logging

## Logging levels

### Batch worked without errors, but transactions inside the batches were executed with errors

**WARNING** outstanding transactions (`batch was executed in hlf`)

### Information about errors in the robot during transaction processing

**ERROR** Information in case of batches execution errors (after `batch was executed in hlf, hlfTxID`)

**WARNING** mismatch (`endorsement mismatch, err:`)

**ERROR** connection problems (in the sdk if peer/orderer is turned off) (`robot for [%s] channel finished after cancel context %+v`, `robot for [%s] channel finished with: %+v, repeat after delay`)

**ERROR** timeout from client, from peer (`timeout`)

**WARNING** batch splits (`executeBatch: request size exceeded and txs more than 1, will split batch`)

**WARNING** retrai batch, including number of attempts, number of transactions, swaps, multiswaps (`retrying execute, attempt: %d, err: %s, batch: %s` on and on)

### Config info

**INFO** Robot configuration (`Robot config:`)

**DEBUG** Start collecting blocks from the channel (`robot for [%s] channel started`)

**DEBUG** Start batch / swap processing on the channel (`start collect from %d block in channel %s`)

### collect batch

**DEBUG** How the batch is assembled (`collect batch`, `batch collected`)

**DEBUG** What are the transactions in the batch (`batch collected, batch for exec:`)

### batchExecute

**DEBUG** Which peers have requests been sent to (`targetPeers:`)

**DEBUG** What response was received from the peers and from the order (`response`, `response.Responses[0].ProposalRespons`)

**DEBUG** Transaction execution statuses inside the batch (`tx was executed`)
