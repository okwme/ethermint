package types

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	zeroex "github.com/InjectiveLabs/zeroex-go"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common/math"
)

const RouterKey = ModuleName

// MsgCreateSpotOrder
type MsgCreateSpotOrder struct {
	Sender sdk.AccAddress   `json:"sender"`
	Order  *SafeSignedOrder `json:"order"`
}

// Route should return the name of the module
func (msg MsgCreateSpotOrder) Route() string { return RouterKey }

// Type should return the action
func (msg MsgCreateSpotOrder) Type() string { return "createSpotOrder" }

// ValidateBasic runs stateless checks on the message
func (msg MsgCreateSpotOrder) ValidateBasic() error {
	if msg.Sender.Empty() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Sender.String())
	}

	if msg.Order == nil {
		return sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "no make order specified")
	} else if _, err := msg.Order.ToSignedOrder().ComputeOrderHash(); err != nil {
		return sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, fmt.Sprintf("hash check failed: %v", err))
	} else if !isValidSignature(msg.Order.Signature) {
		return sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "invalid signature")
	}

	return nil
}

// isValidSignature checks that the signature of the order is correct
func isValidSignature(signature []byte) bool {
	signatureType := zeroex.SignatureType(signature[len(signature)-1])

	switch signatureType {
	case zeroex.InvalidSignature, zeroex.IllegalSignature:
		return false

	case zeroex.EIP712Signature:
		if len(signature) != 66 {
			return false
		}
		// TODO: Do further validation by splitting into r,s,v and do ECRecover

	case zeroex.EthSignSignature:
		if len(signature) != 66 {
			return false
		}
		// TODO: Do further validation by splitting into r,s,v, add prefix to hash
		// and do ECRecover

	case zeroex.ValidatorSignature:
		if len(signature) < 21 {
			return false
		}

	case zeroex.PreSignedSignature, zeroex.WalletSignature, zeroex.EIP1271WalletSignature:
		return true

	default:
		return false
	}

	return true
}

// GetSignBytes encodes the message for signing
func (msg MsgCreateSpotOrder) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgCreateSpotOrder) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}

// MsgCreateDerivativeOrder
type MsgCreateDerivativeOrder struct {
	Sender                 sdk.AccAddress   `json:"sender"`
	Order                  *SafeSignedOrder `json:"order"`
	InitialQuantityMatched BigNum           `json:"initialQuantityMatched"`
}

// Route should return the name of the module
func (msg MsgCreateDerivativeOrder) Route() string { return RouterKey }

// Type should return the action
func (msg MsgCreateDerivativeOrder) Type() string { return "createDerivativeOrder" }

// ValidateBasic runs stateless checks on the message
func (msg MsgCreateDerivativeOrder) ValidateBasic() error {
	if msg.Sender.Empty() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Sender.String())
	}

	if msg.Order == nil {
		return sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "no make order specified")
	} else if _, err := msg.Order.ToSignedOrder().ComputeOrderHash(); err != nil {
		return sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, fmt.Sprintf("hash check failed: %v", err))
	} else if !isValidSignature(msg.Order.Signature) {
		return sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "invalid signature")
	}

	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgCreateDerivativeOrder) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgCreateDerivativeOrder) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}

// MsgRequestFillSpotOrder
type MsgRequestFillSpotOrder struct {
	Sender            sdk.AccAddress     `json:"sender"`
	SignedTransaction *SignedTransaction `json:"signedTransaction"`
	TxOrigin          Address            `json:"txOrigin"`
	ApprovalSignature HexBytes           `json:"approvalSignature"`
}

// Route should return the name of the module
func (msg MsgRequestFillSpotOrder) Route() string { return RouterKey }

// Type should return the action
func (msg MsgRequestFillSpotOrder) Type() string { return "requestFillSpotOrder" }

// ValidateBasic runs stateless checks on the message
func (msg MsgRequestFillSpotOrder) ValidateBasic() error {
	if msg.Sender.Empty() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Sender.String())
	}

	if msg.TxOrigin.IsEmpty() {
		return sdkerrors.Wrap(ErrBadField, "no txOrigin address specified")
	} else if len(msg.SignedTransaction.Salt) == 0 {
		return sdkerrors.Wrap(ErrBadField, "no salt specified")
	} else if msg.SignedTransaction.SignerAddress.IsEmpty() {
		return sdkerrors.Wrap(ErrBadField, "no signerAddress address specified")
	} else if msg.SignedTransaction.Domain.VerifyingContract.IsEmpty() {
		return sdkerrors.Wrap(ErrBadField, "no verifyingContract address specified")
	} else if len(msg.SignedTransaction.Domain.ChainID) == 0 {
		return sdkerrors.Wrap(ErrBadField, "no chainID specified")
	} else if len(msg.SignedTransaction.GasPrice) == 0 {
		return sdkerrors.Wrap(ErrBadField, "no gasPrice specified")
	} else if len(msg.SignedTransaction.ExpirationTimeSeconds) == 0 {
		return sdkerrors.Wrap(ErrBadField, "no expirationTimeSeconds specified")
	} else if !isValidSignature(msg.SignedTransaction.Signature) {
		return sdkerrors.Wrap(ErrBadField, "invalid transaction signature")
	} else if !isValidSignature(msg.ApprovalSignature) {
		return sdkerrors.Wrap(ErrBadField, "invalid approval signature")
	}

	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgRequestFillSpotOrder) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgRequestFillSpotOrder) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}

// MsgRequestSoftCancelSpotOrder
type MsgRequestSoftCancelSpotOrder struct {
	Sender            sdk.AccAddress     `json:"sender"`
	SignedTransaction *SignedTransaction `json:"signedTransaction"`
	TxOrigin          Address            `json:"txOrigin"`
	ApprovalSignature HexBytes           `json:"approvalSignature"`
}

// Route should return the name of the module
func (msg MsgRequestSoftCancelSpotOrder) Route() string { return RouterKey }

// Type should return the action
func (msg MsgRequestSoftCancelSpotOrder) Type() string { return "softCancelSpotOrder" }

// ValidateBasic runs stateless checks on the message
func (msg MsgRequestSoftCancelSpotOrder) ValidateBasic() error {
	if msg.Sender.Empty() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Sender.String())
	}
	if msg.TxOrigin.IsEmpty() {
		return sdkerrors.Wrap(ErrBadField, "no txOrigin address specified")
	} else if len(msg.SignedTransaction.Salt) == 0 {
		return sdkerrors.Wrap(ErrBadField, "no salt specified")
	} else if msg.SignedTransaction.SignerAddress.IsEmpty() {
		return sdkerrors.Wrap(ErrBadField, "no signerAddress address specified")
	} else if msg.SignedTransaction.Domain.VerifyingContract.IsEmpty() {
		return sdkerrors.Wrap(ErrBadField, "no verifyingContract address specified")
	} else if len(msg.SignedTransaction.Domain.ChainID) == 0 {
		return sdkerrors.Wrap(ErrBadField, "no chainID specified")
	} else if len(msg.SignedTransaction.GasPrice) == 0 {
		return sdkerrors.Wrap(ErrBadField, "no gasPrice specified")
	} else if !isValidSignature(msg.SignedTransaction.Signature) {
		return sdkerrors.Wrap(ErrBadField, "invalid signature")
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgRequestSoftCancelSpotOrder) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgRequestSoftCancelSpotOrder) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}

// MsgFilledSpotOrder encodes an update msg when order
// becomes fully or partially filled.
type MsgFilledSpotOrder struct {
	Sender       sdk.AccAddress `json:"sender"`
	BlockNum     uint64         `json:"blockNum"`
	TxHash       Hash           `json:"txHash"`
	OrderHash    Hash           `json:"orderHash"`
	AmountFilled BigNum         `json:"amountFilled"`
}

// Route should return the name of the module
func (msg MsgFilledSpotOrder) Route() string { return RouterKey }

// Type should return the action
func (msg MsgFilledSpotOrder) Type() string { return "filledSpotOrder" }

// ValidateBasic runs stateless checks on the message
func (msg MsgFilledSpotOrder) ValidateBasic() error {
	if msg.Sender.Empty() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Sender.String())
	}

	if msg.BlockNum == 0 {
		return sdkerrors.Wrap(ErrBadField, "no blockNum is specified")
	} else if (msg.TxHash == Hash{}) {
		return sdkerrors.Wrap(ErrBadField, "no txHash is specified")
	} else if (msg.OrderHash == Hash{}) {
		return sdkerrors.Wrap(ErrBadField, "no orderHash is specified")
	}

	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgFilledSpotOrder) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgFilledSpotOrder) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}

// MsgCancelledSpotOrder
type MsgCancelledSpotOrder struct {
	Sender    sdk.AccAddress `json:"sender"`
	BlockNum  uint64         `json:"blockNum"`
	TxHash    Hash           `json:"txHash"`
	OrderHash Hash           `json:"orderHash"`
}

// Route should return the name of the module
func (msg MsgCancelledSpotOrder) Route() string { return RouterKey }

// Type should return the action
func (msg MsgCancelledSpotOrder) Type() string { return "cancelledSpotOrder" }

// ValidateBasic runs stateless checks on the message
func (msg MsgCancelledSpotOrder) ValidateBasic() error {
	if msg.Sender.Empty() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Sender.String())
	}

	if msg.BlockNum == 0 {
		return sdkerrors.Wrap(ErrBadField, "no blockNum is specified")
	} else if (msg.TxHash == Hash{}) {
		return sdkerrors.Wrap(ErrBadField, "no txHash is specified")
	} else if (msg.OrderHash == Hash{}) {
		return sdkerrors.Wrap(ErrBadField, "no orderHash is specified")
	}

	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgCancelledSpotOrder) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgCancelledSpotOrder) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}

// MsgFilledDerivativeOrder encodes an update msg when a position
// becomes fully or partially filled.
type MsgFilledDerivativeOrder struct {
	Sender         sdk.AccAddress `json:"sender"`
	BlockNum       uint64         `json:"blockNum"`
	TxHash         Hash           `json:"txHash"`
	MakerAddress   Address        `json:"makerAddress"`
	MarketID       Hash           `json:"marketId"`
	OrderHash      Hash           `json:"orderHash"`
	PositionID     BigNum         `json:"positionId"`
	QuantityFilled BigNum         `json:"quantityFilled"`
	ContractPrice  BigNum         `json:"contractPrice"`
	IsLong         bool           `json:"isLong"`
}

// Route should return the name of the module
func (msg MsgFilledDerivativeOrder) Route() string { return RouterKey }

// Type should return the action
func (msg MsgFilledDerivativeOrder) Type() string { return "filledDerivativeOrder" }

// ValidateBasic runs stateless checks on the message
func (msg MsgFilledDerivativeOrder) ValidateBasic() error {
	if msg.Sender.Empty() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Sender.String())
	}

	if msg.BlockNum == 0 {
		return sdkerrors.Wrap(ErrBadField, "no blockNum is specified")
	} else if (msg.TxHash == Hash{}) {
		return sdkerrors.Wrap(ErrBadField, "no txHash is specified")
	} else if (msg.OrderHash == Hash{}) {
		return sdkerrors.Wrap(ErrBadField, "no orderHash is specified")
	} else if (msg.MarketID == Hash{}) {
		return sdkerrors.Wrap(ErrBadField, "no marketId is specified")
	} else if (msg.MakerAddress == Address{}) {
		return sdkerrors.Wrap(ErrBadField, "no makerAddress is specified")
	}

	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgFilledDerivativeOrder) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgFilledDerivativeOrder) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}

// MsgCancelledDerivativeOrder
type MsgCancelledDerivativeOrder struct {
	Sender       sdk.AccAddress `json:"sender"`
	BlockNum     uint64         `json:"blockNum"`
	TxHash       Hash           `json:"txHash"`
	MakerAddress Address        `json:"makerAddress"`
	MarketID     Hash           `json:"marketId"`
	OrderHash    Hash           `json:"orderHash"`
	PositionID   BigNum         `json:"positionId"`
}

// Route should return the name of the module
func (msg MsgCancelledDerivativeOrder) Route() string { return RouterKey }

// Type should return the action
func (msg MsgCancelledDerivativeOrder) Type() string { return "cancelledDerivativeOrder" }

// ValidateBasic runs stateless checks on the message
func (msg MsgCancelledDerivativeOrder) ValidateBasic() error {
	if msg.Sender.Empty() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Sender.String())
	}

	if msg.BlockNum == 0 {
		return sdkerrors.Wrap(ErrBadField, "no blockNum is specified")
	} else if (msg.TxHash == Hash{}) {
		return sdkerrors.Wrap(ErrBadField, "no txHash is specified")
	} else if (msg.OrderHash == Hash{}) {
		return sdkerrors.Wrap(ErrBadField, "no orderHash is specified")
	} else if (msg.MarketID == Hash{}) {
		return sdkerrors.Wrap(ErrBadField, "no marketId is specified")
	} else if (msg.MakerAddress == Address{}) {
		return sdkerrors.Wrap(ErrBadField, "no makerAddress is specified")
	}

	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgCancelledDerivativeOrder) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgCancelledDerivativeOrder) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}

// MsgRegisterSpotMarket defines a RegisterSpotMarket message
type MsgRegisterSpotMarket struct {
	Sender         sdk.AccAddress `json:"sender"`
	Name           string         `json:"name"`
	MakerAssetData HexBytes       `json:"makerAssetData"`
	TakerAssetData HexBytes       `json:"takerAssetData"`
	Enabled        bool           `json:"enabled"`
}

// Route should return the name of the module
func (msg MsgRegisterSpotMarket) Route() string { return RouterKey }

// Type should return the action
func (msg MsgRegisterSpotMarket) Type() string { return "registerSpotMarket" }

// ValidateBasic runs stateless checks on the message
func (msg MsgRegisterSpotMarket) ValidateBasic() error {
	if msg.Sender.Empty() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Sender.String())
	}
	if len(msg.Name) == 0 {
		return sdkerrors.Wrap(ErrBadField, "no trade pair name specified")
	} else if parts := strings.Split(msg.Name, "/"); len(parts) != 2 ||
		len(strings.TrimSpace(parts[0])) == 0 || len(strings.TrimSpace(parts[1])) == 0 {
		return sdkerrors.Wrap(ErrBadField, "pair name must be in format AAA/BBB")
	}
	if len(msg.MakerAssetData) == 0 {
		return sdkerrors.Wrap(ErrBadField, "no maker asset data specified")
	} else if len(msg.TakerAssetData) == 0 {
		return sdkerrors.Wrap(ErrBadField, "no taker asset data specified")
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgRegisterSpotMarket) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgRegisterSpotMarket) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}

// MsgRegisterDerivativeMarket defines a RegisterDerivativeMarket message
type MsgRegisterDerivativeMarket struct {
	Sender       sdk.AccAddress `json:"sender"`
	Ticker       string         `json:"ticker"`
	Oracle       common.Address `json:"oracle"`
	BaseCurrency common.Address `json:"baseCurrency"`
	Nonce        BigNum         `json:"nonce"`
	MarketID     Hash           `json:"marketID"`
	Enabled      bool           `json:"enabled"`
}

// Route should return the name of the module
func (msg MsgRegisterDerivativeMarket) Route() string { return RouterKey }

// Type should return the action
func (msg MsgRegisterDerivativeMarket) Type() string { return "registerDerivativeMarket" }

// ValidateBasic runs stateless checks on the message
func (msg MsgRegisterDerivativeMarket) ValidateBasic() error {
	if msg.Sender.Empty() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Sender.String())
	}

	market := DerivativeMarket{
		Ticker:       msg.Ticker,
		Oracle:       msg.Oracle.Bytes(),
		BaseCurrency: msg.BaseCurrency.Bytes(),
		Nonce:        msg.Nonce,
		MarketID:     Hash{},
		Enabled:      true,
	}
	if len(msg.Ticker) == 0 {
		return sdkerrors.Wrap(ErrBadField, "no market ticker name specified")
	} else if parts := strings.Split(msg.Ticker, "/"); len(parts) != 2 ||
		len(strings.TrimSpace(parts[0])) == 0 || len(strings.TrimSpace(parts[1])) == 0 {
		return sdkerrors.Wrap(ErrBadField, "market ticker must be in format AAA/BBB")
	}
	hash, err := market.Hash()
	if hash != msg.MarketID.Hash {

		errMsg := "The MarketID " + msg.MarketID.String() + " provided does not match the MarketID " + hash.String() + " computed"
		errMsg += "\n for Ticker: " + msg.Ticker + "\nOracle: " + msg.Oracle.Hex() + "\nBaseCurrency: " + msg.BaseCurrency.Hex() + "\nNonce: " + msg.Nonce.Decimal().String()
		return sdkerrors.Wrap(ErrMarketInvalid, errMsg)
	}
	if err != nil {
		return sdkerrors.Wrap(ErrMarketInvalid, err.Error())
	}

	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgRegisterDerivativeMarket) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgRegisterDerivativeMarket) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}

// MsgSuspendDerivativeMarket allows to suspend a derivative market.
type MsgSuspendDerivativeMarket struct {
	Sender   sdk.AccAddress `json:"sender"`
	MarketID Hash           `json:"marketId"`
}

// Route should return the name of the module
func (msg MsgSuspendDerivativeMarket) Route() string { return RouterKey }

// Type should return the action
func (msg MsgSuspendDerivativeMarket) Type() string {
	return "suspendDerivativeMarket"
}

// ValidateBasic runs stateless checks on the message
func (msg MsgSuspendDerivativeMarket) ValidateBasic() error {
	if msg.Sender.Empty() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Sender.String())
	} else if (msg.MarketID.Hash == common.Hash{}) {
		return sdkerrors.Wrap(ErrBadField, "no derivative market ID specified")
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgSuspendDerivativeMarket) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgSuspendDerivativeMarket) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}

// MsgResumeDerivativeMarket allows to resume a suspended derivative market.
type MsgResumeDerivativeMarket struct {
	Sender   sdk.AccAddress `json:"sender"`
	MarketID Hash           `json:"marketId"`
}

// Route should return the name of the module
func (msg MsgResumeDerivativeMarket) Route() string { return RouterKey }

// Type should return the action
func (msg MsgResumeDerivativeMarket) Type() string {
	return "resumeDerivativeMarket"
}

// ValidateBasic runs stateless checks on the message
func (msg MsgResumeDerivativeMarket) ValidateBasic() error {
	if msg.Sender.Empty() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Sender.String())
	} else if (msg.MarketID.Hash == common.Hash{}) {
		return sdkerrors.Wrap(ErrBadField, "no derivative market ID specified")
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgResumeDerivativeMarket) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgResumeDerivativeMarket) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}

// MsgSuspendSpotMarket defines a SuspendSpotMarket message
type MsgSuspendSpotMarket struct {
	Sender         sdk.AccAddress `json:"sender"`
	Name           string         `json:"name"`
	MakerAssetData HexBytes       `json:"makerAssetData"`
	TakerAssetData HexBytes       `json:"takerAssetData"`
}

// Route should return the name of the module
func (msg MsgSuspendSpotMarket) Route() string { return RouterKey }

// Type should return the action
func (msg MsgSuspendSpotMarket) Type() string { return "suspendSpotMarket" }

// ValidateBasic runs stateless checks on the message
func (msg MsgSuspendSpotMarket) ValidateBasic() error {
	if msg.Sender.Empty() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Sender.String())
	}

	if len(msg.Name) == 0 {
		return sdkerrors.Wrap(ErrBadField, "no trade pair name specified")
	}

	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgSuspendSpotMarket) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgSuspendSpotMarket) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}

// MsgResumeSpotMarket defines a ResumeSpotMarket message
type MsgResumeSpotMarket struct {
	Sender         sdk.AccAddress `json:"sender"`
	Name           string         `json:"name"`
	MakerAssetData HexBytes       `json:"makerAssetData"`
	TakerAssetData HexBytes       `json:"takerAssetData"`
}

// Route should return the name of the module
func (msg MsgResumeSpotMarket) Route() string { return RouterKey }

// Type should return the action
func (msg MsgResumeSpotMarket) Type() string { return "resumeSpotMarket" }

// ValidateBasic runs stateless checks on the message
func (msg MsgResumeSpotMarket) ValidateBasic() error {
	if msg.Sender.Empty() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Sender.String())
	}

	if len(msg.Name) == 0 {
		return sdkerrors.Wrap(ErrBadField, "no trade pair name specified")
	}

	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgResumeSpotMarket) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgResumeSpotMarket) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}

// SafeSignedOrder is a special signed order structure
// for including in Msgs, because it consists of primitive types.
// Avoid using raw *big.Int in Msgs.
type SafeSignedOrder struct {
	// ChainID is a network identifier of the order.
	ChainID int64 `json:"chainID,omitempty"`
	// Exchange v3 contract address.
	ExchangeAddress Address `json:"exchangeAddress,omitempty"`
	// Address that created the order.
	MakerAddress Address `json:"makerAddress,omitempty"`
	// Address that is allowed to fill the order. If set to "0x0", any address is
	// allowed to fill the order.
	TakerAddress Address `json:"takerAddress,omitempty"`
	// Address that will receive fees when order is filled.
	FeeRecipientAddress Address `json:"feeRecipientAddress,omitempty"`
	// Address that is allowed to call Exchange contract methods that affect this
	// order. If set to "0x0", any address is allowed to call these methods.
	SenderAddress Address `json:"senderAddress,omitempty"`
	// Amount of makerAsset being offered by maker. Must be greater than 0.
	MakerAssetAmount BigNum `json:"makerAssetAmount,omitempty"`
	// Amount of takerAsset being bid on by maker. Must be greater than 0.
	TakerAssetAmount BigNum `json:"takerAssetAmount,omitempty"`
	// Amount of Fee Asset paid to feeRecipientAddress by maker when order is filled. If set to
	// 0, no transfer of Fee Asset from maker to feeRecipientAddress will be attempted.
	MakerFee BigNum `json:"makerFee,omitempty"`
	// Amount of Fee Asset paid to feeRecipientAddress by taker when order is filled. If set to
	// 0, no transfer of Fee Asset from taker to feeRecipientAddress will be attempted.
	TakerFee BigNum `json:"takerFee,omitempty"`
	// Timestamp in seconds at which order expires.
	ExpirationTimeSeconds BigNum `json:"expirationTimeSeconds,omitempty"`
	// Arbitrary number to facilitate uniqueness of the order's hash.
	Salt BigNum `json:"salt,omitempty"`
	// ABIv2 encoded data that can be decoded by a specified proxy contract when
	// transferring makerAsset.
	MakerAssetData HexBytes `json:"makerAssetData,omitempty"`
	// ABIv2 encoded data that can be decoded by a specified proxy contract when
	// transferring takerAsset.
	TakerAssetData HexBytes `json:"takerAssetData,omitempty"`
	// ABIv2 encoded data that can be decoded by a specified proxy contract when
	// transferring makerFee.
	MakerFeeAssetData HexBytes `json:"makerFeeAssetData,omitempty"`
	// ABIv2 encoded data that can be decoded by a specified proxy contract when
	// transferring takerFee.
	TakerFeeAssetData HexBytes `json:"takerFeeAssetData,omitempty"`
	// Order signature.
	Signature HexBytes `json:"signature,omitempty"`
}

// NewSafeSignedOrder constructs a new SafeSignedOrder from given zeroex.SignedOrder.
func NewSafeSignedOrder(o *zeroex.SignedOrder) *SafeSignedOrder {
	return zo2so(o)
}

// ToSignedOrder returns an appropriate zeroex.SignedOrder defined by SafeSignedOrder.
func (m *SafeSignedOrder) ToSignedOrder() *zeroex.SignedOrder {
	o, err := so2zo(m)
	if err != nil {
		panic(err)
	}
	return o
}

// zo2so internal function converts model from *zeroex.SignedOrder to *SafeSignedOrder.
func zo2so(o *zeroex.SignedOrder) *SafeSignedOrder {
	if o == nil {
		return nil
	}
	return &SafeSignedOrder{
		ChainID:               o.ChainID.Int64(),
		ExchangeAddress:       Address{o.ExchangeAddress},
		MakerAddress:          Address{o.MakerAddress},
		TakerAddress:          Address{o.TakerAddress},
		FeeRecipientAddress:   Address{o.FeeRecipientAddress},
		SenderAddress:         Address{o.SenderAddress},
		MakerAssetAmount:      BigNum(o.MakerAssetAmount.String()),
		TakerAssetAmount:      BigNum(o.TakerAssetAmount.String()),
		MakerFee:              BigNum(o.MakerFee.String()),
		TakerFee:              BigNum(o.TakerFee.String()),
		ExpirationTimeSeconds: BigNum(o.ExpirationTimeSeconds.String()),
		Salt:                  BigNum(o.Salt.String()),
		MakerAssetData:        o.MakerAssetData,
		TakerAssetData:        o.TakerAssetData,
		MakerFeeAssetData:     o.MakerFeeAssetData,
		TakerFeeAssetData:     o.TakerFeeAssetData,
		Signature:             o.Signature,
	}
}

// so2zo internal function converts model from *SafeSignedOrder to *zeroex.SignedOrder.
func so2zo(o *SafeSignedOrder) (*zeroex.SignedOrder, error) {
	if o == nil {
		return nil, nil
	}
	order := zeroex.Order{
		ChainID:             big.NewInt(o.ChainID),
		ExchangeAddress:     o.ExchangeAddress.Address,
		MakerAddress:        o.MakerAddress.Address,
		TakerAddress:        o.TakerAddress.Address,
		MakerAssetData:      o.MakerAssetData,
		TakerAssetData:      o.TakerAssetData,
		MakerFeeAssetData:   o.MakerFeeAssetData,
		TakerFeeAssetData:   o.TakerFeeAssetData,
		SenderAddress:       o.SenderAddress.Address,
		FeeRecipientAddress: o.FeeRecipientAddress.Address,
	}
	if v, ok := math.ParseBig256(string(o.MakerAssetAmount)); !ok {
		return nil, errors.New("makerAssetAmmount parse failed")
	} else {
		order.MakerAssetAmount = v
	}
	if v, ok := math.ParseBig256(string(o.MakerFee)); !ok {
		return nil, errors.New("makerFee parse failed")
	} else {
		order.MakerFee = v
	}
	if v, ok := math.ParseBig256(string(o.TakerAssetAmount)); !ok {
		return nil, errors.New("takerAssetAmmount parse failed")
	} else {
		order.TakerAssetAmount = v
	}
	if v, ok := math.ParseBig256(string(o.TakerFee)); !ok {
		return nil, errors.New("takerFee parse failed")
	} else {
		order.TakerFee = v
	}
	if v, ok := math.ParseBig256(string(o.ExpirationTimeSeconds)); !ok {
		return nil, errors.New("expirationTimeSeconds parse failed")
	} else {
		order.ExpirationTimeSeconds = v
	}
	if v, ok := math.ParseBig256(string(o.Salt)); !ok {
		return nil, errors.New("salt parse failed")
	} else {
		order.Salt = v
	}
	signedOrder := &zeroex.SignedOrder{
		Order:     order,
		Signature: o.Signature,
	}
	return signedOrder, nil
}
