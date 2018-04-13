package validator

import (
	"fmt"
	"reflect"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lino-network/lino/global"
	acc "github.com/lino-network/lino/tx/account"
	vote "github.com/lino-network/lino/tx/vote"
	"github.com/lino-network/lino/types"
)

func NewHandler(am acc.AccountManager, valManager ValidatorManager, voteManager vote.VoteManager, gm global.GlobalManager) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case ValidatorDepositMsg:
			return handleDepositMsg(ctx, valManager, voteManager, am, msg)
		case ValidatorWithdrawMsg:
			return handleWithdrawMsg(ctx, valManager, am, gm, msg)
		case ValidatorRevokeMsg:
			return handleRevokeMsg(ctx, valManager, gm, msg)
		default:
			errMsg := fmt.Sprintf("Unrecognized validator Msg type: %v", reflect.TypeOf(msg).Name())
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

// Handle DepositMsg
func handleDepositMsg(ctx sdk.Context, valManager ValidatorManager, voteManager vote.VoteManager, am acc.AccountManager, msg ValidatorDepositMsg) sdk.Result {
	// Must have an normal acount
	if !am.IsAccountExist(ctx, msg.Username) {
		return ErrUsernameNotFound().Result()
	}

	coin, err := types.LinoToCoin(msg.Deposit)
	if err != nil {
		return err.Result()
	}

	// withdraw money from validator's bank
	if err = am.MinusCoin(ctx, msg.Username, coin); err != nil {
		return err.Result()
	}

	// Register the user if this name has not been registered
	if !valManager.IsValidatorExist(ctx, msg.Username) {
		// check validator minimum voting deposit requirement
		if !voteManager.CanBecomeValidator(ctx, msg.Username) {
			return ErrVotingDepositNotEnough().Result()
		}
		if err := valManager.RegisterValidator(ctx, msg.Username, msg.ValPubKey.Bytes(), coin); err != nil {
			return err.Result()
		}
	} else {
		// Deposit coins
		if err := valManager.Deposit(ctx, msg.Username, coin); err != nil {
			return err.Result()
		}
	}

	// Try to become oncall validator
	if joinErr := valManager.TryBecomeOncallValidator(ctx, msg.Username); joinErr != nil {
		return joinErr.Result()
	}
	return sdk.Result{}
}

// Handle Withdraw Msg
func handleWithdrawMsg(ctx sdk.Context, vm ValidatorManager, am acc.AccountManager, gm global.GlobalManager, msg ValidatorWithdrawMsg) sdk.Result {
	coin, err := types.LinoToCoin(msg.Amount)
	if err != nil {
		return err.Result()
	}

	if !vm.IsLegalWithdraw(ctx, msg.Username, coin) {
		return ErrIllegalWithdraw().Result()
	}

	if err := vm.Withdraw(ctx, msg.Username, coin, gm); err != nil {
		return err.Result()
	}
	return sdk.Result{}
}

// Handle RevokeMsg
func handleRevokeMsg(ctx sdk.Context, vm ValidatorManager, gm global.GlobalManager, msg ValidatorRevokeMsg) sdk.Result {
	if err := vm.WithdrawAll(ctx, msg.Username, gm); err != nil {
		return err.Result()
	}
	if err := vm.RemoveValidatorFromAllLists(ctx, msg.Username); err != nil {
		return err.Result()
	}

	return sdk.Result{}
}