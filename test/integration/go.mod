module github.com/anoideaopen/robot/test/integration

go 1.26.4

tool (
	github.com/IBM/idemix/tools/idemixgen
	github.com/anoideaopen/acl
	github.com/anoideaopen/foundation/test/chaincode/cc
	github.com/anoideaopen/foundation/test/chaincode/fiat
	github.com/anoideaopen/foundation/test/chaincode/industrial
	github.com/anoideaopen/robot
	github.com/hyperledger/fabric/cmd/configtxgen
	github.com/hyperledger/fabric/cmd/cryptogen
	github.com/hyperledger/fabric/cmd/discover
	github.com/hyperledger/fabric/cmd/orderer
	github.com/hyperledger/fabric/cmd/osnadmin
	github.com/hyperledger/fabric/cmd/peer
)

require (
	github.com/anoideaopen/common-component v0.0.7
	github.com/anoideaopen/foundation v0.1.8
	github.com/anoideaopen/foundation/test/integration v0.0.0-20260618101433-4e4eeb257ffc
	github.com/anoideaopen/glog v0.0.4
	github.com/anoideaopen/robot v0.1.2-0.20260528192439-be92214fb831
	github.com/hyperledger/fabric v1.4.0-rc1.0.20260618074816-86c1172bec37
	github.com/hyperledger/fabric-sdk-go v1.0.1-0.20210927191040-3e3a3c6aeec9
	github.com/onsi/ginkgo/v2 v2.31.0
	github.com/onsi/gomega v1.42.1
)

require (
	code.cloudfoundry.org/clock v1.15.0 // indirect
	github.com/IBM/idemix v0.0.2 // indirect
	github.com/IBM/idemix/bccsp/schemes/aries v0.0.0-20241220065751-dc7206770307 // indirect
	github.com/IBM/idemix/bccsp/schemes/weak-bb v0.0.0-20241220065751-dc7206770307 // indirect
	github.com/IBM/idemix/bccsp/types v0.0.0-20241220065751-dc7206770307 // indirect
	github.com/IBM/mathlib v0.0.3-0.20250709075152-a138079496c3 // indirect
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/ProjectZKM/Ziren/crates/go-runtime/zkvm_runtime v0.0.0-20260607022201-88e0521b82d3 // indirect
	github.com/VictoriaMetrics/fastcache v1.13.3 // indirect
	github.com/alecthomas/kingpin/v2 v2.4.0 // indirect
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20240927000941-0f3dac36c52b // indirect
	github.com/anoideaopen/acl v0.1.4 // indirect
	github.com/avast/retry-go/v4 v4.7.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.24.5 // indirect
	github.com/btcsuite/btcd/btcutil v1.2.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudflare/cfssl v1.6.5 // indirect
	github.com/consensys/gnark-crypto v0.20.1 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/crate-crypto/go-eth-kzg v1.5.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/ddulesov/gogost v1.0.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.1 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-connections v0.7.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.3.3 // indirect
	github.com/ethereum/c-kzg-4844/v2 v2.1.7 // indirect
	github.com/ethereum/go-ethereum v1.17.3 // indirect
	github.com/expr-lang/expr v1.17.8 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	github.com/go-kit/kit v0.13.0 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.6.1 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.30.3 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/certificate-transparency-go v1.1.7 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20260402051712-545e8a4df936 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0 // indirect
	github.com/holiman/uint256 v1.3.2 // indirect
	github.com/hyperledger-labs/SmartBFT v1.0.1 // indirect
	github.com/hyperledger/aries-bbs-go v0.0.0-20240528084656-761671ea73bc // indirect
	github.com/hyperledger/fabric-amcl v0.0.0-20230602173724-9e02669dceb2 // indirect
	github.com/hyperledger/fabric-chaincode-go/v2 v2.3.0 // indirect
	github.com/hyperledger/fabric-config v0.3.1 // indirect
	github.com/hyperledger/fabric-lib-go v1.1.4 // indirect
	github.com/hyperledger/fabric-protos-go-apiv2 v0.3.7 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jmoiron/sqlx v1.3.5 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/kilic/bls12-381 v0.1.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/miekg/pkcs11 v1.1.2 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/moby/api v1.54.2 // indirect
	github.com/moby/moby/client v0.4.1 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.3.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/procfs v0.20.1 // indirect
	github.com/redis/go-redis/v9 v9.20.1 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/spf13/viper v1.21.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/supranational/blst v0.3.16 // indirect
	github.com/sykesm/zap-logfmt v0.0.4 // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20220721030215-126854af5e6d // indirect
	github.com/tedsuo/ifrit v0.0.0-20260418191334-846868129986 // indirect
	github.com/weppos/publicsuffix-go v0.50.3 // indirect
	github.com/xhit/go-str2duration/v2 v2.1.0 // indirect
	github.com/zmap/zcrypto v0.0.0-20230310154051-c8b263fd8300 // indirect
	github.com/zmap/zlint/v3 v3.5.0 // indirect
	go.etcd.io/etcd/api/v3 v3.6.12 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.6.12 // indirect
	go.etcd.io/etcd/pkg/v3 v3.6.12 // indirect
	go.etcd.io/etcd/server/v3 v3.6.12 // indirect
	go.etcd.io/raft/v3 v3.6.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.69.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	go.yaml.in/yaml/v4 v4.0.0-rc.4 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	golang.org/x/tools v0.46.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260615183401-62b3387ff324 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260615183401-62b3387ff324 // indirect
	google.golang.org/grpc v1.81.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/anoideaopen/robot => ../../
	github.com/hyperledger/fabric-sdk-go => github.com/anoideaopen/fabric-sdk-go v0.1.1
)
