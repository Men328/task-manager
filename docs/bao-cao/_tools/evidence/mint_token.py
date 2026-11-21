"""Tạo session JWT (HS256) cho 1 profile trên máy local, giống identity phát hành.

python mint_token.py <profile_id> <email> <out_file>
Đọc SESSION_SECRET từ deployments/docker/.env; KHÔNG in secret hay token ra màn hình.
"""
import base64, hashlib, hmac, json, sys, time
from pathlib import Path

ENV = Path(__file__).resolve().parents[4] / "deployments/docker/.env"


def b64(data: bytes) -> str:
    return base64.urlsafe_b64encode(data).rstrip(b"=").decode()


def secret() -> bytes:
    for line in ENV.read_text().splitlines():
        if line.startswith("SESSION_SECRET="):
            return line.split("=", 1)[1].strip().strip('"').strip("'").encode()
    raise SystemExit("SESSION_SECRET not found")


profile_id, email, out = sys.argv[1:4]
now = int(time.time())
header = {"alg": "HS256", "typ": "JWT"}
claims = {"email": email, "iss": "taskmanager-identity", "sub": profile_id, "iat": now, "exp": now + 7 * 24 * 3600}
signing_input = b64(json.dumps(header, separators=(",", ":")).encode()) + "." + b64(json.dumps(claims, separators=(",", ":")).encode())
sig = hmac.new(secret(), signing_input.encode(), hashlib.sha256).digest()
Path(out).write_text(signing_input + "." + b64(sig))
print("token written:", out)
