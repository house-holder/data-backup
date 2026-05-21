import sys
from hashlib import sha256

def main():
    for arg in sys.argv[1:]:
        print(getFileChecksum(arg))

def getFileChecksum(filePath: str) -> str:
    h = sha256()
    with open(filePath, "rb") as f:
        for chunk in iter(lambda: f.read(8192), b""):
            h.update(chunk)
    return h.hexdigest()

if __name__ == "__main__":
    main()
