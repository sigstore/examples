package main

import "time"

// The Auditor watches a Rekor t-log continuously.
// Once an hour or so, it checks the latest signed tree hash and
// performs a consistency proof (making sure the log has only been appended-to since
// the last run). It then writes the results of this audit into an AuditResult message,
// including the time of this audit.

type auditResult struct {
	OurTime               time.Time
	CurrentIndex          int64
	CurrentSignedTreeHash []byte
}

// Then the auditor gets this result timestamped by a TSA (freetsa.org), and bundles this into a final message:

type timestampedAuditResult struct {
	AuditResult     auditResult
	Timestamp       []byte
	TimestampServer string
}

// This final message is marshalled to JSON and signed by the auditor's own private key.
// This message, signature and public key are then uploaded into Rekor itself.
