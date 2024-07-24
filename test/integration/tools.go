//go:build tools
// +build tools

package tools

import (
	_ "github.com/IBM/idemix/tools/idemixgen"
	_ "github.com/anoideaopen/acl"
	_ "github.com/anoideaopen/channel-transfer"
	_ "github.com/anoideaopen/foundation/test/chaincode/cc"
	_ "github.com/anoideaopen/foundation/test/chaincode/fiat"
	_ "github.com/anoideaopen/foundation/test/chaincode/industrial"
	_ "github.com/anoideaopen/robot"
	_ "github.com/hyperledger/fabric/cmd/configtxgen"
	_ "github.com/hyperledger/fabric/cmd/cryptogen"
	_ "github.com/hyperledger/fabric/cmd/discover"
	_ "github.com/hyperledger/fabric/cmd/orderer"
	_ "github.com/hyperledger/fabric/cmd/osnadmin"
	_ "github.com/hyperledger/fabric/cmd/peer"
)
