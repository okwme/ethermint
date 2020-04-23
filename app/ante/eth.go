package ante

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"

	emint "github.com/cosmos/ethermint/types"
	evmtypes "github.com/cosmos/ethermint/x/evm/types"

	ethcore "github.com/ethereum/go-ethereum/core"
)

// EthSetupContextDecorator sets the infinite GasMeter in the Context and wraps
// the next AnteHandler with a defer clause to recover from any downstream
// OutOfGas panics in the AnteHandler chain to return an error with information
// on gas provided and gas used.
// CONTRACT: Must be first decorator in the chain
// CONTRACT: Tx must implement GasTx interface
type EthSetupContextDecorator struct{}

// NewEthSetupContextDecorator creates a new EthSetupContextDecorator
func NewEthSetupContextDecorator() EthSetupContextDecorator {
	return EthSetupContextDecorator{}
}

// AnteHandle sets the infinite gas meter to done to ignore costs in AnteHandler checks.
// This is undone at the EthGasConsumeDecorator, where the context is set with the
// ethereum tx GasLimit.
func (escd EthSetupContextDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	ctx = ctx.WithBlockGasMeter(sdk.NewInfiniteGasMeter())

	// all transactions must implement GasTx
	gasTx, ok := tx.(authante.GasTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be GasTx")
	}

	// Decorator will catch an OutOfGasPanic caused in the next antehandler
	// AnteHandlers must have their own defer/recover in order for the BaseApp
	// to know how much gas was used! This is because the GasMeter is created in
	// the AnteHandler, but if it panics the context won't be set properly in
	// runTx's recover call.
	defer func() {
		if r := recover(); r != nil {
			switch rType := r.(type) {
			case sdk.ErrorOutOfGas:
				log := fmt.Sprintf(
					"out of gas in location: %v; gasLimit: %d, gasUsed: %d",
					rType.Descriptor, gasTx.GetGas(), ctx.GasMeter().GasConsumed(),
				)
				err = sdkerrors.Wrap(sdkerrors.ErrOutOfGas, log)
			default:
				panic(r)
			}
		}
	}()

	return next(ctx, tx, simulate)
}

// EthMempoolFeeDecorator validates that sufficient fees have been provided that
// meet a minimum threshold defined by the proposer (for mempool purposes during CheckTx).
type EthMempoolFeeDecorator struct{}

// NewEthMempoolFeeDecorator creates a new EthMempoolFeeDecorator
func NewEthMempoolFeeDecorator() EthMempoolFeeDecorator {
	return EthMempoolFeeDecorator{}
}

// AnteHandle verifies that enough fees have been provided by the
// Ethereum transaction that meet the minimum threshold set by the block
// proposer.
//
// NOTE: This should only be run during a CheckTx mode.
func (emfd EthMempoolFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if !ctx.IsCheckTx() {
		return next(ctx, tx, simulate)
	}

	msgEthTx, ok := tx.(evmtypes.MsgEthereumTx)
	if !ok {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "invalid transaction type: %T", tx)
	}

	// fee = GP * GL
	fee := sdk.NewInt64DecCoin(emint.DenomDefault, msgEthTx.Fee().Int64())

	minGasPrices := ctx.MinGasPrices()

	allGTE := true
	for _, v := range minGasPrices {
		if !fee.IsGTE(v) {
			allGTE = false
		}
	}

	// it is assumed that the minimum fees will only include the single valid denom
	if !ctx.MinGasPrices().IsZero() && !allGTE {
		// reject the transaction that does not meet the minimum fee
		return ctx, sdkerrors.Wrap(
			sdkerrors.ErrInsufficientFee,
			fmt.Sprintf("insufficient fee, got: %q required: %q", fee, ctx.MinGasPrices()),
		)
	}

	return next(ctx, tx, simulate)
}

// EthSigVerificationDecorator validates an ethereum signature
type EthSigVerificationDecorator struct{}

// NewEthSigVerificationDecorator creates a new EthSigVerificationDecorator
func NewEthSigVerificationDecorator() EthSigVerificationDecorator {
	return EthSigVerificationDecorator{}
}

// AnteHandle validates the signature and returns sender address
func (esvd EthSigVerificationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	msgEthTx, ok := tx.(evmtypes.MsgEthereumTx)
	if !ok {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "invalid transaction type: %T", tx)
	}

	// parse the chainID from a string to a base-10 integer
	chainID, ok := new(big.Int).SetString(ctx.ChainID(), 10)
	if !ok {
		return ctx, sdkerrors.Wrap(emint.ErrInvalidChainID, ctx.ChainID())
	}

	// validate sender/signature
	// NOTE: signer is retrieved from the transaction on the next AnteDecorator
	_, err = msgEthTx.VerifySig(chainID)
	if err != nil {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "signature verification failed")
	}

	return next(ctx, msgEthTx, simulate)
}

// AccountVerificationDecorator validates an account balance checks
type AccountVerificationDecorator struct {
	ak auth.AccountKeeper
	bk bank.Keeper
}

// NewAccountVerificationDecorator creates a new AccountVerificationDecorator
func NewAccountVerificationDecorator(ak auth.AccountKeeper, bk bank.Keeper) AccountVerificationDecorator {
	return AccountVerificationDecorator{
		ak: ak,
		bk: bk,
	}
}

// AnteHandle validates the signature and returns sender address
func (avd AccountVerificationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if !ctx.IsCheckTx() {
		return next(ctx, tx, simulate)
	}

	msgEthTx, ok := tx.(evmtypes.MsgEthereumTx)
	if !ok {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "invalid transaction type: %T", tx)
	}

	// sender address should be in the tx cache
	address := msgEthTx.From()
	if address == nil {
		panic("sender address is nil")
	}

	acc := avd.ak.GetAccount(ctx, address)
	if acc == nil {
		return ctx, fmt.Errorf("account %s is nil", address)
	}

	// on InitChain make sure account number == 0
	if ctx.BlockHeight() == 0 && acc.GetAccountNumber() != 0 {
		return ctx, sdkerrors.Wrapf(
			sdkerrors.ErrInvalidSequence,
			"invalid account number for height zero (got %d)", acc.GetAccountNumber(),
		)
	}

	// validate sender has enough funds
	balance := avd.bk.GetBalance(ctx, acc.GetAddress(), emint.DenomDefault)
	if balance.Amount.BigInt().Cmp(msgEthTx.Cost()) < 0 {
		return ctx, sdkerrors.Wrapf(
			sdkerrors.ErrInsufficientFunds,
			"%s < %s%s", balance.String(), msgEthTx.Cost().String(), emint.DenomDefault,
		)
	}

	return next(ctx, tx, simulate)
}

// NonceVerificationDecorator that the nonce matches
type NonceVerificationDecorator struct {
	ak auth.AccountKeeper
}

// NewNonceVerificationDecorator creates a new NonceVerificationDecorator
func NewNonceVerificationDecorator(ak auth.AccountKeeper) NonceVerificationDecorator {
	return NonceVerificationDecorator{
		ak: ak,
	}
}

// AnteHandle validates that the transaction nonce is valid (equivalent to the sender account’s
// current nonce).
func (nvd NonceVerificationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	msgEthTx, ok := tx.(evmtypes.MsgEthereumTx)
	if !ok {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "invalid transaction type: %T", tx)
	}

	// sender address should be in the tx cache
	address := msgEthTx.From()
	if address == nil {
		panic("sender address is nil")
	}

	acc := nvd.ak.GetAccount(ctx, address)
	if acc == nil {
		return ctx, fmt.Errorf("account %s is nil", address)
	}

	seq := acc.GetSequence()
	if msgEthTx.Data.AccountNonce != seq {
		return ctx, sdkerrors.Wrap(
			sdkerrors.ErrInvalidSequence,
			fmt.Sprintf("invalid nonce; got %d, expected %d", msgEthTx.Data.AccountNonce, seq),
		)
	}

	return next(ctx, tx, simulate)
}

// EthGasConsumeDecorator validates enough intrinsic gas for the transaction and
// gas consumption.
type EthGasConsumeDecorator struct {
	ak auth.AccountKeeper
	sk types.SupplyKeeper
}

// NewEthGasConsumeDecorator creates a new EthGasConsumeDecorator
func NewEthGasConsumeDecorator(ak auth.AccountKeeper, sk types.SupplyKeeper) EthGasConsumeDecorator {
	return EthGasConsumeDecorator{
		ak: ak,
		sk: sk,
	}
}

// AnteHandle validates that the Ethereum tx message has enough to cover intrinsic gas
// (during CheckTx only) and that the sender has enough balance to pay for the gas cost.
//
// Intrinsic gas for a transaction is the amount of gas
// that the transaction uses before the transaction is executed. The gas is a
// constant value of 21000 plus any cost inccured by additional bytes of data
// supplied with the transaction.
func (egcd EthGasConsumeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	msgEthTx, ok := tx.(evmtypes.MsgEthereumTx)
	if !ok {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "invalid transaction type: %T", tx)
	}

	// sender address should be in the tx cache
	address := msgEthTx.From()
	if address == nil {
		panic("sender address is nil")
	}

	// Fetch sender account from signature
	senderAcc, err := auth.GetSignerAcc(ctx, egcd.ak, address)
	if err != nil {
		return ctx, err
	}

	if senderAcc == nil {
		return ctx, fmt.Errorf("sender account %s is nil", address)
	}

	gasLimit := msgEthTx.GetGas()
	gas, err := ethcore.IntrinsicGas(msgEthTx.Data.Payload, msgEthTx.To() == nil, true, false)
	if err != nil {
		return ctx, sdkerrors.Wrap(err, "failed to compute intrinsic gas cost")
	}

	// intrinsic gas verification during CheckTx
	if ctx.IsCheckTx() && gasLimit < gas {
		return ctx, fmt.Errorf("intrinsic gas too low: %d < %d", gasLimit, gas)
	}

	// Charge sender for gas up to limit
	if gasLimit != 0 {
		// Cost calculates the fees paid to validators based on gas limit and price
		cost := new(big.Int).Mul(msgEthTx.Data.Price, new(big.Int).SetUint64(gasLimit))

		feeAmt := sdk.NewCoins(
			sdk.NewCoin(emint.DenomDefault, sdk.NewIntFromBigInt(cost)),
		)

		err = auth.DeductFees(egcd.sk, ctx, senderAcc, feeAmt)
		if err != nil {
			return ctx, err
		}
	}

	// Set gas meter after ante handler to ignore gaskv costs
	newCtx = auth.SetGasMeter(simulate, ctx, gasLimit)
	newCtx.GasMeter().ConsumeGas(gas, "eth intrinsic gas")

	return next(newCtx, tx, simulate)
}

// IncrementSenderSequenceDecorator increments the sequence of the signers. The
// main difference with the SDK's IncrementSequenceDecorator is that the MsgEthereumTx
// doesn't implement the SigVerifiableTx interface.
//
// CONTRACT: must be called after msg.VerifySig in order to cache the sender address.
type IncrementSenderSequenceDecorator struct {
	ak auth.AccountKeeper
}

// NewIncrementSenderSequenceDecorator creates a new IncrementSenderSequenceDecorator.
func NewIncrementSenderSequenceDecorator(ak auth.AccountKeeper) IncrementSenderSequenceDecorator {
	return IncrementSenderSequenceDecorator{
		ak: ak,
	}
}

// AnteHandle handles incrementing the sequence of the sender.
func (issd IncrementSenderSequenceDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// no need to increment sequence on RecheckTx
	if ctx.IsReCheckTx() && !simulate {
		return next(ctx, tx, simulate)
	}

	// get and set account must be called with an infinite gas meter in order to prevent
	// additional gas from being deducted.
	oldCtx := ctx.WithBlockGasMeter(sdk.NewInfiniteGasMeter())

	msgEthTx, ok := tx.(evmtypes.MsgEthereumTx)
	if !ok {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "invalid transaction type: %T", tx)
	}

	// increment sequence of all signers
	for _, addr := range msgEthTx.GetSigners() {
		acc := issd.ak.GetAccount(oldCtx, addr)
		if err := acc.SetSequence(acc.GetSequence() + 1); err != nil {
			panic(err)
		}
		issd.ak.SetAccount(oldCtx, acc)
	}

	return next(ctx, tx, simulate)
}
