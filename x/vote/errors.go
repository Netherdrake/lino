package vote

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lino-network/lino/types"
)

// Error constructors
func ErrUsernameNotFound() sdk.Error {
	return types.NewError(types.CodeVoteManagerFailed, fmt.Sprintf("Username not found"))
}

func ErrIllegalWithdraw() sdk.Error {
	return types.NewError(types.CodeVoteManagerFailed, fmt.Sprintf("Illegal withdraw"))
}

func ErrNoCoinToWithdraw() sdk.Error {
	return types.NewError(types.CodeVoteManagerFailed, fmt.Sprintf("No coin to withdraw"))
}

func ErrRegisterFeeNotEnough() sdk.Error {
	return types.NewError(types.CodeVoteManagerFailed, fmt.Sprintf("Register fee not enough"))
}

func ErrInvalidUsername() sdk.Error {
	return types.NewError(types.CodeVoteManagerFailed, fmt.Sprintf("Invalid Username"))
}

func ErrValidatorCannotRevoke() sdk.Error {
	return types.NewError(types.CodeVoteManagerFailed, fmt.Sprintf("Invalid revoke"))
}

func ErrVoteExist() sdk.Error {
	return types.NewError(types.CodeVoteManagerFailed, fmt.Sprintf("Vote exist"))
}
