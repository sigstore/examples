# Public Key Detector

This example shows how to run a Rekor monitor to watch the log for uses of a specific
public key.

## Overview

The code starts tailing the log at the latest log index.
Every 15m it checks the log for new entries and inspects them.
If the specified public key is found in the log, the program will log "KEY FOUND" and the index
your key was found at.

You can run the program locally with:

```
go run . <path to public key>
```

## Sending Emails

To do so, set the SENDGRID_API_KEY environment variable, and suply an eamil
You can configure this program to send emails when a key is found.

To do so, pass the email address to send emails to as well as the public key:

```
export SENDGRID_API_KEY=<API KEY HERE>
go run . <path to public key> <email>
```

## K8s

You can deploy this in k8s with `ko` as well!
First setup the email, key and SENDGRID_API_KEY as a secret:

```
kubectl create secret generic emailer --from-file=pub=cosign.pub --from-literal=sendgrid_api_key=$SENDGRID_API_KEY --from-literal=email=<your email>
```

Then:

`ko apply -f config/`
