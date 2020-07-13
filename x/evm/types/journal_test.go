package types

import (
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	abci "github.com/tendermint/tendermint/abci/types"
	tmlog "github.com/tendermint/tendermint/libs/log"
	tmdb "github.com/tendermint/tm-db"

	sdkcodec "github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authclient "github.com/cosmos/cosmos-sdk/x/auth/client"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/params"

	ethcmn "github.com/ethereum/go-ethereum/common"

	"github.com/cosmos/ethermint/codec"
	"github.com/cosmos/ethermint/crypto"
	"github.com/cosmos/ethermint/types"
)

type JournalTestSuite struct {
	suite.Suite

	address ethcmn.Address
	journal *journal
	ctx     sdk.Context
	stateDB *CommitStateDB
}

func newTestCodec() *sdkcodec.Codec {
	cdc := sdkcodec.New()

	RegisterCodec(cdc)
	sdk.RegisterCodec(cdc)
	types.RegisterCodec(cdc)
	auth.RegisterCodec(cdc)
	bank.RegisterCodec(cdc)
	crypto.RegisterCodec(cdc)
	sdkcodec.RegisterCrypto(cdc)

	return cdc
}

func (suite *JournalTestSuite) SetupTest() {
	suite.setup()

	privkey, err := crypto.GenerateKey()
	suite.Require().NoError(err)

	suite.address = ethcmn.BytesToAddress(privkey.PubKey().Address().Bytes())
	suite.journal = newJournal()

	// acc := types.EthAccount{
	// 	BaseAccount: auth.NewBaseAccount(sdk.AccAddress(suite.address.Bytes()), nil, 0, 0),
	// 	CodeHash:    ethcrypto.Keccak256(nil),
	// }

	// suite.stateDB.accountKeeper.SetAccount(suite.ctx, acc)
	suite.stateDB.bankKeeper.SetBalance(suite.ctx, sdk.AccAddress(suite.address.Bytes()), sdk.NewCoin(types.DenomDefault, sdk.NewInt(100)))
}

// setup performs a manual setup of the GoLevelDB and mounts the required IAVL stores. We use the manual
// setup here instead of the Ethermint app test setup because the journal methods are private and using
// the latter would result in a cycle dependency. We also want to avoid declaring the journal methods public
// to maintain consistency with the Geth implementation.
func (suite *JournalTestSuite) setup() {
	authKey := sdk.NewKVStoreKey(auth.StoreKey)
	bankKey := sdk.NewKVStoreKey(bank.StoreKey)
	storeKey := sdk.NewKVStoreKey(StoreKey)

	db := tmdb.NewDB("state", tmdb.GoLevelDBBackend, "temp")
	defer func() {
		os.RemoveAll("temp")
	}()

	cms := store.NewCommitMultiStore(db)
	cms.MountStoreWithDB(authKey, sdk.StoreTypeIAVL, db)
	cms.MountStoreWithDB(bankKey, sdk.StoreTypeIAVL, db)
	cms.MountStoreWithDB(storeKey, sdk.StoreTypeIAVL, db)

	// load latest version (root)
	err := cms.LoadLatestVersion()
	suite.Require().NoError(err)

	cdc := newTestCodec()
	appCodec := codec.NewAppCodec(cdc)
	authclient.Codec = appCodec

	keyParams := sdk.NewKVStoreKey(params.StoreKey)
	tkeyParams := sdk.NewTransientStoreKey(params.TStoreKey)
	paramsKeeper := params.NewKeeper(appCodec, keyParams, tkeyParams)
	// Set specific supspaces
	authSubspace := paramsKeeper.Subspace(auth.DefaultParamspace)
	bankSubspace := paramsKeeper.Subspace(bank.DefaultParamspace)
	ak := auth.NewAccountKeeper(appCodec, authKey, authSubspace, types.ProtoAccount)
	bk := bank.NewBaseKeeper(appCodec, bankKey, ak, bankSubspace, nil)

	suite.ctx = sdk.NewContext(cms, abci.Header{ChainID: "8"}, false, tmlog.NewNopLogger())
	suite.stateDB = NewCommitStateDB(suite.ctx, storeKey, ak, bk)
}

func TestJournalTestSuite(t *testing.T) {
	suite.Run(t, new(JournalTestSuite))
}

func (suite *JournalTestSuite) TestJournal_append_revert() {
	testCases := []struct {
		name  string
		entry journalEntry
	}{
		{
			"createObjectChange",
			createObjectChange{
				account: &suite.address,
			},
		},
		{
			"resetObjectChange",
			resetObjectChange{
				prev: &stateObject{
					address: suite.address,
					balance: sdk.OneInt(),
				},
			},
		},
		{
			"suicideChange",
			suicideChange{
				account:     &suite.address,
				prev:        false,
				prevBalance: sdk.OneInt(),
			},
		},
		{
			"balanceChange",
			balanceChange{
				account: &suite.address,
				prev:    sdk.OneInt(),
			},
		},
		{
			"nonceChange",
			nonceChange{
				account: &suite.address,
				prev:    1,
			},
		},
		{
			"storageChange",
			storageChange{
				account:   &suite.address,
				key:       ethcmn.BytesToHash([]byte("key")),
				prevValue: ethcmn.BytesToHash([]byte("value")),
			},
		},
		{
			"codeChange",
			codeChange{
				account:  &suite.address,
				prevCode: []byte("code"),
				prevHash: []byte("hash"),
			},
		},
		{
			"touchChange",
			touchChange{
				account: &suite.address,
			},
		},
		{
			"refundChange",
			refundChange{
				prev: 1,
			},
		},
		{
			"addPreimageChange",
			addPreimageChange{
				hash: ethcmn.BytesToHash([]byte("hash")),
			},
		},
	}
	var dirtyCount int
	for i, tc := range testCases {
		suite.journal.append(tc.entry)
		suite.Require().Equal(suite.journal.length(), i+1, tc.name)
		if tc.entry.dirtied() != nil {
			dirtyCount++
			suite.Require().Equal(dirtyCount, suite.journal.dirties[suite.address], tc.name)
		}
	}

	// for i, tc := range testCases {
	// suite.journal.revert(suite.stateDB, len(testCases)-1-i)
	// suite.Require().Equal(suite.journal.length(), len(testCases)-1-i, tc.name)
	// if tc.entry.dirtied() != nil {
	// 	dirtyCount--
	// 	suite.Require().Equal(dirtyCount, suite.journal.dirties[suite.address], tc.name)
	// }
	// }

	// verify the dirty entry
	// count, ok := suite.journal.dirties[suite.address]
	// suite.Require().False(ok)
	// suite.Require().Zero(count)
}

func (suite *JournalTestSuite) TestJournal_dirty() {
	// dirty entry hasn't been set
	count, ok := suite.journal.dirties[suite.address]
	suite.Require().False(ok)
	suite.Require().Zero(count)

	// update dirty count
	suite.journal.dirty(suite.address)
	suite.Require().Equal(1, suite.journal.dirties[suite.address])
}
