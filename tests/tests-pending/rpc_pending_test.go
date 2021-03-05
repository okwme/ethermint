// This is a test utility for Ethermint's Web3 JSON-RPC services.
//
// To run these tests please first ensure you have the ethermintd running
//
// You can configure the desired HOST and MODE as well in integration-test-all.sh
package pending

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	rpctypes "github.com/cosmos/ethermint/rpc/types"
	util "github.com/cosmos/ethermint/tests"
)

const (
	addrA         = "0xc94770007dda54cF92009BFF0dE90c06F603a09f"
	addrAStoreKey = 0
)

var (
	MODE = os.Getenv("MODE")
	from = []byte{}
)

type Request struct {
	Version string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int         `json:"id"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type Response struct {
	Error  *RPCError       `json:"error"`
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
}

func TestMain(m *testing.M) {
	if MODE != "pending" {
		_, _ = fmt.Fprintln(os.Stdout, "Skipping pending RPC test")
		return
	}

	var err error
	from, err = util.GetAddress()
	if err != nil {
		fmt.Printf("failed to get account: %s\n", err)
		os.Exit(1)
	}

	// Start all tests
	code := m.Run()
	os.Exit(code)
}

func TestEth_Pending_GetBalance(t *testing.T) {
	var res hexutil.Big
	rpcRes := util.Call(t, "eth_getBalance", []string{addrA, "latest"})
	err := res.UnmarshalJSON(rpcRes.Result)
	require.NoError(t, err)
	preTxLatestBalance := res.ToInt()

	rpcRes = util.Call(t, "eth_getBalance", []string{addrA, "pending"})
	err = res.UnmarshalJSON(rpcRes.Result)
	require.NoError(t, err)
	preTxPendingBalance := res.ToInt()

	t.Logf("Got pending balance %s for %s pre tx\n", preTxPendingBalance, addrA)
	t.Logf("Got latest balance %s for %s pre tx\n", preTxLatestBalance, addrA)

	param := make([]map[string]string, 1)
	param[0] = make(map[string]string)
	param[0]["from"] = "0x" + fmt.Sprintf("%x", from)
	param[0]["to"] = addrA
	param[0]["value"] = "0xA"
	param[0]["gasLimit"] = "0x5208"
	param[0]["gasPrice"] = "0x1"

	// TODO: rpc_test_fix: unable to unlock a secp256k1 account
	_ = util.Call(t, "personal_unlockAccount", []interface{}{param[0]["from"], ""})

	rpcRes = util.Call(t, "eth_sendTransaction", param)
	require.Nil(t, rpcRes.Error)

	rpcRes = util.Call(t, "eth_getBalance", []string{addrA, "pending"})
	err = res.UnmarshalJSON(rpcRes.Result)
	require.NoError(t, err)
	postTxPendingBalance := res.ToInt()
	t.Logf("Got pending balance %s for %s post tx\n", postTxPendingBalance, addrA)

	require.Equal(t, preTxPendingBalance.Add(preTxPendingBalance, big.NewInt(10)), postTxPendingBalance)

	rpcRes = util.Call(t, "eth_getBalance", []string{addrA, "latest"})
	err = res.UnmarshalJSON(rpcRes.Result)
	require.NoError(t, err)
	postTxLatestBalance := res.ToInt()
	t.Logf("Got latest balance %s for %s post tx\n", postTxLatestBalance, addrA)

	require.Equal(t, preTxLatestBalance, postTxLatestBalance)
}

func TestEth_Pending_GetTransactionCount(t *testing.T) {
	prePendingNonce := util.GetNonce(t, "pending")
	t.Logf("Pending nonce before tx is %d", prePendingNonce)

	currentNonce := util.GetNonce(t, "latest")
	t.Logf("Current nonce is %d", currentNonce)

	param := make([]map[string]string, 1)
	param[0] = make(map[string]string)
	param[0]["from"] = "0x" + fmt.Sprintf("%x", from)
	param[0]["to"] = addrA
	param[0]["value"] = "0xA"
	param[0]["gasLimit"] = "0x5208"
	param[0]["gasPrice"] = "0x1"

	_ = util.Call(t, "personal_unlockAccount", []interface{}{param[0]["from"], ""}) // TODO: rpc_test_fix: unable to unlock a secp256k1 account

	txRes := util.Call(t, "eth_sendTransaction", param)
	require.Nil(t, txRes.Error)

	pendingNonce := util.GetNonce(t, "pending")
	latestNonce := util.GetNonce(t, "latest")
	t.Logf("Latest nonce is %d", latestNonce)
	require.Equal(t, currentNonce, latestNonce)
	t.Logf("Pending nonce is %d", pendingNonce)
	require.NotEqual(t, latestNonce, pendingNonce)

	require.Greater(t, uint64(pendingNonce), uint64(latestNonce))
	require.Equal(t, uint64(prePendingNonce)+uint64(1), uint64(pendingNonce))
}

func TestEth_Pending_GetBlockTransactionCountByNumber(t *testing.T) {
	rpcRes := util.Call(t, "eth_getBlockTransactionCountByNumber", []interface{}{"pending"})
	var preTxPendingTxCount hexutil.Uint
	err := json.Unmarshal(rpcRes.Result, &preTxPendingTxCount)
	require.NoError(t, err)
	t.Logf("Pre tx pending nonce is %d", preTxPendingTxCount)

	rpcRes = util.Call(t, "eth_getBlockTransactionCountByNumber", []interface{}{"latest"})
	var preTxLatestTxCount hexutil.Uint
	err = json.Unmarshal(rpcRes.Result, &preTxLatestTxCount)
	require.NoError(t, err)
	t.Logf("Pre tx latest nonce is %d", preTxLatestTxCount)

	param := make([]map[string]string, 1)
	param[0] = make(map[string]string)
	param[0]["from"] = "0x" + fmt.Sprintf("%x", from)
	param[0]["to"] = addrA
	param[0]["value"] = "0xA"
	param[0]["gasLimit"] = "0x5208"
	param[0]["gasPrice"] = "0x1"

	// TODO: rpc_test_fix: unable to unlock a secp256k1 account
	_ = util.Call(t, "personal_unlockAccount", []interface{}{param[0]["from"], ""})

	txRes := util.Call(t, "eth_sendTransaction", param)
	require.Nil(t, txRes.Error)

	rpcRes = util.Call(t, "eth_getBlockTransactionCountByNumber", []interface{}{"pending"})
	var postTxPendingTxCount hexutil.Uint
	err = json.Unmarshal(rpcRes.Result, &postTxPendingTxCount)
	require.NoError(t, err)
	t.Logf("Post tx pending nonce is %d", postTxPendingTxCount)

	rpcRes = util.Call(t, "eth_getBlockTransactionCountByNumber", []interface{}{"latest"})
	var postTxLatestTxCount hexutil.Uint
	err = json.Unmarshal(rpcRes.Result, &postTxLatestTxCount)
	require.NoError(t, err)
	t.Logf("Post tx latest nonce is %d", postTxLatestTxCount)

	require.Equal(t, uint64(preTxPendingTxCount)+uint64(1), uint64(postTxPendingTxCount))
	require.NotEqual(t, uint64(postTxPendingTxCount)-uint64(preTxPendingTxCount), uint64(postTxLatestTxCount)-uint64(preTxLatestTxCount))
}

func TestEth_Pending_GetBlockByNumber(t *testing.T) {
	// TODO: rpc_test_fix: bloom filter not found from tendermint
	rpcRes := util.Call(t, "eth_getBlockByNumber", []interface{}{"latest", true})
	var preTxLatestBlock map[string]interface{}
	err := json.Unmarshal(rpcRes.Result, &preTxLatestBlock)
	require.NoError(t, err)
	preTxLatestTxs := len(preTxLatestBlock["transactions"].([]interface{}))

	rpcRes = util.Call(t, "eth_getBlockByNumber", []interface{}{"pending", true})
	var preTxPendingBlock map[string]interface{}
	err = json.Unmarshal(rpcRes.Result, &preTxPendingBlock)
	require.NoError(t, err)
	preTxPendingTxs := len(preTxPendingBlock["transactions"].([]interface{}))

	param := make([]map[string]string, 1)
	param[0] = make(map[string]string)
	param[0]["from"] = "0x" + fmt.Sprintf("%x", from)
	param[0]["to"] = addrA
	param[0]["value"] = "0xA"
	param[0]["gasLimit"] = "0x5208"
	param[0]["gasPrice"] = "0x1"

	txRes := util.Call(t, "eth_sendTransaction", param)
	require.Nil(t, txRes.Error)

	rpcRes = util.Call(t, "eth_getBlockByNumber", []interface{}{"pending", true})
	var postTxPendingBlock map[string]interface{}
	err = json.Unmarshal(rpcRes.Result, &postTxPendingBlock)
	require.NoError(t, err)
	postTxPendingTxs := len(postTxPendingBlock["transactions"].([]interface{}))
	require.Greater(t, postTxPendingTxs, preTxPendingTxs)

	rpcRes = util.Call(t, "eth_getBlockByNumber", []interface{}{"latest", true})
	var postTxLatestBlock map[string]interface{}
	err = json.Unmarshal(rpcRes.Result, &postTxLatestBlock)
	require.NoError(t, err)
	postTxLatestTxs := len(postTxLatestBlock["transactions"].([]interface{}))
	require.Equal(t, preTxLatestTxs, postTxLatestTxs)

	require.Greater(t, postTxPendingTxs, preTxPendingTxs)
}

func TestEth_Pending_GetTransactionByBlockNumberAndIndex(t *testing.T) {
	var pendingTx []*rpctypes.Transaction
	resPendingTxs := util.Call(t, "eth_pendingTransactions", []string{})
	err := json.Unmarshal(resPendingTxs.Result, &pendingTx)
	require.NoError(t, err)
	pendingTxCount := len(pendingTx)

	data := "0x608060405234801561001057600080fd5b5061011e806100206000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c8063bc9c707d14602d575b600080fd5b603360ab565b6040518080602001828103825283818151815260200191508051906020019080838360005b8381101560715780820151818401526020810190506058565b50505050905090810190601f168015609d5780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b60606040518060400160405280600681526020017f617261736b61000000000000000000000000000000000000000000000000000081525090509056fea2646970667358221220a31fa4c1ce0b3651fbf5401c511b483c43570c7de4735b5c3b0ad0db30d2573164736f6c63430007050033"
	param := make([]map[string]string, 1)
	param[0] = make(map[string]string)
	param[0]["from"] = "0x" + fmt.Sprintf("%x", from)
	param[0]["to"] = addrA
	param[0]["value"] = "0xA"
	param[0]["gasLimit"] = "0x5208"
	param[0]["gasPrice"] = "0x1"
	param[0]["data"] = data

	// TODO: rpc_test_fix: unable to unlock a secp256k1 account
	_ = util.Call(t, "personal_unlockAccount", []interface{}{param[0]["from"], ""})

	txRes := util.Call(t, "eth_sendTransaction", param)
	require.Nil(t, txRes.Error)

	rpcRes := util.Call(t, "eth_getTransactionByBlockNumberAndIndex", []interface{}{"pending", "0x" + fmt.Sprintf("%X", pendingTxCount)})
	var pendingBlockTx map[string]interface{}
	err = json.Unmarshal(rpcRes.Result, &pendingBlockTx)
	require.NoError(t, err)

	// verify the pending tx has all the correct fields from the tx sent.
	require.NotEmpty(t, pendingBlockTx["hash"])
	require.Equal(t, pendingBlockTx["value"], "0xa")
	require.Equal(t, data, pendingBlockTx["input"])

	rpcRes = util.Call(t, "eth_getTransactionByBlockNumberAndIndex", []interface{}{"latest", "0x" + fmt.Sprintf("%X", pendingTxCount)})
	var latestBlock map[string]interface{}
	err = json.Unmarshal(rpcRes.Result, &latestBlock)
	require.NoError(t, err)

	// verify the pending trasnaction does not exist in the latest block info.
	require.Empty(t, latestBlock)
}

func TestEth_Pending_GetTransactionByHash(t *testing.T) {
	data := "0x608060405234801561001057600080fd5b5061011e806100206000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c806302eb691b14602d575b600080fd5b603360ab565b6040518080602001828103825283818151815260200191508051906020019080838360005b8381101560715780820151818401526020810190506058565b50505050905090810190601f168015609d5780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b60606040518060400160405280600d81526020017f617261736b61776173686572650000000000000000000000000000000000000081525090509056fea264697066735822122060917c5c2fab8c058a17afa6d3c1d23a7883b918ea3c7157131ea5b396e1aa7564736f6c63430007050033"
	param := make([]map[string]string, 1)
	param[0] = make(map[string]string)
	param[0]["from"] = "0x" + fmt.Sprintf("%x", from)
	param[0]["to"] = addrA
	param[0]["value"] = "0xA"
	param[0]["gasLimit"] = "0x5208"
	param[0]["gasPrice"] = "0x1"
	param[0]["data"] = data

	// TODO: rpc_test_fix: unable to unlock a secp256k1 account
	_ = util.Call(t, "personal_unlockAccount", []interface{}{param[0]["from"], ""})

	txRes := util.Call(t, "eth_sendTransaction", param)
	var txHash common.Hash
	err := txHash.UnmarshalJSON(txRes.Result)
	require.NoError(t, err)

	rpcRes := util.Call(t, "eth_getTransactionByHash", []interface{}{txHash})
	var pendingBlockTx map[string]interface{}
	err = json.Unmarshal(rpcRes.Result, &pendingBlockTx)
	require.NoError(t, err)

	// verify the pending tx has all the correct fields from the tx sent.
	require.NotEmpty(t, pendingBlockTx)
	require.NotEmpty(t, pendingBlockTx["hash"])
	require.Equal(t, pendingBlockTx["value"], "0xa")
	require.Equal(t, pendingBlockTx["input"], data)
}

func TestEth_Pending_SendTransaction_PendingNonce(t *testing.T) {
	currNonce := util.GetNonce(t, "latest")
	param := make([]map[string]string, 1)
	param[0] = make(map[string]string)
	param[0]["from"] = "0x" + fmt.Sprintf("%x", from)
	param[0]["to"] = addrA
	param[0]["value"] = "0xA"
	param[0]["gasLimit"] = "0x5208"
	param[0]["gasPrice"] = "0x1"

	// TODO: rpc_test_fix: unable to unlock a secp256k1 account
	_ = util.Call(t, "personal_unlockAccount", []interface{}{param[0]["from"], ""})

	// first transaction
	txRes1 := util.Call(t, "eth_sendTransaction", param)
	require.Nil(t, txRes1.Error)
	pendingNonce1 := util.GetNonce(t, "pending")
	require.Greater(t, uint64(pendingNonce1), uint64(currNonce))

	// second transaction
	param[0]["to"] = "0x7f0f463c4d57b1bd3e3b79051e6c5ab703e803d9"
	txRes2 := util.Call(t, "eth_sendTransaction", param)
	require.Nil(t, txRes2.Error)
	pendingNonce2 := util.GetNonce(t, "pending")
	require.Greater(t, uint64(pendingNonce2), uint64(currNonce))
	require.Greater(t, uint64(pendingNonce2), uint64(pendingNonce1))

	// third transaction
	param[0]["to"] = "0x7fb24493808b3f10527e3e0870afeb8a953052d2"
	txRes3 := util.Call(t, "eth_sendTransaction", param)
	require.Nil(t, txRes3.Error)
	pendingNonce3 := util.GetNonce(t, "pending")
	require.Greater(t, uint64(pendingNonce3), uint64(currNonce))
	require.Greater(t, uint64(pendingNonce3), uint64(pendingNonce2))
}
