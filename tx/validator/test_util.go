// TODO need to refactor

package validator

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lino-network/lino/genesis"
	"github.com/lino-network/lino/global"
	acc "github.com/lino-network/lino/tx/account"
	vote "github.com/lino-network/lino/tx/vote"
	"github.com/lino-network/lino/types"
	"github.com/stretchr/testify/assert"
	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/go-crypto"
	dbm "github.com/tendermint/tmlibs/db"
)

// Construct some global addrs and txs for tests.
var (
	TestAccountKVStoreKey   = sdk.NewKVStoreKey("account")
	TestValidatorKVStoreKey = sdk.NewKVStoreKey("validator")
	TestGlobalKVStoreKey    = sdk.NewKVStoreKey("global")
	TestVoteKVStoreKey      = sdk.NewKVStoreKey("vote")
  
	initCoin = types.NewCoin(100)
)

func InitGlobalManager(ctx sdk.Context, gm *global.GlobalManager) error {
	globalState := genesis.GlobalState{
		TotalLino:                10000,
		GrowthRate:               sdk.Rat{98, 1000},
		InfraAllocation:          sdk.Rat{20, 100},
		ContentCreatorAllocation: sdk.Rat{55, 100},
		DeveloperAllocation:      sdk.Rat{20, 100},
		ValidatorAllocation:      sdk.Rat{5, 100},
		ConsumptionFrictionRate:  sdk.Rat{1, 100},
		FreezingPeriodHr:         24 * 7,
	}
	return gm.InitGlobalManager(ctx, globalState)
}

func setupTest(t *testing.T, height int64) (sdk.Context, *acc.AccountManager, *ValidatorManager, *vote.VoteManager, *global.GlobalManager) {
	ctx := getContext(height)
	accManager := acc.NewAccountManager(TestAccountKVStoreKey)
	postManager := NewValidatorManager(TestValidatorKVStoreKey)
	globalManager := global.NewGlobalManager(TestGlobalKVStoreKey)
	voteManager := vote.NewVoteManager(TestVoteKVStoreKey)
	err := InitGlobalManager(ctx, globalManager)
	assert.Nil(t, err)
	return ctx, accManager, postManager, voteManager, globalManager
}

func getContext(height int64) sdk.Context {
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(TestAccountKVStoreKey, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(TestValidatorKVStoreKey, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(TestGlobalKVStoreKey, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(TestVoteKVStoreKey, sdk.StoreTypeIAVL, db)
	ms.LoadLatestVersion()

	return sdk.NewContext(ms, abci.Header{Height: height}, false, nil)
}

// helper function to create an account for testing purpose
func createTestAccount(ctx sdk.Context, am *acc.AccountManager, username string) types.AccountKey {
	priv := crypto.GenPrivKeyEd25519()
	am.AddCoinToAddress(ctx, priv.PubKey().Address(), initCoin)
	am.CreateAccount(ctx, types.AccountKey(username), priv.PubKey(), types.NewCoin(0))
	return types.AccountKey(username)
}