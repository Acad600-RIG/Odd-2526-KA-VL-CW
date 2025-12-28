# Face Recognition API

Minimal FastAPI service for registering and verifying faces using DeepFace embeddings and a simple liveness eye-detection check.

## Requirements
- Python 3.9+ (64-bit recommended)
- Pip packages:
  - fastapi
  - uvicorn[standard]
  - deepface
  - opencv-python
  - numpy
  - tensorflow-cpu (CPU-only) or tensorflow (GPU), though GPU is disabled in code

Install packages (from the project folder):
```bash
python -m venv .venv
./.venv/Scripts/Activate.ps1  # PowerShell
pip install -r requirements.txt
```

## Run the API
From the `Face Recognition` folder:
```bash
uvicorn main:app --host 0.0.0.0 --port 8000 --reload
```

## Initial format
- Pattern: two uppercase letters, two digits, dash, and a run number 1 or 2.
- Regex: `^[A-Z]{2}\d{2}-[1-2]$`
- Examples: `KA24-1`, `VL24-1`, `CW23-2`

## Endpoints
- `POST /register-face` (multipart form)
  - Fields: `initial` (string, required), `file` (image, required)
  - Stores embedding in `face_embeddings.json` keyed by the validated initial.
- `GET /check-registration/{initial}`
  - Returns whether the initial is already registered.
- `POST /verify-face` (multipart form)
  - Field: `file` (image, required)
  - Returns best match initial and similarity if above threshold, else failure.

## Quick cURL examples
Register a face:
```bash
curl -X POST "http://localhost:8000/register-face" \
  -F "initial=KA24-1" \
  -F "file=@/path/to/face.jpg"
```

Check registration:
```bash
curl "http://localhost:8000/check-registration/KA24-1"
```

Verify a face:
```bash
curl -X POST "http://localhost:8000/verify-face" \
  -F "file=@/path/to/face.jpg"
```

## Data storage
- Embeddings persist in `face_embeddings.json` in the project directory.
- Debug images for liveness (`debug_liveness_input.jpg`, `debug_liveness_output.jpg`) are written alongside the app when liveness is checked.

## Notes
- GPU is explicitly disabled in code; TensorFlow will run on CPU.
- Only images are accepted; non-image uploads return HTTP 422.
