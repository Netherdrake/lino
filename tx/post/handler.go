package post

import (
	"fmt"
	"reflect"

	sdk "github.com/cosmos/cosmos-sdk/types"
	acc "github.com/lino-network/lino/tx/account"
	"github.com/lino-network/lino/tx/global"
	"github.com/lino-network/lino/types"
)

func NewHandler(pm PostManager, am acc.AccountManager, gm global.GlobalManager) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case CreatePostMsg:
			return handleCreatePostMsg(ctx, msg, pm, am, gm)
		case DonateMsg:
			return handleDonateMsg(ctx, msg, pm, am, gm)
		case LikeMsg:
			return handleLikeMsg(ctx, msg, pm, am, gm)
		case ReportOrUpvoteMsg:
			return handleReportOrUpvoteMsg(ctx, msg, pm, am, gm)
		case ViewMsg:
			return handleViewMsg(ctx, msg, pm, am, gm)
		default:
			errMsg := fmt.Sprintf("Unrecognized post msg type: %v", reflect.TypeOf(msg).Name())
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

// Handle RegisterMsg
func handleCreatePostMsg(ctx sdk.Context, msg CreatePostMsg, pm PostManager, am acc.AccountManager, gm global.GlobalManager) sdk.Result {
	if !am.IsAccountExist(ctx, msg.Author) {
		return ErrCreatePostAuthorNotFound(msg.Author).Result()
	}
	permLink := types.GetPermLink(msg.Author, msg.PostID)
	if pm.IsPostExist(ctx, permLink) {
		return ErrCreateExistPost(permLink).Result()
	}
	if len(msg.ParentAuthor) > 0 || len(msg.ParentPostID) > 0 {
		parentPostKey := types.GetPermLink(msg.ParentAuthor, msg.ParentPostID)
		if !pm.IsPostExist(ctx, parentPostKey) {
			return ErrCommentInvalidParent(parentPostKey).Result()
		}
		if err := pm.AddComment(ctx, parentPostKey, msg.Author, msg.PostID); err != nil {
			return err.Result()
		}
	}
	if err := pm.CreatePost(ctx, &msg.PostCreateParams); err != nil {
		return err.Result()
	}

	return sdk.Result{}
}

// Handle LikeMsg
func handleLikeMsg(ctx sdk.Context, msg LikeMsg, pm PostManager, am acc.AccountManager, gm global.GlobalManager) sdk.Result {
	if !am.IsAccountExist(ctx, msg.Username) {
		return ErrLikePostUserNotFound(msg.Username).Result()
	}
	permLink := types.GetPermLink(msg.Author, msg.PostID)
	if !pm.IsPostExist(ctx, permLink) {
		return ErrLikeNonExistPost(permLink).Result()
	}
	if err := pm.AddOrUpdateLikeToPost(ctx, permLink, msg.Username, msg.Weight); err != nil {
		return err.Result()
	}

	return sdk.Result{}
}

// Handle ViewMsg
func handleViewMsg(ctx sdk.Context, msg ViewMsg, pm PostManager, am acc.AccountManager, gm global.GlobalManager) sdk.Result {
	if !am.IsAccountExist(ctx, msg.Username) {
		return ErrViewPostUserNotFound(msg.Username).Result()
	}
	permLink := types.GetPermLink(msg.Author, msg.PostID)
	if !pm.IsPostExist(ctx, permLink) {
		return ErrViewNonExistPost(permLink).Result()
	}
	if err := pm.AddOrUpdateViewToPost(ctx, permLink, msg.Username); err != nil {
		return err.Result()
	}

	return sdk.Result{}
}

// Handle DonateMsg
func handleDonateMsg(ctx sdk.Context, msg DonateMsg, pm PostManager, am acc.AccountManager, gm global.GlobalManager) sdk.Result {
	permLink := types.GetPermLink(msg.Author, msg.PostID)
	coin, err := types.LinoToCoin(msg.Amount)
	if err != nil {
		return ErrDonateFailed(permLink).TraceCause(err, "").Result()
	}
	if !am.IsAccountExist(ctx, msg.Username) {
		return ErrDonateUserNotFound(msg.Username).Result()
	}
	if !pm.IsPostExist(ctx, permLink) {
		return ErrDonatePostNotFound(permLink).Result()
	}
	if msg.FromChecking {
		if err := am.MinusCheckingCoin(ctx, msg.Username, coin); err != nil {
			return ErrAccountCheckingCoinNotEnough(permLink).Result()
		}
	} else {
		if err := am.MinusSavingCoin(ctx, msg.Username, coin); err != nil {
			return ErrAccountSavingCoinNotEnough(permLink).Result()
		}
	}
	sourceAuthor, sourcePostID, err := pm.GetSourcePost(ctx, permLink)
	if err != nil {
		return ErrDonateFailed(permLink).TraceCause(err, "").Result()
	}
	if sourceAuthor != types.AccountKey("") && sourcePostID != "" {
		sourcePermLink := types.GetPermLink(sourceAuthor, sourcePostID)

		redistributionSplitRate, err := pm.GetRedistributionSplitRate(ctx, sourcePermLink)
		if err != nil {
			return ErrDonateFailed(permLink).TraceCause(err, "").Result()
		}
		sourceIncome := types.RatToCoin(coin.ToRat().Mul(sdk.OneRat.Sub(redistributionSplitRate)))
		coin = coin.Minus(sourceIncome)
		if err := processDonationFriction(
			ctx, msg.Username, sourceIncome, sourceAuthor, sourcePostID, msg.FromApp, am, pm, gm); err != nil {
			return err.Result()
		}
	}
	if err := processDonationFriction(
		ctx, msg.Username, coin, msg.Author, msg.PostID, msg.FromApp, am, pm, gm); err != nil {
		return err.Result()
	}
	return sdk.Result{}
}

func processDonationFriction(
	ctx sdk.Context, consumer types.AccountKey, coin types.Coin,
	postAuthor types.AccountKey, postID string, fromApp types.AccountKey,
	am acc.AccountManager, pm PostManager, gm global.GlobalManager) sdk.Error {
	postKey := types.GetPermLink(postAuthor, postID)
	if coin.IsZero() {
		return nil
	}
	if !am.IsAccountExist(ctx, postAuthor) {
		return ErrDonateAuthorNotFound(postKey, postAuthor)
	}
	consumptionFrictionRate, err := gm.GetConsumptionFrictionRate(ctx)
	if err != nil {
		return ErrDonateFailed(postKey).TraceCause(err, "")
	}
	frictionCoin := types.RatToCoin(coin.ToRat().Mul(consumptionFrictionRate))
	directDeposit := coin.Minus(frictionCoin)
	if err := pm.AddDonation(ctx, postKey, consumer, directDeposit); err != nil {
		return ErrDonateFailed(postKey).TraceCause(err, "")
	}
	if err := am.AddSavingCoin(ctx, postAuthor, directDeposit); err != nil {
		return ErrDonateFailed(postKey).TraceCause(err, "")
	}
	if err := gm.AddConsumption(ctx, coin); err != nil {
		return ErrDonateFailed(postKey).TraceCause(err, "")
	}
	// evaluate this consumption can get the result, the result is used to get inflation from pool
	evaluateResult, err := evaluateConsumption(ctx, consumer, coin, postAuthor, postID, am, pm, gm)
	if err != nil {
		return err
	}
	rewardEvent := RewardEvent{
		PostAuthor: postAuthor,
		PostID:     postID,
		Consumer:   consumer,
		Evaluate:   evaluateResult,
		Original:   coin,
		Friction:   frictionCoin,
		FromApp:    fromApp,
	}
	if err := gm.AddFrictionAndRegisterContentRewardEvent(
		ctx, rewardEvent, frictionCoin, evaluateResult); err != nil {
		return err
	}
	return nil
}

func evaluateConsumption(
	ctx sdk.Context, consumer types.AccountKey, coin types.Coin, postAuthor types.AccountKey,
	postID string, am acc.AccountManager, pm PostManager, gm global.GlobalManager) (types.Coin, sdk.Error) {
	numOfConsumptionOnAuthor, err := am.GetDonationRelationship(ctx, consumer, postAuthor)
	if err != nil {
		return types.NewCoin(0), err
	}
	created, totalReward, err := pm.GetCreatedTimeAndReward(ctx, types.GetPermLink(postAuthor, postID))
	if err != nil {
		return types.NewCoin(0), err
	}
	return gm.EvaluateConsumption(ctx, coin, numOfConsumptionOnAuthor, created, totalReward)

}

// Handle ReportMsgOrUpvoteMsg
func handleReportOrUpvoteMsg(
	ctx sdk.Context, msg ReportOrUpvoteMsg, pm PostManager, am acc.AccountManager, gm global.GlobalManager) sdk.Result {
	if !am.IsAccountExist(ctx, msg.Username) {
		return ErrReportUserNotFound(msg.Username).Result()
	}
	postKey := types.GetPermLink(msg.Author, msg.PostID)
	stake, err := am.GetStake(ctx, msg.Username)
	if err != nil {
		return ErrReportFailed(postKey).TraceCause(err, "").Result()
	}
	if !pm.IsPostExist(ctx, postKey) {
		return ErrReportPostDoesntExist(postKey).Result()
	}
	sourceAuthor, sourcePostID, err := pm.GetSourcePost(ctx, postKey)
	if err != nil {
		return ErrReportFailed(postKey).TraceCause(err, "").Result()
	}
	if sourceAuthor != types.AccountKey("") && sourcePostID != "" {
		sourcePermLink := types.GetPermLink(sourceAuthor, sourcePostID)
		if err := pm.ReportOrUpvoteToPost(
			ctx, sourcePermLink, msg.Username, stake, msg.IsReport); err != nil {
			return err.Result()
		}
	} else {
		if err := pm.ReportOrUpvoteToPost(
			ctx, postKey, msg.Username, stake, msg.IsReport); err != nil {
			return err.Result()
		}
	}
	return sdk.Result{}
}
