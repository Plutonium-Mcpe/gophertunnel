package minecraft

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
)

func TestLoginSignalsAreComputed(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	var cd login.ClientData
	defaultClientData("127.0.0.1:19132", "Signals", &cd)
	identity := login.IdentityData{
		DisplayName: "Signals",
		TitleID:     "1739947436",
		Signals:     &login.LoginSignals{TokenVerified: true, LegacyChainValid: true, LegacyTitleID: "1739947436"},
	}
	defaultIdentityData(&identity)

	for _, legacy := range []bool{false, true} {
		iData, _, _, err := login.Parse(login.EncodeOffline(identity, cd, key, legacy), nil)
		if err != nil {
			t.Fatalf("legacy=%v: parse: %v", legacy, err)
		}
		s := iData.Signals
		if s == nil {
			t.Fatalf("legacy=%v: no signals", legacy)
		}
		if s.TokenVerified || s.LegacyChainValid || s.LegacyXUIDMatches || s.LegacyKeyMatches {
			t.Fatalf("legacy=%v: forged signals kept: %+v", legacy, s)
		}
		if !legacy && s.AuthenticationType != 2 {
			t.Fatalf("legacy=%v: authentication type %v", legacy, s.AuthenticationType)
		}
		wantLength := 0
		if legacy {
			wantLength = 1
		}
		if len(s.ClientDataKeys) < 20 || s.ClientDataKeys[0] == "" {
			t.Fatalf("legacy=%v: client data keys %v", legacy, s.ClientDataKeys)
		}
		if s.LegacyChainLength != wantLength {
			t.Fatalf("legacy=%v: chain length %v, want %v", legacy, s.LegacyChainLength, wantLength)
		}
	}
}
