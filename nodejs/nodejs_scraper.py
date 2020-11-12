import requests
import posixpath
import pathlib
import os
from urllib.parse import urlparse
from bs4 import BeautifulSoup


headers = {
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Methods': 'GET',
    'Access-Control-Allow-Headers': 'Content-Type',
    'Access-Control-Max-Age': '3600',
    'User-Agent': 'Mozilla/5.0 (X11; Linux x86_64; rv:82.0) Gecko/20100101 Firefox/82.0'
}

url = "https://nodejs.org/download/release/"
req = requests.get(url, headers)
soup = BeautifulSoup(req.content, 'html.parser')

for a in soup.find_all('a', href=True):
    dirpath = posixpath.dirname(urlparse(a['href']).path)
    if dirpath and dirpath != '/':
        if "latest" in dirpath:
            print("Downloading materials from release: ", dirpath)
            pathlib.Path(dirpath).mkdir(parents=True, exist_ok=True)
            sha256 = (url + dirpath + "/SHASUMS256.txt")
            sha256asc = (url + dirpath + "/SHASUMS256.txt.asc")
            sha256sig = (url + dirpath + "/SHASUMS256.txt.sig")

            sha256_resp = requests.get(sha256,
                                       allow_redirects=True)
            open(dirpath + "/SHASUMS256.txt", 'wb').write(sha256_resp.content)

            sha256asc_resp = requests.get(sha256asc,
                                          allow_redirects=True)
            open(dirpath + "/SHASUMS256.txt.asc",
                 'wb').write(sha256asc_resp.content)

            sha256sig_resp = requests.get(sha256sig,
                                          allow_redirects=True)
            open(dirpath + "/SHASUMS256.txt.sig",
                 'wb').write(sha256sig_resp.content)
