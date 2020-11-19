# Node JS Scrapper

This script will pull down the latest release folders from https://nodejs.org/download/release/
parse out the GPG values (signature, public key) and add them along with the SHA and
URL to a rekor file. It will then load those entries into the Rekor transparency log.

The script is run in three parts:

`python nodejs_scraper.py --download_archive`

This downloads all signing artifacts from node.js

`python nodejs_scraper.py --post_archive`

Posts all downloaded signing artifacts to the rekor server

`python nodejs_scraper.py --monitor`

Polls the server for changes to the `latest/release` folder on nodejs.org
If a change is found, it will re-download the release folder and post new
entries to rekor.