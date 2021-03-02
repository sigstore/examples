# Rekor Timestamper

This directory contains a small Go app that monitors a Rekor log and adds trusted timestamps to it.

The basic flow is:

* Generate a private/public keypair
* Every ten minutes:
  * Get the latest STH from Rekor
  * Perform a consistency proof
  * Generate a JSON struct indicating that we performed this "audit" at this time.
  * Get that "audit" struct timestamped by an RFC3161 TSA.
  * Sign this timestamped audit result with our private key.
  * Add this as an entry back into Rekor.

Additionally: 
* We serve our public key over http at /keys
* We store the last 1000 audits and serve them at /entries

This is mainly done for educational and illustrative purposes, but also shows the power/flexibility of a
Signature Transparency log.

The Rekor log has log entries every 10 minutes that contain third-party attestations of time.
If you do not trust the Rekor system clock, and want to get some form of proof that entries in the log were
created at or before a certain time, these entries can strengthen that claim.

# Operation

This binary is designed to run on Kubernetes.
It attempts to self-recover from restarts by caching some state locally on a PV.

Deploy it to any cluster with: `ko apply -f config/` from this directory.

Port-forward with: `kubectl port-forward deployment/timestamper 8080:8080`

View logs with: `kubectl logs -f deployment/timestamper`.
