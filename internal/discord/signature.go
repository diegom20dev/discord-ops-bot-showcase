package discord

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
)

type SignatureValidator struct {
	publicKey string
}

func NewSignatureValidator(publicKey string) *SignatureValidator {
	return &SignatureValidator{publicKey: publicKey}
}

// VerifySignature validates Discord interaction request signature
// https://discord.com/developers/docs/interactions/receiving-and-responding#security
func (sv *SignatureValidator) VerifySignature(signature, timestamp, body string) error {
	publicKeyBytes, err := hex.DecodeString(sv.publicKey)
	if err != nil {
		return fmt.Errorf("failed to decode public key: %w", err)
	}

	message := timestamp + body
	signatureBytes, err := hex.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	if !ed25519.Verify(ed25519.PublicKey(publicKeyBytes), []byte(message), signatureBytes) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}
