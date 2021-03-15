package main

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/sendgrid/sendgrid-go"

	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"github.com/go-openapi/runtime"
	"github.com/google/trillian"
	tclient "github.com/google/trillian/client"
	tcrypto "github.com/google/trillian/crypto"
	"github.com/google/trillian/merkle/rfc6962/hasher"

	"github.com/sigstore/rekor/cmd/cli/app"
	"github.com/sigstore/rekor/pkg/generated/client"
	"github.com/sigstore/rekor/pkg/generated/client/entries"
	"github.com/sigstore/rekor/pkg/generated/models"
	"github.com/sigstore/rekor/pkg/types"
	"github.com/sigstore/rekor/pkg/types/rekord"
	rekord_v001 "github.com/sigstore/rekor/pkg/types/rekord/v0.0.1"
	"github.com/sigstore/rekor/pkg/types/rpm"
	rpm_v001 "github.com/sigstore/rekor/pkg/types/rpm/v0.0.1"
)

const (
	rekorAddr = "https://api.rekor.dev"
)

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

func main() {
	watchKey, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	email := ""
	if len(os.Args) > 2 {
		email = os.Args[2]
	}
	if err := run(string(watchKey), email); err != nil {
		panic(err)
	}
}

func run(watchedKey, email string) error {
	rekorClient, err := app.GetRekorClient(rekorAddr)
	if err != nil {
		panic(err)
	}

	pluggableTypeMap := map[string]string{
		rekord.KIND: rekord_v001.APIVERSION,
		rpm.KIND:    rpm_v001.APIVERSION,
	}
	fmt.Println(pluggableTypeMap)

	pub := initRekorPublicKey(rekorClient)

	tick := time.NewTicker(30 * time.Second)

	size, err := currentSize(rekorClient, pub)
	if err != nil {
		return err
	}
	lastIndexSeen := size - 1

	for {
		<-tick.C

		log.Printf("performing check at index %d", lastIndexSeen)
		curSize, err := currentSize(rekorClient, pub)
		if err != nil {
			log.Println(err)
			continue
		}
		curLastIndex := curSize - 1

		for ; lastIndexSeen <= curLastIndex; lastIndexSeen++ {
			params := entries.NewGetLogEntryByIndexParams()
			params.LogIndex = int64(lastIndexSeen)
			resp, err := rekorClient.Entries.GetLogEntryByIndex(params)
			if err != nil {
				log.Println(err)
				continue
			}

			// the response is a map with one element, a uuid->resp mapping
			for _, p := range resp.Payload {
				b, err := base64.StdEncoding.DecodeString(p.Body.(string))
				if err != nil {
					log.Println(err)
					continue
				}
				pe, err := models.UnmarshalProposedEntry(bytes.NewBuffer(b), runtime.JSONConsumer())
				if err != nil {
					log.Println(err)
					continue
				}
				eimpl, err := types.NewEntry(pe)
				if err != nil {
					log.Println(err)
					continue
				}

				var pk string
				var hash string
				var sig string
				switch v := eimpl.(type) {
				case *rekord_v001.V001Entry:
					pk = string([]byte(v.RekordObj.Signature.PublicKey.Content))
					sig = base64.StdEncoding.EncodeToString([]byte(v.RekordObj.Signature.Content))
					hash = *v.RekordObj.Data.Hash.Value
				case *rpm_v001.V001Entry:
					pk = string([]byte(v.RPMModel.PublicKey.Content))
				default:
					fmt.Println("no type found")
				}

				if pk == watchedKey {
					fmt.Println("FOUND KEY! SENDING EMAIL!")
					if os.Getenv("SENDGRID_API_KEY") != "" {
						if err := sendEmail(email, lastIndexSeen, hash, sig); err != nil {
							log.Println(err)
							continue
						}
					}
				}
			}
		}
	}
}

func currentSize(c *client.Rekor, pub interface{}) (uint64, error) {
	verifier := tclient.NewLogVerifier(hasher.DefaultHasher, pub, crypto.SHA256)

	li, err := c.Tlog.GetLogInfo(nil)
	if err != nil {
		return 0, err
	}
	sth := trillian.SignedLogRoot{
		KeyHint:          []byte(*li.Payload.SignedTreeHead.KeyHint),
		LogRoot:          []byte(*li.Payload.SignedTreeHead.LogRoot),
		LogRootSignature: []byte(*li.Payload.SignedTreeHead.Signature),
	}
	lr, err := tcrypto.VerifySignedLogRoot(pub, verifier.SigHash, &sth)
	if err != nil {
		return 0, err
	}
	return lr.TreeSize, nil
}

func sendEmail(email string, index uint64, hash, sig string) error {
	from := mail.NewEmail("Sigstore Watcher", email)
	to := mail.NewEmail("", email)
	subject := "key usage detected by sigstore!"
	body := fmt.Sprintf(`ALERT!
Your public key was spotted by sigstore at index: %d in the Sigstore Signature Transparency Log.
The key signed hash %s for signature %s.

You can see more details with:

rekor-cli get --log-index %d

`, index, hash, sig, index)
	message := mail.NewSingleEmail(from, subject, to, body, "")
	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	response, err := client.Send(message)
	if err != nil {
		return err
	}
	fmt.Println(response)
	return nil
}
