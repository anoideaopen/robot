# Robot

## TOC
- [Robot](#-robot)
  - [TOC](#-toc)
  - [Description](#-description)
  - [Architecture](#-architecture)
    - [Robot service](#-robot-service)
    - [Robot service without swaps](#-robot-service-without-swaps)
    - [Robot's components](#-robots-components)
  - [Scaling](#-scaling)
  - [Dependencies](#-dependencies)
  - [Build](#-build)
    - [Go](#-go)
    - [Docker](#-docker)
  - [Configuration yaml file](#-configuration-yaml-file)
    - [Logger configuration](#-logger-configuration)
    - [Robots' configuration](#-robots-configuration)
      - [initBlockNum](#-initblocknum)
        - [Example 1:](#-example-1)
  - [Connection profile](#-connection-profile)
        - [Example 2:](#-example-2)
  - [Run](#-run)
  - [Tests](#-tests)
    - [Unit tests](#-unit-tests)
    - [Integration tests](#-integration-tests)
  - [Autodoc](#-autodoc)
  - [Metrics](#-metrics)
  - [Metrics Detailed Description](#-metrics-detailed-description)
    - [Robot Metrics](#-robot-metrics)
- [GRAFANA Dashboard](#-grafana-dashboard)
    - [Dashboard Sections](#-dashboard-sections)
  - [Links](#-links)
  - [License](#-license)

## Description
Robot is a service that executes swapping and batching on the Atomyze platform #robot#off#offchain#batch#swap#atomic#

------
## Architecture
### Robot service
![Robot service](doc/assets/robot-service.drawio.svg)
### Robot service without swaps
![Robot service without swaps](doc/assets/robot-service-without-swaps.drawio.svg)
### Robot's components
![Robot's components](doc/assets/robot-components.drawio.svg)

**Collector** receives events about new blocks from HLF.\
**Parser** extracts from HLF block only data for the specific channel. \
**ChRobot** orchestrates all other components.\
**Storage** stores numbers of last handled blocks. Used on start and recovery.\
**Batch** collects data whereas it's limits aren't achieved.\
**Executor** decides which HLF peers should receive created batch and send it to them.

------
## Scaling
Robot might be configured as a single service that handles all HLF channels or as multiple services each of them handles its own non-overlapping group of HLF channels.

![Robot scaling](doc/assets/robot-scaling.drawio.svg)

It is possible to have more than one instance of robot service that handles the same HLF channel simultaneously during the deployment.

![Robot deploy](doc/assets/robot-deploy.drawio.svg)

However, it is not recommended configuring robot services that works with the same HLF channels, due to the fact that state storage  uses optimistic lock that would allow only the one ChRobot to send created batch to avoid MVCC conflict.

![Robot conflict](doc/assets/robot-conflict.drawio.svg)

------
## Dependencies
- HLF
- Redis
- Vault (optional)

------
## Build

### Go
```shell
go build -ldflags="-X 'main.AppInfoVer={Version}'"
```
### Docker

To build docker image you need to provide a `REGISTRY_NETRC` build-arg (to fetch go modules from private registry).

```shell
docker build \
  --platform linux/amd64 \
  .
```
Base build image can be overridden with `BUILDER_IMAGE` (default `golang`) and `BUILDER_VERSION` (default `1.18-alpine`). Sometimes it may be useful in
CI environments.

------
## Configuration yaml file
```yaml
# Example

# logger
logLevel: debug # values: error, warning, info, debug, trace
logType: lr-txt # values: std, lr-txt, lr-txt-dev, lr-json, lr-json-dev

# Web server port
# Endpoints:
# /info    - application info
# /metrics - prometheus metrics
# /healthz - liveness probe
# /readyz  - readiness probe
serverPort: 8080

# Fabric
profilePath: /path/to/Fabric/connection.yaml # path to Fabric connection profile
userName: backend                            # Fabric user
useSmartBFT: true                            # Use SmartBFT consensus algorithm or Raft consensus algorithm

# Block parser configuration
txSwapPrefix: swaps                 # prefix of keys in HLF which store tx swaps
txMultiSwapPrefix: multi_swap       # prefix of keys in HLF which store tx multi swaps
txPreimagePrefix: batchTransactions # prefix of keys in HLF which store tx preimages

# Robots configuration
robots:
  - chName: fiat                    # channel for batches
    collectorsBufSize: 1            # buffer size of blockData
    src:                            # sources of transactions, swaps, multiswaps, keys of swaps and keys of multiswaps
      - chName: fiat
        initBlockNum: 1             # block number to start from
    execOpts:                       # robot execute options
      executeTimeout: 0s            # timeout of sending-executing a batch in HLF (duration of batchExecute). If it is empty it is used a value from the defaultRobotExecOpts
      waitCommitAttempts: 1         # number of attempts checking that a batch was committed in HLF. If it is empty it is used a value from the defaultRobotExecOpts
      waitCommitAttemptTimeout: 5s  # timeout of checking that a batch was committed in HLF. If it is empty it is used a value from the defaultRobotExecOpts

# Batch limits
delayAfterChRobotError: 3s  # delay after not unrecoverable channel error before retry run channel miner again
defaultBatchLimits:         # at least one of limits must be filled
  batchBlocksCountLimit: 10 # max blocks count in a batch
  batchLenLimit: 1000       # max number of transactions, swaps, multiswaps, keys of swaps and keys of multiswaps in a batch
  batchSizeLimit: 100000    # max batch size in bytes
  batchTimeoutLimit: 300ms  # max waiting time before generating a batch
  
# Robots execute options
defaultRobotExecOpts:
  executeTimeout: 0s            # default timeout of sending-executing a batch in the HLF (duration of batchExecute)
  waitCommitAttempts: 3         # default number of attempts checking that a batch was committed in the HLF
  waitCommitAttemptTimeout: 15s # default timeout of checking that a batch was committed in the HLF

# Redis configuration
redisStor:
  dbPrefix: robot # Redis db prefix
  addr:           # Redis addresses
    - redis-6379:6379
    - redis-6380:6380
  password: secret # Redis password
  withTLS: true    # enable TLS for communication with Redis
  rootCAs: /path/to/ca1.pem,/path/to/ca2.pem # comma-separated root CA's certificates list for TLS with Redis

# Crypto configuration
cryptoSrc: vault # values: local, vault, google
vaultCryptoSettings: # HashiCorp Vault configuration
  vaultToken: 123                     # access token for Vault
  useRenewableVaultTokens: false
  vaultAddress: http://localhost:8200 # Vault instance address
  vaultAuthPath:                      # to which endpoint to send requests to get a temporary Vault token
  vaultRole:                          # Vault role
  vaultServiceTokenPath:              # path to the token file for accessing the k8s
  vaultNamespace: kv/                 # directory where all crypto materials are located in Vault
  userCert: backend@atomyzeMSP-cert.pem # Fabric user's cert
googleCryptoSettings: # Google KMS configuration
  gcloudProject: # GCloud project ID
  gcloudCreds:   # path to GCloud credentials
  userCert:      # Fabric user's cert

# Prometheus configuration
promMetrics:
  prefix: robot_ # Prometheus prefix
```

### Logger configuration
List of available `logLevel` values:
- error
- warning
- info
- debug
- trace

List of available `logType` values: 
- std
- lr-txt
- lr-txt-dev
- lr-json
- lr-json-dev

### Robots' configuration
#### initBlockNum
Due to the fact that the number of block
from which robot have to start might be in
`the storage (last successfully handled block)` and `config.yaml (robots->src->initBlockNum value)`
in different states
it is important to know how the robot makes choice between them:
- if the value for a channel presents only in the config the robot uses it
- if the values present in both the robot takes the largest

##### Example 1:
config.yaml:
```yaml
robots:
  - chName: fiat
    src:
    - chName: ch1
      initBlockNum: 50
    - chName: ch2
      initBlockNum: 100
    - chName: ch3
      initBlockNum: 0
    - chName: ch4
      initBlockNum: 0   
```
Storage:
```json
{
  "ch1": 75,
  "ch2": 75,
  "ch3": 75
}
```
Result:
```
ch1 starts from the 75 block      # 50 < 75
ch2 starts from the 100 block     # 75 < 100
ch3 starts from the 75 block      # 75 from the storage
ch4 starts from the 0 block       # 0 from config.yaml
```
------

## Connection profile
It is important to set reasonable timeouts in a connection profile
##### Example 2:
connection.yaml:
```yaml
name: basic-network
version: 1.0.0
client:
  organization: Atomyze

  logging:
    level: info

  connection:
    timeout:
      peer:
        endorser: '300'
      orderer: '300'

  peer:
    timeout:
      response: 5s
      connection: 3s
      discovery:
        # Expiry period for discovery service greylist filter
        # The channel client will greylist peers that are found to be offline
        # to prevent re-selecting them in subsequent retries.
        # This interval will define how long a peer is greylisted
        greylistExpiry: 1s
      registrationResponse: 10s
    orderer:
      timeout:
        connection: 3s
        response: 5s
    global:
      timeout:
        query: 5s
        execute: 5s
        resmgmt: 5s
      cache:
        connectionIdle: 30s
        eventServiceIdle: 2m
        channelConfig: 60s
        channelMembership: 10s
        
  credentialStore:
    #...
  tlsCerts:
    #...

channels:
  #...
organizations:
  #...
orderers:
  #...
peers:
  #...
```
------

## Run
```shell
./robot -c=config.yaml
```
or
```shell
export ROBOT_CONFIG="config.yaml" && ./robot
```
or create file ```config.yaml``` next to the robot executable\
or create file ```/etc/config.yaml```

Also, it is possible to [override](https://github.com/spf13/viper#working-with-environment-variables) values from config by env variables with `ROBOT_` prefix
```shell
export ROBOT_REDISSTOR_PASSWORD=123456 &&
export ROBOT_VAULTCRYPTOSETTINGS_VAULTAUTHPATH="v1/auth/kubernetes/login" &&
./robot -c=config.yaml
```

------
## Tests
### Unit tests
```shell
# Run unit tests
go test ./... -short
```
### Integration tests
Setup [testing environment](https://github.com/atomyze-foundation/sandbox)

```shell
# Run integration tests
export ROBOT_TEST_HLF_PROFILE="/path/to/connection.yaml" &&    # Path to the Fabric connection profile
export ROBOT_TEST_HLF_USER="User1" &&                          # Fabric user. Default: User1
export ROBOT_TEST_HLF_CERT="/path/to/cert.pem" &&              # Path to HLF cert. Takes credentialStore from the Fabric connection profile + {hlfUser}@{orgName}-cert.pem
export ROBOT_TEST_HLF_SK="/path/to/msp/keystore/9ac7152_sk" && # Path to HLF secret key. Takes cryptoStore from the Fabric connection profile + "keystore/priv_sk"
export ROBOT_TEST_HLF_FIAT_OWNER_KEY_BASE58CHECK="" &&         # Base58Check from fiat owner secret key
export ROBOT_TEST_HLF_CC_OWNER_KEY_BASE58CHECK=""  &&          # Base58Check from cc owner secret key
export ROBOT_TEST_HLF_INDUSTRIAL_OWNER_KEY_BASE58CHECK=""      # Base58Check from industrial owner secret key
export ROBOT_TEST_HLF_CH_FIAT="" &&                            # Fiat chaincode name
export ROBOT_TEST_HLF_CH_CC="" &&                              # Cc chaincode name
export ROBOT_TEST_HLF_CH_INDUSTRIAL="" &&                      # Industrial chaincode name
export ROBOT_TEST_HLF_CH_NO_CC="" &&                           # Channel without installed chaincodes for testing the robot service behaviour
export ROBOT_TEST_HLF_USE_SMART_BFT="true" &&                  # Use SmartBFT consensus algorithm or Raft consensus algorithm. Default: true.
export ROBOT_TEST_HLF_DO_SWAPS="false" &&                      # Whether run swap test scenarios. Default: false
export ROBOT_TEST_HLF_DO_MSWAPS="false" &&                     # Whether run multiswap test scenarios. Default: false
export ROBOT_TEST_HLF_INDUSTRIAL_GROUP1="" &&                  # Group for multiswap test scenarios
export ROBOT_TEST_HLF_INDUSTRIAL_GROUP2="" &&                  # Group for multiswap test scenarios
export ROBOT_TEST_REDIS_ADDR="127.0.0.1:6379" &&               # Redis address. Default: 127.0.0.1:6379
export ROBOT_TEST_REDIS_PASS="test" &&                         # Redis password. Default: test
go test ./... -p 1
```

------
## Autodoc

[doc/godoc/pkg/github.com/atomyze-foundation/robot/index.html](doc/godoc/pkg/github.com/atomyze-foundation/robot/index.html)

------
## Metrics
Metrics are available at `/metrics`\
The robot service provides these metrics:
- **go default metrics**
- **app_init_duration_seconds**\
  _gauge_, app init duration
- **app_info**\
  _counter_, app info, all sufficient payload is set through labels:\
  Labels:\
  _ver_ - app version (or hash commit) \
  _ver_sdk_fabric_ - sdk fabric version (or hash commit) with which the app was built \
  _build_date_ - build date
- **batches_executed_total**\
  _counter_, amount of executed batches\
  Labels:\
  _robot_ - robot's destination channel\
  _iserr_ - executed with error - true, false
- **tx_executed_total**\
  _counter_, amount of executed transactions (in all executed batches) \
  Labels:\
  _robot_ - robot's destination channel\
  _txtype_ - type of transaction - (tx, swap, mswap, swapkey, mswapkey)
- **batch_execute_duration_seconds**\
  _histogram_, time spent on sending-executing a batch in HLF (duration of batchExecute)\
  Labels:\
  _robot_ - robot's destination channel\
- **batch_size_estimated_diff**\
  _histogram_, relative difference between real and assumed batch size\
  Labels:\
  _robot_ - robot's channel
- **batch_size_bytes**\
  _histogram_, batch size\
  Labels:\
  _robot_ - robot's destination channel
- **batch_size_bytes_total**\
  _counter_, batch size\
  Labels:\
  _robot_ - robot's destination channel
- **ord_reqsize_exceeded_total**\
  _counter_, amount of times request size was exceeded during executeBatch\
  Labels:\
  _robot_ - robot's destination channel\
  _is_first_attempt_ - whether it is a first attempt or not - true, false 
- **src_channel_errors_total**\
  _counter_, amount of times there was an error on creating HLF events source\
  Labels:\
  _robot_ - robot's destination channel\
  _channel_ - robot's source channel\
  _is_first_attempt_ - whether it is a first attempt or not - true, false\
  _is_src_ch_closed_ - whether source channel was closed or not - true, false\
  _is_timeout_ - whether await timeout was reached - true, false
- **batch_tx_count**\
  _histogram_, total amount of transactions, swaps, multiswaps, keys of swaps and keys of multiswaps in a batch\
  Labels:\
  _robot_ - robot's channel
- **batch_collect_duration_seconds** \
  _histogram_, time spent on collecting a batch. Batch collects by asking collectors. Batch is ready when one of the batch limits on the number of transactions, size or timeout occurs\
  Labels:\
  _robot_ - robot's destination channel
- **tx_waiting_process_count**\
  _gauge_, amount of transactions (counts everything that is in a batch - transaction ids, swaps, multiswaps, keys of swaps and keys of multiswaps) awaiting execution when collector added them into a queue\
  Labels:\
  _robot_ - robot's destination channel\
  _channel_ - robot's source channel
- **height_ledger_blocks**\
  _gauge_, ledger's block num where batch was committed. Takes after every _executeBatch_\
  Labels:\
  _robot_ - robot's destination channel\
- **collector_process_block_num**\
  _gauge_, the block number processing by the collector\
  Labels:\
  _channel_ - robot's source channel\
  _robot_ - robot's destination channel
- **block_tx_count**\
  _histogram_, amount of transactions in a block\
  Labels:\
  _robot_ - robot's destination channel\
  _channel_ - robot's source channel
- **started_total**\
  _counter_, amount of times robot was started (it's main cycle) \
  Labels:\
  _robot_ - robot's destination channel
- **stopped_total**\
  _counter_, amount of times robot was stopped (it's main cycle) \
  Labels:\
  _robot_ - robot's destination channel \
  _iserr_ - executed with error - true, false \
  _err_type_ - error type on interaction with external system (HLF, Redis, etc.) In other case it is internal\
  _component_ - robot service's component (executor, collector, storage, etc.)

Most all the metrics are available on robot service start, but some metrics might be measured only during robot service work.\
These metrics are available after a batch was created and successfully sent to HLF:
- batch_execute_duration_seconds
- batch_size_estimated_diff
- batch_size_bytes
- batch_tx_count
- batch_collect_duration_seconds
- block_tx_count


## Metrics Detailed Description

Dashboard and alerting for monitoring the operation of the Robot service are implemented based on the Prometheus and Grafana systems.

The Robot service generates metrics data during its operation.

The metrics data from Robot is sent to Prometheus, where they are stored in the database. Additionally, Prometheus processes the metrics data according to the rules defined for Robot. In case certain situations defined by the rules occur, notifications (alerts) are issued for the corresponding events.

The Grafana system uses the data from the Prometheus database to generate indicators and dashboard graphs in a temporal retrospective manner.

### Robot Metrics

-1- **app_init_duration_seconds**

Type: sensor

Indicator: Application initialization duration, time in seconds required to start the application

-2- **app_info**

Type: counter; application information

Indicators:
- ver - application version (or hash commit)
- ver_sdk_fabric - sdk fabric version (or hash commit)
- build_date - build date

-3- **batches_executed_total**

Type: counter; the number of processed batches broken down by destination channels and execution with or without errors.

Indicators:

- robot - robot destination channel
- iserr - executed with an error - true, false

Description:

- When operations are conducted on the platform, the counter value should increase for at least some destination channels.

-4- **tx_executed_total**

Type: counter; the number of executed transactions (including swaps, multiswaps, swap keys) in all processed batches broken down by destination channels and transaction types.

Indicators:

- robot - robot destination channel
- txtype - transaction type

Description:

- When operations are conducted on the platform, the counter value should increase for at least some destination channels and transaction types.

-5- **batch_execute_duration_seconds**

Type: histogram; time spent on sending-executing the packet in HLF (broken down by channels)

Indicators:

- robot - robot destination channel

-6- **batch_size_estimated_diff**

Type: histogram; relative difference between actual and estimated packet sizes

Indicators:

- robot - robot channel

Description:

- When serializing the packet in protobuf, its size increases. An initial estimate of the increase can be made in two ways. In the first case, serialization is performed and the size after serialization is determined each time a transaction (swap, etc.) is added to the packet. However, this method is quite costly. The second, more economical method is to use a special method in protobuf to estimate the size of the entire packet immediately after serialization. The metric shows the difference between the results of the two methods.

-7- **batch_size_bytes**

Type: histogram; packet size in bytes broken down by destination channels.

Indicators:

- robot - robot destination channel

-8- **batch_size_bytes_total**

Type: counter; the total volume of processed packets at the current moment in time broken down by destination channels (the current sum of values of **batch_size_bytes**).

Indicators:

- robot - robot destination channel

-9- **ord_reqsize_exceeded_total**

Type: counter; the number of request size exceedances during packet execution (batch)

Indicators:

- robot - robot destination channel
- is_first_attempt - whether this is the first attempt or not - true, false

Description:

- The robot collects transactions into batches (batch). The batch is formed either by timeout, or by maximum size, or by the maximum number of transactions, swaps, etc.
- The batch is sent to orderers, which return an rw-set, the size of which is not known in advance. If the size of the rw-set exceeds the maximum allowable fabric size (fabric parameter), an error is returned from the fabric SDK. In this case, the batch is split in half, and the received parts are executed separately.
- With each such split, the counter value increases by one. There may be a situation where several splits are needed to process the batch. For each of them, the counter is incremented.
- The counter is reset upon restarting the robot. A "fluctuating" counter value (increment by one over a long period) is normal. Attention should be paid to a noticeable monotonic increase in value over a short period of time.

-10- **src_channel_errors_total**

Type: counter; the number of errors when creating the source of HLF events

Indicators:

- robot - robot destination channel
- channel - robot's source channel
- is_first_attempt - whether this is the first attempt or not - true, false
- is_src_ch_closed - whether the source channel was closed or not - true, false
- is_timeout - whether a timeout was reached - true, false

Description:

- Components of the "collector" that are part of the robot service subscribe to channels assigned to them to analyze events in the channels, which then ensures the formation of batches. When creating a subscription, collisions are possible (fabric is unavailable or misconfigured, cryptography inconsistencies, network failures, etc.), which prevent the collector from subscribing to events in the channel it services.
- Each unsuccessful subscription attempt increases the value of the metric counter. Each collector has its own counter.
- It is recorded whether the subscription attempt of the respective collector was the first or not.
- Counters are reset when the robot is restarted. A "fluctuating" counter value (increment by one over a long period) is normal. Attention should be paid to a noticeable monotonic increase in value over a short period of time.

-11- **batch_tx_count**

Type: histogram; the total number of transactions, swaps, multiswaps, swap keys, and multiswap keys for each packet broken down by destination channels.

Indicators:

- robot - robot channel

-12- **batch_collect_duration_seconds**

Type: histogram; the time spent on packet formation. Calculated for each packet.

Indicators:

- robot - robot destination channel

Description:

- Packet formation is completed either upon reaching a timeout, reaching the maximum packet size, or reaching the maximum number of transactions for the packet. The time spent on packet formation is recorded.

-13- **tx_waiting_process_count**

Type: sensor; the number of transactions (including transactions, swaps, multiswaps, swap keys, and multiswap keys) waiting for execution

Indicators:
- robot - robot destination channel
- channel - robot's source channel

Description:

- The formed packet is not executed immediately upon completion of formation but after the processing of the previous packet is finished. The number of transactions waiting for execution is recorded in the metric. An increase in the queue of transactions waiting for execution may indicate problems.

-14- **height_ledger_blocks**

Type: sensor; the ledger block number in which the packet was recorded, broken down by destination channels. Formed after each **executeBatch**.

Indicators:
- robot - robot destination channel

Description:

- The block number in the ledger returned by the SDK after sending the packet for execution. The block number values should only increase.

-15- **collector_process_block_num**

Type: sensor; the current block number being processed by the collector during packet formation

Indicators:

- robot - robot destination channel
- channel - robot's source channel

-16- **block_tx_count**

Type: histogram; the number of transactions in the block

Indicators:

- robot - robot destination channel
- channel - robot's source channel

Description:

- The number of transactions in the current block being processed by the collector. An analytical indicator that allows determining, based on a comparison with the **batch_tx_count** values, whether multiple packets are formed from one block or, on the contrary, one packet from multiple blocks.

-17- **started_total**

Type: counter; the number of robot launches (main cycle) broken down by destination channels.

Indicators:

- robot - robot destination channel

Description:

- An increase in the number of robot launches in any channel (channels) may indicate problems.

-18- **stopped_total**

Type: counter; the number of robot stops (main cycle)

Indicators:

- robot - robot destination channel
- iserr - executed with an error - true, false
- err_type - type of error in interaction with an external system (HLF, Redis, etc.)
- component - robot service component (executor, collector, storage, etc.)

Description:

- The indicators allow analyzing the reasons for service stops.

Most metrics are available when the robot service is started, but some metrics can only be measured during the operation of the robot service. These metrics are available after creating a packet and successfully sending it to HLF.

-   batch\_execute\_duration\_seconds
-   batch\_size\_estimated\_diff
-   batch\_size\_bytes
-   batch\_tx\_count
-   batch\_collect\_duration\_seconds
-   block\_tx\_count

# GRAFANA Dashboard

After logging in, you need to select the Default folder from the Dashboards menu (Figure 1) and within it - the NewRobot dashboard.

Figure 1. Dashboards Menu. Selecting the NewRobot dashboard.

After selecting the NewRobot dashboard, you will be taken to the dashboard window (Figure 2).

Figure 2. Overview of the NewRobot Dashboard

In the upper right corner of the dashboard, there is a control block (Figure 3) that provides content refresh for the dashboard (two circular arrows) and allows you to set the update interval for the dashboard (right menu) and the data display period (left menu).

Figure 3. Data Display Mode Control Block

### Dashboard Sections

Figure 4. Dashboard Sections, Fragment 1

The dashboard sections are described according to their arrangement (Figure 3) from left to right, moving down the rows. The overall view of the dashboard in Figure 3 is divided into three enlarged fragments in Figures 4 - 6. Each section is associated with a figure depicting the section and a number corresponding to the metric in the "Robot Metrics" section.

**App Init Duration Seconds**

Figure: 4

Metric No: 1

Description: Duration of Robot initialization in seconds

**App Info**

Figure: 4

Metric No: 2

Description: Information about the Robot service, including ver - Robot service version, ver_sdk_fabrik - SDK Fabric version

**Height Ledger Blocks**

Figure: 4

Metric No: 14

Description: For channels processed by the Robot service - the block number of the blockchain in which the last package was successfully processed (in Figure 4 pairs: ct - 0, dc - 0, dcdac - 106282, dcgmk - 0, dcrsb - 0, fiat - 0, hermitage - 640, ...)

**Started**

Figure: 4

Metric No: 17

Description: The number of Robot subsystem launches for the channels processed by the service (in Figure 4 pairs: ct - 1, dc - 1, dcdac - 2, dcgmk - 1, dcrsb - 1, fiat - 1, hermitage - 1, ...)

**Stopped without errors**

Figure: 4

Metric No: 18

Description: The number of Robot subsystem stops without errors (iserr=false) for the channels processed by the service (in Figure 4 pairs: ct - 0, dc - 0, dcdac - 0, ...)

**Stopped with errors**

Figure: 4

Metric No: 18

Description: The number of Robot subsystem stops with errors (iserr=true) for the channels processed by the service (in Figure 4 pairs: ct - 0, dc - 0, dcdac - 0, ...)

**Amount of executed transactions (in all executed batches)**

Figure: 4

Metric No: 4

Description: Average number of executed transactions over a time interval (in all executed batches)

**Collector process block num**

Figure: 4

Metric No: 15

Description: Average number of blocks processed by the collector over a time interval (in all executed batches)

Figure 5. Dashboard Sections, Fragment 2

The sections described below represent data in the form of histograms/graphs depending on time. To obtain precise parameter values for a given moment in time for all such representations, you need to place the mouse cursor at a specific point on the view. This will bring up a form with precise parameter values for that moment in time. An example of such clarification is provided below for the **Block Tx Count 95 percentile** section (Figure 6).

**Block Tx Count 95 percentile**

Figure: 5

Metric No: 16

Description: The 95th percentile value of the number of transactions in a block for channels processed by the Robot service

Comment: When you hover the mouse cursor over the histogram, a form with precise values for that moment in time appears (Figure 6).

Figure 6. Determining precise values for channels on the **Block Tx Count 95 percentile** histogram

**TX waiting process count**

Figure: 5

Metric No: 13

Description: The number of transactions (including everything in the package - transaction IDs, swaps, multi-swaps, swap keys, and multi-usage keys) awaiting execution after being added to the queue by the collector (robot - destination channel, channel - source channel)

**Batch size total**

Figure: 5

Metric No: 8

Description: Packet sizes by destination channels

**Batches executed total**

Figure: 5

Metric No: 3

Description: The average number of processed batches by destination channels without errors, iserr=false, over intervals

**Batches executed errors**

Figure: 5

Metric No: 3

Description: The average number of processed batches by destination channels with errors, iserr=true, over intervals

Figure 7. Dashboard Sections, Fragment 3

**Batch Collect Duration Seconds 95 percentile**

Figure: 7

Metric No: 12

Description: The 95th percentile value of the time spent on collecting a package. The package is assembled by polling collectors. The package is ready when one of the package limits on the number of transactions, size, or timeout is reached.

**Batch execute duration seconds 95 percentile**

Figure: 7

Metric No: 5

Description: The 95th percentile value of the time spent on sending-executing a package in HLF

**Batch Size 95 percentile**

Figure: 7

Metric No: 7

Description: The 95th percentile value of the package size for a given Robot destination channel

**Batch Size Estimated Diff 95 percentile**

Figure: 7

Metric No: 6

Description: The 95th percentile value of the relative difference between the actual and estimated package sizes for a given Robot destination channel

**Batch Size TX count 95 percentile**

Figure: 7

Metric No: 11

Description: The 95th percentile value of the total number of transactions, swaps, multi-swaps, swap keys, and multi-usage keys in a package for a given Robot destination channel

## Links

* No

## License
[Default license](LICENSE)