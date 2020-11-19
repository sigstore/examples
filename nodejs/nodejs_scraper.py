import argparse
import base64
import datetime
import errno
import gnupg
import hashlib
import shutil
import json
import logging
import posixpath
import os
import pathlib
import requests
import sys
import time
import threading
from pathlib import Path
from urllib.parse import urlparse
from urllib.request import urlopen
from bs4 import BeautifulSoup

hd = os.path.expanduser('~')
arch_dir = os.path.join(hd, '.nodejs-files/')
rekor_dir = os.path.join(arch_dir, 'rekor-files/')
tracker_file = (os.path.join(arch_dir + ".monitor_tracker.txt"))
pub_ring = (os.path.join(arch_dir + "pubring.kbx"))
rekor_url = "http://localhost:3000/api/v1/add"
# release_url = "http://0.0.0.0:8000/"
# latest_url = "http://0.0.0.0:8000/latest"
release_url = "https://nodejs.org/download/release/"
latest_url = "https://nodejs.org/download/release/latest"

logging.basicConfig(level=logging.INFO)

parser = argparse.ArgumentParser(description='node.js rekor monitor.')

parser.add_argument(
    '--download_archive',
    action='store_true',
    help='Download the entire archive'
)

parser.add_argument(
    '--post_archive',
    action='store_true',
    help='Post the entire archive to rekor'
)

parser.add_argument(
    '--monitor',
    action='store_true',
    help='Monitor node.js and when an update is made, pull and post to rekor'
)

args = parser.parse_args()

headers = {
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Methods': 'GET',
    'Access-Control-Allow-Headers': 'Content-Type',
    'Access-Control-Max-Age': '3600',
    'User-Agent': 'Mozilla/5.0 (X11; Linux x86_64; rv:82.0) Gecko/20100101 Firefox/82.0'
}


def get_pub_keys(gpg):
    logging.info("Downloading Public keys (this might take a while)..")
    import_result = gpg.recv_keys(
        'pool.sks-keyservers.net',
        '4ED778F539E3634C779C87C6D7062848A1AB005C',
        '94AE36675C464D64BAFA68DD7434390BDBE9B9C5',
        '1C050899334244A8AF75E53792EF661D867B9DFA',
        '71DCFD284A79C3B38668286BC97EC7A07EDE3FC1',
        '8FCCA13FEF1D0C2E91008E09770F7A9A5AE15600',
        'C4F0DFFF4E8C1A8236409D08E73BC641CC11F4C8',
        'C82FA3AE1CBEDC6BE46B9360C43CEC45C17AB93C',
        'DD8F2338BAE7501E3DD5AC78C273792F7D83545D',
        'A48C2BEE680E841632CD4E44F07496B3EB3C1762',
        '108F52B48DB57BB0CC439B2997B01419BD92F80A',
        'B9E2F5981AA6E0CD28160D9FF13993A75599653C',
        '9554F04D7259F04124DE6B476D5A82AC7E37093B',
        'B9AE9905FFD7803F25714661B63B535A4C206CA9',
        '77984A986EBC2AA786BC0F66B01FBB92821C587A',
        '93C7E9E91B49E432C2F75674B0A78B0A6C481CF6',
        '56730D5401028683275BD23C23EFEFE93C4CFFFE',
        'FD3A5288F042B6850C66B31F09FE44734EB7990E',
        '114F43EE0176B71C7BC219DD50A3051F888C628D',
        '7937DFD2AB06298B2293C3187D33FF9D0246406D',
    )
    return(import_result)


def monitor_site(poll_interval):
    pass


def post_arch():
    # TODO check if we have the pub key
    gpg = gnupg.GPG(gnupghome='/tmp/.gnupg')

    if not os.path.isdir(arch_dir):
        logging.error(
            f"{arch_dir} does not exist, you likely need to run --download_archive first")
        sys.exit(1)
    try:
        os.mkdir(rekor_dir)
    except OSError as e:
        if e.errno != errno.EEXIST:
            raise

    gpg = gnupg.GPG(gnupghome=(arch_dir))
    if not os.path.isfile(pub_ring):
        maintainer_keys = get_pub_keys(gpg)
        logging.info(f"Imported {maintainer_keys.count} maintainer keys")
    encoded = None
    artifact_url = None
    artifact_hash = None
    signature = None
    public_key = None
    user_name = None
    for subdir, dirs, files in os.walk(arch_dir):
        for file in files:
            node_release = subdir.split("/")[-1]
            if "SHASUMS256" in file:
                stream = open((os.path.join(subdir, file)), 'rb')
                data = stream.read()
                if file.endswith(".txt"):
                    artifact_url = os.path.join(
                        release_url, node_release, file)
                    artifact_hash = hashlib.sha256(data).hexdigest()
                elif file.endswith(".sig"):
                    signature = base64.b64encode(data).decode("utf-8")
                elif file.endswith(".asc"):
                    file_path = os.path.join(subdir, file)
                    stream = open(file_path, "rb")
                    verified = gpg.verify_file(stream)
                    ascii_armored_public_keys = gpg.export_keys(
                        verified.key_id)
                    message_bytes = ascii_armored_public_keys.encode('ascii')
                    base64_bytes = base64.b64encode(message_bytes)
                    public_key = base64_bytes.decode('ascii')
                else:
                    pass
            if file.endswith(".sig"):
                target_dir = os.path.join(rekor_dir, node_release)
                if not os.path.isdir(target_dir):
                    try:
                        os.mkdir(target_dir)
                    except OSError as e:
                        if e.errno != errno.EEXIST:
                            raise
                rekor_json = {'URL': artifact_url, "SHA": artifact_hash,
                              "PublicKey": public_key, "Signature": signature}
                rekor_file = os.path.join(target_dir, artifact_hash + ".json")
                # write JSON out to the file
                json.dump(rekor_json, open(rekor_file, "w"))
                # post to rekor
                logging.info(f"Posting rekor file: {rekor_file}")
                file_dict = {"fileupload": (open(rekor_file, "rb"))}
                response = requests.post(
                    rekor_url, files=file_dict)
                logging.info(response.text)


def polling_system():
    logging.info(time.ctime())
    threading.Timer(10, polling_system).start()
    response = urlopen(latest_url).read()
    currentHash = hashlib.sha224(response).hexdigest()
    with open(tracker_file) as read_tracker:
        currentState = read_tracker.read()
    if currentHash == currentState:
        logging.info("No changes detected")
    else:
        logging.info("Site update detected")
        logging.info("Refreshing archive")
        with open(tracker_file, "w") as write_tracker:
            write_tracker.write(currentHash)
            write_tracker.close()
        try:
            latest = os.path.join(arch_dir + "latest")
            shutil.rmtree(latest)
        except OSError as e:
            logging.info(
                f"Error: removing folder: {e.filename} - {e.strerror}.")
        logging.info("Grabbing latest files")
        download_arch()
        logging.info("Posting refreshed files")
        post_arch()
        # Update tracking file


def monitor():
    if not os.path.isdir(rekor_dir):
        logging.error(
            f"{rekor_dir} does not exist, you likely need to run --download_archive and --post-archive first")
        sys.exit(1)

    if not os.path.isfile(tracker_file):
        logging.info("Making first tracking record.")
        response = urlopen(latest_url).read()
        siteHash = hashlib.sha224(response).hexdigest()
        tracker_write = open(tracker_file, 'w')
        tracker_write.write(siteHash)
        tracker_write.close()
    logging.info(f"Polling latest: {latest_url}")
    polling_system()


def download_arch():
    logging.info(f"Downloading latest files to: {arch_dir}")
    try:
        os.mkdir(arch_dir)
    except OSError as e:
        if e.errno != errno.EEXIST:
            raise
    req = requests.get(release_url, headers)
    soup = BeautifulSoup(req.content, 'html.parser')

    for a in soup.find_all('a', href=True):
        dirpath = posixpath.dirname(urlparse(a['href']).path)
        if dirpath and dirpath != '/':
            if "latest" in dirpath:
                logging.info(f"Downloading materials from release: {dirpath}")
                pathlib.Path(
                    arch_dir + dirpath).mkdir(parents=True, exist_ok=True)
                sha256_url = (release_url + dirpath + "/SHASUMS256.txt")
                sha256asc_url = (release_url + dirpath + "/SHASUMS256.txt.asc")
                sha256sig_url = (release_url + dirpath + "/SHASUMS256.txt.sig")

                sha256_resp = requests.get(sha256_url,
                                           allow_redirects=True)
                open(arch_dir + dirpath + "/SHASUMS256.txt",
                     'wb').write(sha256_resp.content)

                sha256asc_resp = requests.get(sha256asc_url,
                                              allow_redirects=True)
                open(arch_dir + dirpath + "/SHASUMS256.txt.asc",
                     'wb').write(sha256asc_resp.content)

                sha256sig_resp = requests.get(sha256sig_url,
                                              allow_redirects=True)
                open(arch_dir + dirpath + "/SHASUMS256.txt.sig",
                     'wb').write(sha256sig_resp.content)


if __name__ == "__main__":
    if args.download_archive:
        download_arch()
    elif args.post_archive:
        post_arch()
    elif args.monitor:
        monitor()
