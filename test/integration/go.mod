module github.com/anoideaopen/robot/test/integration

go 1.24.0

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
	github.com/anoideaopen/common-component v0.0.6
	github.com/anoideaopen/foundation v0.1.1
	github.com/anoideaopen/foundation/test/integration v0.0.0-20250218101850-4aacf7f846b9
	github.com/anoideaopen/glog v0.0.3
	github.com/anoideaopen/robot v0.1.0
	github.com/hyperledger/fabric v1.4.0-rc1.0.20250212230350-0211a21efb5a
	github.com/hyperledger/fabric-sdk-go v1.0.1-0.20210927191040-3e3a3c6aeec9
	github.com/onsi/ginkgo/v2 v2.22.2
	github.com/onsi/gomega v1.36.2
)

require (
	code.cloudfoundry.org/clock v1.15.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/IBM/idemix v0.0.2-0.20240913182345-72941a5f41cd // indirect
	github.com/IBM/idemix/bccsp/schemes/aries v0.0.0-20240913182345-72941a5f41cd // indirect
	github.com/IBM/idemix/bccsp/schemes/weak-bb v0.0.0-20240913182345-72941a5f41cd // indirect
	github.com/IBM/idemix/bccsp/types v0.0.0-20240913182345-72941a5f41cd // indirect
	github.com/IBM/mathlib v0.0.3-0.20231011094432-44ee0eb539da // indirect
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/VictoriaMetrics/fastcache v1.12.2 // indirect
	github.com/alecthomas/kingpin/v2 v2.4.0 // indirect
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20240626203959-61d1e3462e30 // indirect
	github.com/anoideaopen/acl v0.1.0 // indirect
	github.com/avast/retry-go/v4 v4.6.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.17.0 // indirect
	github.com/btcsuite/btcd/btcutil v1.1.6 // indirect
	github.com/btcsuite/btcutil v1.0.3-0.20201208143702-a53e38424cce // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudflare/cfssl v1.4.1 // indirect
	github.com/consensys/bavard v0.1.22 // indirect
	github.com/consensys/gnark-crypto v0.14.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/crate-crypto/go-ipa v0.0.0-20240724233137-53bbb0ceb27a // indirect
	github.com/crate-crypto/go-kzg-4844 v1.1.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/ddulesov/gogost v1.0.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/docker/docker v27.1.2+incompatible // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.2.1 // indirect
	github.com/ethereum/c-kzg-4844 v1.0.2 // indirect
	github.com/ethereum/go-ethereum v1.15.0 // indirect
	github.com/ethereum/go-verkle v0.2.2 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/fsouza/go-dockerclient v1.12.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/go-kit/kit v0.13.0 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.24.0 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v0.0.5-0.20220116011046-fa5810519dcb // indirect
	github.com/google/certificate-transparency-go v1.0.21 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/pprof v0.0.0-20241210010833-40e02aabc2ad // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.25.1 // indirect
	github.com/holiman/uint256 v1.3.2 // indirect
	github.com/hyperledger-labs/SmartBFT v0.0.0-20241013183757-134292d4208a // indirect
	github.com/hyperledger/aries-bbs-go v0.0.0-20240528084656-761671ea73bc // indirect
	github.com/hyperledger/fabric-amcl v0.0.0-20230602173724-9e02669dceb2 // indirect
	github.com/hyperledger/fabric-chaincode-go/v2 v2.3.0 // indirect
	github.com/hyperledger/fabric-config v0.3.0 // indirect
	github.com/hyperledger/fabric-lib-go v1.1.3-0.20240523144151-25edd1eaf5f5 // indirect
	github.com/hyperledger/fabric-protos-go-apiv2 v0.3.6 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/joho/godotenv v1.4.0 // indirect
	github.com/kilic/bls12-381 v0.1.0 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/miekg/pkcs11 v1.1.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mmcloughlin/addchain v0.4.0 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/patternmatcher v0.6.0 // indirect
	github.com/moby/sys/sequential v0.5.0 // indirect
	github.com/moby/sys/user v0.1.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc6 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.20.5 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.59.1 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/redis/go-redis/v9 v9.7.0 // indirect
	github.com/sagikazarmark/locafero v0.6.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.7.0 // indirect
	github.com/spf13/cobra v1.8.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.20.0-alpha.6 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/supranational/blst v0.3.13 // indirect
	github.com/sykesm/zap-logfmt v0.0.4 // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20220721030215-126854af5e6d // indirect
	github.com/tedsuo/ifrit v0.0.0-20230516164442-7862c310ad26 // indirect
	github.com/weppos/publicsuffix-go v0.5.0 // indirect
	github.com/xhit/go-str2duration/v2 v2.1.0 // indirect
	github.com/zmap/zcrypto v0.0.0-20190729165852-9051775e6a2e // indirect
	github.com/zmap/zlint v0.0.0-20190806154020-fd021b4cfbeb // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.16 // indirect
	go.etcd.io/etcd/pkg/v3 v3.5.16 // indirect
	go.etcd.io/etcd/raft/v3 v3.5.16 // indirect
	go.etcd.io/etcd/server/v3 v3.5.16 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel v1.34.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.34.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.34.0 // indirect
	go.opentelemetry.io/otel/metric v1.34.0 // indirect
	go.opentelemetry.io/otel/sdk v1.34.0 // indirect
	go.opentelemetry.io/otel/trace v1.34.0 // indirect
	go.opentelemetry.io/proto/otlp v1.5.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/crypto v0.35.0 // indirect
	golang.org/x/exp v0.0.0-20240904232852-e7e105dedf7e // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/sync v0.11.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	golang.org/x/tools v0.29.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250115164207-1a7da9e5054f // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250115164207-1a7da9e5054f // indirect
	google.golang.org/grpc v1.70.0 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	rsc.io/tmplfunc v0.0.3 // indirect
)

replace (
	github.com/anoideaopen/robot => ../../
	github.com/hyperledger/fabric-sdk-go => github.com/anoideaopen/fabric-sdk-go v0.1.0
)
