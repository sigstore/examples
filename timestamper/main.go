package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/sigstore/rekor/cmd/cli/app"
	"github.com/sigstore/rekor/pkg/generated/client"
)

// These are our globals, they get set up in main()

const gp = "globals.json"

type globals struct {
	// Marshalled
	Pub                []byte
	Priv               ed25519.PrivateKey
	Last1000Serialized [][]byte
	Last1000           []*timestampedAuditResult

	// Not marshalled
	rekorClient    *client.Rekor
	rekorPublicKey interface{}
}

const (
	rekorAddr = "https://api.rekor.dev"
	tsServer  = "https://freetsa.org/tsr"
)

var G globals

func initGlobals() {
	rekorClient, err := app.GetRekorClient(rekorAddr)
	if err != nil {
		panic(err)
	}
	G = globals{
		rekorClient: rekorClient,
	}
	G.rekorPublicKey = initRekorPublicKey(rekorClient)
	if _, err := os.Stat(gp); os.IsNotExist(err) {
		G.Pub, G.Priv = initKeys()

		b, _ := json.Marshal(G)
		ioutil.WriteFile(gp, b, 0644)
		return
	}
	b, err := ioutil.ReadFile(gp)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(b, &G); err != nil {
		panic(err)
	}
}

func initRekorPublicKey(c *client.Rekor) interface{} {
	keyResp, err := c.Tlog.GetPublicKey(nil)
	if err != nil {
		panic(err)
	}
	publicKey := keyResp.Payload

	block, _ := pem.Decode([]byte(publicKey))
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		panic(err)
	}
	return pub
}

func initKeys() ([]byte, ed25519.PrivateKey) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	derBytes, err := x509.MarshalPKIXPublicKey(public)
	if err != nil {
		panic(err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derBytes,
	})

	return pemBytes, private
}

func main() {
	initGlobals()

	log.Println("keys ready, length:", len(G.Last1000))
	http.HandleFunc("/keys", func(w http.ResponseWriter, r *http.Request) {
		w.Write(G.Pub)
		log.Println("served public key")
	})
	http.HandleFunc("/entries", func(w http.ResponseWriter, r *http.Request) {
		for _, l := range G.Last1000Serialized {
			fmt.Fprintln(w, string(l))
		}
		log.Println("served entries")
	})

	go func() {
		for {
			runLoop()
		}
	}()
	http.ListenAndServe("localhost:8080", nil)
}

func runLoop() {
	t := time.Tick(30 * time.Minute)
	for range t {
		this, err := runAudit()
		if err != nil {
			log.Println("ERROR:", err)
			continue
		}
		ser, _ := json.Marshal(this)
		G.Last1000 = append(G.Last1000, this)
		G.Last1000Serialized = append(G.Last1000Serialized, ser)
		if len(G.Last1000) >= 1000 {
			fmt.Println("dropping entry from: ", G.Last1000[0].AuditResult.OurTime)
			G.Last1000 = G.Last1000[1:]
			G.Last1000Serialized = G.Last1000Serialized[1:]
		}

		// Write it to a swap file and copy it over atomically
		b, _ := json.Marshal(G)
		ioutil.WriteFile(gp+".tmp", b, 0644)
		os.Rename(gp+".tmp", gp)
	}
}
