package transaction

import (
	"github.com/pkg/errors"
	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire"
	"github.com/tendermint/go-wire/data"
	. "github.com/tendermint/tmlibs/common"
	keys "github.com/tendermint/go-crypto/keys"
)

type PostTx struct {
	Address   data.Bytes       `json:"address"`   // Hash of the PubKey
	Title     string           `json:"title"`
	Content   string           `json:"content"`
	Sequence  int              `json:"sequence"`  // Must be 1 greater than the last committed PostTx
	Parent    []byte           `json:"parent"`
	Signature crypto.Signature `json:"signature"` // Depends on the PubKey type and the whole Tx
	PubKey    crypto.PubKey    `json:"pub_key"`   // Is present iff Sequence == 0
}

func (tx *PostTx) SignBytes(chainID string) []byte {
	signBytes := wire.BinaryBytes(chainID)
	sig := tx.Signature
	tx.Signature = crypto.Signature{}
	signBytes = append(signBytes, wire.BinaryBytes(tx)...)
	tx.Signature = sig
	return signBytes
}

func (tx *PostTx) SetSignature(sig crypto.Signature) bool {
	tx.Signature = sig
	return true
}

func (tx PostTx) ValidateBasic() abci.Result {
	if len(tx.Address) != 20 {
		return abci.ErrBaseInvalidInput.AppendLog("Invalid address length")
	}
	if tx.Sequence <= 0 {
		return abci.ErrBaseInvalidInput.AppendLog("Sequence must be greater than 0")
	}
	if tx.Sequence == 1 && tx.PubKey.Empty() {
		return abci.ErrBaseInvalidInput.AppendLog("PubKey must be present when Sequence == 1")
	}
	if tx.Sequence > 1 && !tx.PubKey.Empty() {
		return abci.ErrBaseInvalidInput.AppendLog("PubKey must be nil when Sequence > 1")
	}
	return abci.OK
}

func (tx *PostTx) String() string {
	return Fmt("PostTx{%v, %v, %v, %v, %v, %v}", tx.Address, tx.Title, tx.Content, tx.Sequence, tx.Parent, tx.PubKey)
}

// ============================================================================
// CliPostTx Application transaction structure for client
type CliPostTx struct {
	ChainID string
	signers []crypto.PubKey
	Tx      *PostTx
}

var _ keys.Signable = &CliPostTx{}

// SignBytes returned the unsigned bytes, needing a signature
func (p *CliPostTx) SignBytes() []byte {
	return p.Tx.SignBytes(p.ChainID)
}

// AddSigner sets address and pubkey info on the tx based on the key that
// will be used for signing
func (p *CliPostTx) AddSigner(pk crypto.PubKey) {
	p.Tx.Address = pk.Address()
	if p.Tx.Sequence == 1 {
		p.Tx.PubKey = pk
	}
}

// Sign will add a signature and pubkey.
//
// Depending on the Signable, one may be able to call this multiple times for multisig
// Returns error if called with invalid data or too many times
func (p *CliPostTx) Sign(pubkey crypto.PubKey, sig crypto.Signature) error {
	addr := pubkey.Address()
	set := p.Tx.SetSignature(sig)
	if !set {
		return errors.Errorf("Cannot add signature for address %X", addr)
	}
	return nil
}
// Signers will return the public key(s) that signed if the signature
// is valid, or an error if there is any issue with the signature,
// including if there are no signatures
func (p *CliPostTx) Signers() ([]crypto.PubKey, error) {
	if len(p.signers) == 0 {
		return nil, errors.New("No signatures on SendTx")
	}
	return p.signers, nil
}

// TxBytes returns the transaction data as well as all signatures
// It should return an error if Sign was never called
func (p *CliPostTx) TxBytes() ([]byte, error) {
	// TODO: verify it is signed

	// Code and comment from: basecoin/cmd/basecoin/commands/tx.go
	// Don't you hate having to do this?
	// How many times have I lost an hour over this trick?!
	txBytes := wire.BinaryBytes(struct {
		Tx `json:"unwrap"`
	}{p.Tx})
	return txBytes, nil
}

// TODO: this should really be in the basecoin.types SendTx,
// but that code is too ugly now, needs refactor..
func (p *CliPostTx) ValidateBasic() error {
	if p.ChainID == "" {
		return errors.New("No chain-id specified")
	}
	if len(p.Tx.Address) != 20 {
		return errors.Errorf("Invalid address length: %d", len(p.Tx.Address))
	}
	if p.Tx.Sequence <= 0 {
		return errors.New("Sequence must be greater than 0")
	}

	return nil
}