from hashlib import sha256
import sys
import hashlib

m = hashlib.sha256()

def main():
    args = sys.argv
    argC = size(sys.argv)

def getFileChecksum(filePath: str) -> str:
    data: bytes
    with open(filePath, "rb") as f:
        data = f.read()
    return sha256(data).hexdigest()
