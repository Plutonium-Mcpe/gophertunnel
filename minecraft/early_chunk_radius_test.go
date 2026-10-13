package minecraft

import (
	"bytes"
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// Un packetData sortant de parseData pointe encore sur le tampon de lot du decodeur, qui
// est recycle au lot suivant. Le retenir tel quel pour le rejouer apres le start_game
// revient a rejouer les octets du paquet qui a pris sa place.
func TestRetainedPacketSurvivesBufferReuse(t *testing.T) {
	pooled := []byte{byte(packet.IDRequestChunkRadius), 0x08, 0x00}
	parsed, err := parseData(pooled, &Conn{})
	if err != nil {
		t.Fatalf("parseData: %v", err)
	}
	if parsed.owned {
		t.Fatal("parseData devrait rendre un paquet non proprietaire de son tampon")
	}

	retained := parsed.ensureOwned()
	before := bytes.Clone(retained.payload.Bytes())
	aliased := bytes.Clone(parsed.payload.Bytes())

	// Le decodeur reutilise le tampon pour le lot suivant.
	for i := range pooled {
		pooled[i] = 0xFF
	}

	// Le bug : retenir parsed tel quel laisse sa charge se faire reecrire sous les pieds.
	if bytes.Equal(parsed.payload.Bytes(), aliased) {
		t.Fatal("le paquet non proprietaire aurait du etre reecrit, le test ne prouve rien")
	}
	// Le correctif : la copie retenue est intacte.
	if got := retained.payload.Bytes(); !bytes.Equal(got, before) {
		t.Fatalf("la charge retenue a ete reecrite : %v, attendu %v", got, before)
	}
}

// Le paquet retenu pour le rejeu et celui remis a l'utilisateur doivent etre deux copies
// distinctes : chacun est decode de son cote, et decoder consomme le tampon.
func TestRetainedAndDeliveredCopiesAreIndependent(t *testing.T) {
	pooled := []byte{byte(packet.IDRequestChunkRadius), 0x08, 0x00}
	parsed, err := parseData(pooled, &Conn{})
	if err != nil {
		t.Fatalf("parseData: %v", err)
	}

	retained := parsed.ensureOwned()
	delivered := parsed.ensureOwned()
	if retained == delivered {
		t.Fatal("les deux copies partagent le meme packetData")
	}

	// Consommer la charge de l'une ne doit pas vider l'autre.
	_ = delivered.payload.Next(delivered.payload.Len())
	if retained.payload.Len() == 0 {
		t.Fatal("consommer la copie remise a l'utilisateur a vide celle retenue pour le rejeu")
	}
}
