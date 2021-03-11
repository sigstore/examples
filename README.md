# Rekor examples

This repository contains example code and rekor populators (scripts that pull down information
from release sites and load them into rekor).

## Getting Started

One of the best ways to use Rekor directly is to publish signatures as part of your release process,
and instruct your users to check them against the log before trusting them.
This tutorial shows you can do that using PGP today.

### Setup

Create a PGP keypair (skip if you already have one):

```
$ gpg --generate-key
gpg (GnuPG) 2.2.25; Copyright (C) 2020 Free Software Foundation, Inc.
This is free software: you are free to change and redistribute it.
There is NO WARRANTY, to the extent permitted by law.

Note: Use "gpg --full-generate-key" for a full featured key generation dialog.

GnuPG needs to construct a user ID to identify your key.

Real name: Batman
Email address: lorenc.d@gmail.com
You selected this USER-ID:
    "Batman <lorenc.d@gmail.com>"

<some output not shown>

pub   rsa3072 2021-03-11 [SC] [expires: 2023-03-11]
      0FA5742ABA39275E6ACCB728E568D0F620485E9F
uid                      Batman <lorenc.d@gmail.com>
sub   rsa3072 2021-03-11 [E] [expires: 2023-03-11]
```

Get and build the `rekor-cli` from: [github.com/sigstore/rekor](github.com/sigstore/rekor).


### Build and Sign your release!

We'll build a small tar archive here for pretend (use your normal release artifacts instead)!

```
$ echo "i'm batman" > release
dlorenc-macbookpro:examples dlorenc$ tar -czf release
tar: no files or directories specified
dlorenc-macbookpro:examples dlorenc$ tar -czf release.tar.gz release
```

Sign it!
If you have multiple keys in your keyring, you'll need to pass the key id in the `-u` parameter from when
you created the key.
You can find it with `gpg --list-keys` if you forgot it.

```
gpg -u 0FA5742ABA39275E6ACCB728E568D0F620485E9F  --armor --output release.tar.gz.sig --detach-sig release.tar.gz
```

### Upload it to Rekor!

Now it's time to upload the signature to Rekor!

First we have to export the public key:

```shell
gpg -u 0FA5742ABA39275E6ACCB728E568D0F620485E9F  --export --armor > public.key
```

Then we can upload:
```shell
$ rekor-cli upload --artifact release.tar.gz --signature release.tar.gz.sig --public-key public.key
Created entry at index 1312, available at: https://api.rekor.dev/api/v1/log/entries/ca153cd0f350eecb028aeed298ef5bf74e8e7e5fa3d7f7d055ff51a0a0d489fd
```

If you remember the index and UUID, you can fetch the entry directly:

```shell
$ rekor-cli get --uuid ca153cd0f350eecb028aeed298ef5bf74e8e7e5fa3d7f7d055ff51a0a0d489fd
< a lot of output because the key is huge!>
```

With a bit more bash magic we can see some useful stuff.

The timestamp of when the entry was put in the log can be found with:

```shell
$ rekor-cli get --format=json --log-index 1312 | jq -r .IntegratedTime
```

This can be used as a third-party attestation that you possessed the data and signed it before this particular time
(as recorded and observed by the Rekor server).

You can also check the original hash:

```shell
$ rekor-cli get --format=json --log-index 1312 | jq -r .Body | base64 --decode | jq  -r .spec.data.hash.value
1aa3bc105b1272cc914ff9cb5dc01f1fdc46dcddb28d66ece3b42ba4e9fa0b3f
$ shasum -a 256 release.tar.gz
1aa3bc105b1272cc914ff9cb5dc01f1fdc46dcddb28d66ece3b42ba4e9fa0b3f  release.tar.gz
```

### Searching Rekor

If you lost the signature or can't find it, you can look it up too.
This is the part you should consider documenting for your users.
If you publish entries to the log, and they check the log before trusting them, Rekor will act as a log of all
signatures made by your private key.

If a signature appears in Rekor that you don't know about, or your users find a signature not in the log,
your key might have been compromised!

```shell
$ rekor-cli search --artifact release.tar.gz
Found matching entries (listed by UUID):
ca153cd0f350eecb028aeed298ef5bf74e8e7e5fa3d7f7d055ff51a0a0d489fd
```

We can then download the signature:

```shell
$ rekor-cli get --format=json --uuid ca153cd0f350eecb028aeed298ef5bf74e8e7e5fa3d7f7d055ff51a0a0d489fd | jq -r .Body | base64 --decode | jq -r .spec.signature.content | base64 --decode
-----BEGIN PGP SIGNATURE-----

iQGzBAABCAAdFiEED6V0Kro5J15qzLco5WjQ9iBIXp8FAmBKDLYACgkQ5WjQ9iBI
Xp/wNQv/XGG/WuDQM7Y3Fv5aA+Dqgl4MgxLPVQw0e0jc8cFOe14TyQtPFbzqMiqW
kZ5VH6sd5dSxX1s5yAuX8d12I/rQdp9C6JfW3cvQh7Hh3jVfqMjTNl860LapdXTX
u4tcXZGLgGjuk1hCk0RxL81g6C3aJYmiK5Dc0NqUVlu0+N9pLWUK6Gp3jisS6mhl
mPnOWAcFXPVMHhxwRox8xrnyOG9AKl6yPDznEptHOhNHq7nGsg1Bm7lWrUEYb+nO
9Pprs6WrhiEZgwdq3jfTUuuDtw0sJO7UNTaKy8fNCUjWd22P7EX6r4eUSS6+vXaz
hgQRQ+ShBfIGBypC0JMuDCvLFeifHQrWjHWM1ZN7b6YOJIoltSNCrHXOUCsZIQT5
fNfDInnwCf9Tu70Wo2ZXg8B29CySkefhzNJifP0za2PvBFFV+WltM5YnXTNIPfsQ
ci1kpBM0Zy8LRf+4jI4cRR5IpW/j90R5NAlxPUfAT2V6mUOdm09Z6X4o1MqwBldU
4b3zQbLS
=aDQ1
-----END PGP SIGNATURE-----
```

It matches the existing one!

```shell
$ diff <(rekor-cli get --format=json --uuid ca153cd0f350eecb028aeed298ef5bf74e8e7e5fa3d7f7d055ff51a0a0d489fd | jq -r .Body | base64 --decode | jq -r .spec.signature.content | base64 --decode) release.tar.gz.sig
```

And we can verify the signature with it:

```shell
$ gpg --verify release.tar.gz.sig  release.tar.gz
gpg: Signature made Thu Mar 11 06:27:34 2021 CST
gpg:                using RSA key 0FA5742ABA39275E6ACCB728E568D0F620485E9F
gpg: Good signature from "Batman <lorenc.d@gmail.com>" [ultimate]
```
