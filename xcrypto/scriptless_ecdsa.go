// tokucore
//
// Copyright (c) 2019 TokuBlock
// BSD License

package xcrypto

import (
	"errors"
	"math/big"

	"github.com/tokublock/tokucore/xcrypto/paillier"
	"github.com/tokublock/tokucore/xcrypto/secp256k1"
)

// EcdsaAlice --
type EcdsaAlice struct {
	*EcdsaParty
}

// NewEcdsaAlice -- creates new EcdsaAlice.
func NewEcdsaAlice(prv *PrvKey) (*EcdsaAlice, error) {
	party, err := NewEcdsaParty(prv)
	if err != nil {
		return nil, err
	}
	return &EcdsaAlice{party}, nil
}

// ScriptlessPhase1 -- used to generate final pubkey of parties.
func (alice *EcdsaAlice) ScriptlessPhase1(pub2 *PubKey) *PubKey {
	return alice.Phase1(pub2)
}

// ScriptlessPhase2 -- used to generate k, kinv, scalarR.
func (alice *EcdsaAlice) ScriptlessPhase2(hash []byte) *secp256k1.Scalar {
	return alice.Phase2(hash)
}

// ScriptlessPhase3 -- return the shared R.
func (alice *EcdsaAlice) ScriptlessPhase3(encpk2 *big.Int, encpub2 *paillier.PubKey, r2 *secp256k1.Scalar) *secp256k1.Scalar {
	return alice.Phase3(encpk2, encpub2, r2)
}

// ScriptlessPhase4 -- return the homomorphic ciphertext.
func (alice *EcdsaAlice) ScriptlessPhase4(shareR *secp256k1.Scalar) (*big.Int, error) {
	return alice.Phase4(shareR)
}

// ScriptlessPhase5 -- return the partial signature of alice party.
func (alice *EcdsaAlice) ScriptlessPhase5(shareR *secp256k1.Scalar, sign2 *big.Int) (*big.Int, error) {
	N := alice.N
	kinv := alice.kinv
	encprv := alice.encprv

	sig, err := encprv.Decrypt(sign2)
	if err != nil {
		return nil, err
	}
	s := sig.Mul(sig, kinv).Mod(sig, N)
	return s, nil
}

// ScriptlessPhase6 -- get the secret T.
func (alice *EcdsaAlice) ScriptlessPhase6(alicesig *big.Int, bobsig *big.Int) *big.Int {
	N := alice.N
	t := new(big.Int).Set(alicesig)
	bobsiginv := new(big.Int).ModInverse(bobsig, N)
	t = t.Mul(t, bobsiginv).Mod(t, N)
	return t
}

// EcdsaBob --
type EcdsaBob struct {
	secret *big.Int
	*EcdsaParty
}

// NewEcdsaBob -- creates new EcdsaBob with a secret.
func NewEcdsaBob(prv *PrvKey, secret *big.Int) (*EcdsaBob, error) {
	party, err := NewEcdsaParty(prv)
	if err != nil {
		return nil, err
	}
	return &EcdsaBob{secret, party}, nil
}

// ScriptlessPhase1 -- used to generate final pubkey of parties.
func (bob *EcdsaBob) ScriptlessPhase1(pub2 *PubKey) *PubKey {
	return bob.Phase1(pub2)
}

// ScriptlessPhase2 -- used to generate k, kinv, scalarR.
// R=bobR*secret
func (bob *EcdsaBob) ScriptlessPhase2(hash []byte) *secp256k1.Scalar {
	curve := bob.curve
	secret := bob.secret
	scalar := bob.Phase2(hash)

	tx, ty := curve.ScalarMult(scalar.X, scalar.Y, secret.Bytes())
	return secp256k1.NewScalar(curve, tx, ty)
}

// ScriptlessPhase3 -- return the shared R.
func (bob *EcdsaBob) ScriptlessPhase3(encpk2 *big.Int, encpub2 *paillier.PubKey, r2 *secp256k1.Scalar) *secp256k1.Scalar {
	curve := bob.curve
	secret := bob.secret
	scalar := bob.Phase3(encpk2, encpub2, r2)
	tx, ty := curve.ScalarMult(scalar.X, scalar.Y, secret.Bytes())
	return secp256k1.NewScalar(curve, tx, ty)
}

// ScriptlessPhase4 -- return the homomorphic ciphertext.
func (bob *EcdsaBob) ScriptlessPhase4(shareR *secp256k1.Scalar) (*big.Int, error) {
	return bob.Phase4(shareR)
}

// ScriptlessPhase5 -- return the final signature of two party.
func (bob *EcdsaBob) ScriptlessPhase5(shareR *secp256k1.Scalar, sign2 *big.Int) (*big.Int, error) {
	N := bob.N
	kinv := bob.kinv
	encprv := bob.encprv
	secret := bob.secret
	tinv := new(big.Int).ModInverse(secret, N)

	sig, err := encprv.Decrypt(sign2)
	if err != nil {
		return nil, err
	}
	s := sig.Mul(sig, kinv).Mul(sig, tinv).Mod(sig, N)
	return s, nil
}

// ScriptlessPhase6 -- returns the DER signature.
func (bob *EcdsaBob) ScriptlessPhase6(shareR *secp256k1.Scalar, sig *big.Int) ([]byte, error) {
	N := bob.N

	s := new(big.Int).Set(sig)
	halfOrder := new(big.Int).Rsh(N, 1)
	if s.Cmp(halfOrder) == 1 {
		s.Sub(N, s)
	}
	if s.Sign() == 0 {
		return nil, errors.New("calculated S is zero")
	}
	esig := NewSignatureEcdsa()
	esig.R = shareR.X
	esig.S = s
	return esig.Serialize()
}
