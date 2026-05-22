from hashlib import sha256

def get256String(filePath: str) -> str:
    fileB: bytes
    with open(filePath, "rb") as f:
        fileB = f.read()
    return sha256(fileB).hexdigest()

def main():
    testFiles: str = "./.test-files/"
    print(testFiles) # NOTE: silences unused warning

if __name__ == "__main__":
    main()
