package keeper

import (
	"context"
	"strconv"

	"github.com/alice/checkers/x/checkers/rules"
	"github.com/alice/checkers/x/checkers/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) CreateGame(goCtx context.Context, msg *types.MsgCreateGame) (*types.MsgCreateGameResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// get the index of the next game. The index is stored in the keeper as a uint64,
	// so it must be converted into a string to be set in the StoredGame struct
	nextGame, found := k.Keeper.GetNextGame(ctx)
	if !found {
		panic("NextGame not found")
	}
	newIndex := strconv.FormatUint(nextGame.IdValue, 10)

	// create a new game with this index and message data
	newGame := rules.New()
	storedGame := types.StoredGame{
		Creator:   msg.Creator,
		Index:     newIndex,
		Game:      newGame.String(),
		Turn:      rules.PieceStrings[newGame.Turn],
		Red:       msg.Red,
		Black:     msg.Black,
		MoveCount: 0,
		BeforeId:  types.NoFifoIdKey,
		AfterId:   types.NoFifoIdKey,
		Deadline:  types.FormatDeadline(types.GetNextDeadline(ctx)),
		Winner:    rules.PieceStrings[rules.NO_PLAYER],
	}

	// validate that the new game has been created correctly
	// here msg.Red & msg.Black need to be validated because they were passed
	// as strings from the msg. The other elements have already been validated at
	// this point
	err := storedGame.Validate()
	if err != nil {
		return nil, err
	}

	// register the game in the FIFO list of games and set the nextGame FIFO tail
	k.Keeper.SendToFifoTail(ctx, &storedGame, &nextGame)

	// save the stored game in the keeper's store. This setter method was kindly
	// generated by the ignite cli when we scaffolded the storedGame map as module state
	k.Keeper.SetStoredGame(ctx, storedGame)

	// set the id for the next game to be created. This can be done here without
	// collision because a module's logic is run in a single thread on the node.
	// We can clobber this nextGame.IdValue here becuase it has been stored as a
	// string in the local newIndex variable and nextGame.IdValue is not used again
	nextGame.IdValue++
	k.Keeper.SetNextGame(ctx, nextGame)

	// emit an event to notify the player a game has been started
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, "checkers"),
			sdk.NewAttribute(sdk.AttributeKeyAction, types.StoredGameEventKey),
			sdk.NewAttribute(types.StoredGameEventCreator, msg.Creator),
			sdk.NewAttribute(types.StoredGameEventIndex, newIndex),
			sdk.NewAttribute(types.StoredGameEventRed, msg.Red),
			sdk.NewAttribute(types.StoredGameEventBlack, msg.Black),
		),
	)

	// return the IdValue for the newly created game
	return &types.MsgCreateGameResponse{
		IdValue: newIndex,
	}, nil
}
