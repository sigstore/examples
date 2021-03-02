package main

import (
	"bytes"
	"crypto"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	tclient "github.com/google/trillian/client"
	tcrypto "github.com/google/trillian/crypto"

	rekord_v001 "github.com/sigstore/rekor/pkg/types/rekord/v0.0.1"

	"github.com/digitorus/timestamp"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/google/trillian"
	"github.com/google/trillian/merkle/rfc6962"
	"github.com/sigstore/rekor/pkg/generated/client/entries"
	"github.com/sigstore/rekor/pkg/generated/models"
)

func upload(payload, sig []byte) (models.LogEntry, error) {
	re := rekord_v001.V001Entry{
		RekordObj: models.RekordV001Schema{
			Data: &models.RekordV001SchemaData{
				Content: strfmt.Base64(payload),
			},
			Signature: &models.RekordV001SchemaSignature{
				Content: strfmt.Base64(sig),
				Format:  models.RekordV001SchemaSignatureFormatX509,
				PublicKey: &models.RekordV001SchemaSignaturePublicKey{
					Content: strfmt.Base64(G.Pub),
				},
			},
		},
	}
	returnVal := models.Rekord{
		APIVersion: swag.String(re.APIVersion()),
		Spec:       re.RekordObj,
	}
	params := entries.NewCreateLogEntryParams()
	params.SetProposedEntry(&returnVal)
	r, err := G.rekorClient.Entries.CreateLogEntry(params)
	if err != nil {
		return nil, err
	}
	return r.Payload, nil

}

func stamp(ar auditResult) (*timestampedAuditResult, error) {
	b, _ := json.Marshal(ar)
	tsq, err := timestamp.CreateRequest(bytes.NewReader(b), &timestamp.RequestOptions{
		Hash:         crypto.SHA256,
		Certificates: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	tsr, err := http.Post(tsServer, "application/timestamp-query", bytes.NewReader(tsq))
	if err != nil {
		return nil, err
	}

	if tsr.StatusCode > 200 {
		return nil, fmt.Errorf("unexpected status: %s", tsr.Status)
	}

	resp, err := ioutil.ReadAll(tsr.Body)
	if err != nil {
		return nil, err
	}
	return &timestampedAuditResult{
		AuditResult:     ar,
		Timestamp:       resp,
		TimestampServer: tsServer,
	}, nil
}

func sign(i interface{}) ([]byte, []byte) {
	payload, _ := json.Marshal(i)
	sig := ed25519.Sign(G.Priv, payload)
	return sig, payload
}

func runAudit() (*timestampedAuditResult, error) {
	result, err := G.rekorClient.Tlog.GetLogInfo(nil)
	if err != nil {
		return nil, err
	}

	logInfo := result.GetPayload()

	mustDecode := func(s string) []byte {
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			panic(err)
		}
		return b
	}
	sth := trillian.SignedLogRoot{
		KeyHint:          mustDecode(logInfo.SignedTreeHead.KeyHint.String()),
		LogRoot:          mustDecode(logInfo.SignedTreeHead.LogRoot.String()),
		LogRootSignature: mustDecode(logInfo.SignedTreeHead.Signature.String()),
	}

	verifier := tclient.NewLogVerifier(rfc6962.DefaultHasher, G.rekorPublicKey, crypto.SHA256)
	lr, err := tcrypto.VerifySignedLogRoot(verifier.PubKey, verifier.SigHash, &sth)
	if err != nil {
		return nil, err
	}

	ar := auditResult{
		OurTime:               time.Now(),
		CurrentIndex:          int64(lr.TreeSize),
		CurrentSignedTreeHash: lr.RootHash,
	}

	// Get this timestamped
	stampedResult, err := stamp(ar)
	if err != nil {
		return nil, err
	}

	// Sign this!
	sig, payload := sign(stampedResult)

	// Now upload to the log!
	resp, err := upload(payload, sig)

	for k, v := range resp {
		log.Printf("added log entry uuid=%s index=%d", k, *v.LogIndex)
	}
	return stampedResult, nil
}
