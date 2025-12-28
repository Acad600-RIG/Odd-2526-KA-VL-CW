import os
os.environ.setdefault("CUDA_VISIBLE_DEVICES", "-1")  # Force CPU-only runtime
os.environ.setdefault("TF_CPP_MIN_LOG_LEVEL", "2")   # Reduce TF logging (errors only)

from fastapi import FastAPI, File, UploadFile, HTTPException, Form
from fastapi.middleware.cors import CORSMiddleware
from deepface import DeepFace
import numpy as np
import cv2
import uvicorn
from typing import Dict
import json
import numpy as np
import re

# Extra safety: disable TF GPU from API if present
try:
    import tensorflow as tf
    try:
        tf.config.set_visible_devices([], 'GPU')
    except Exception:
        pass
except Exception:
    tf = None


app = FastAPI()


app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

INITIAL_PATTERN = re.compile(r"^[A-Z]{2}\d{2}-[1-2]$")


def normalize_and_validate_initial(value: str) -> str:
    normalized = value.strip().upper()
    if not INITIAL_PATTERN.match(normalized):
        raise HTTPException(status_code=422, detail="Initial must match pattern AA00-1 or AA00-2")
    return normalized


def numpy_to_list(embedding):
    if isinstance(embedding, dict):
        # If it's a dictionary, assume it's the DeepFace output and extract 'embedding'
        return np.array(embedding.get('embedding', [])).tolist()
    elif isinstance(embedding, list):
        return embedding
    elif isinstance(embedding, np.ndarray):
        return embedding.tolist()
    return [] # Return empty list for unexpected types

def list_to_numpy(embedding_list):
    return np.array(embedding_list)


def load_embeddings():
    try:
        with open('face_embeddings.json', 'r') as f:
            raw_embeddings = json.load(f)
            # Ensure all loaded embeddings are in the correct list format
            cleaned_embeddings = {}
            for initial, embedding_data in raw_embeddings.items():
                cleaned_embeddings[initial] = numpy_to_list(embedding_data)
            return cleaned_embeddings
    except FileNotFoundError:
        return {}

face_embeddings = load_embeddings()

@app.get("/check-registration/{initial}")
async def check_registration(initial: str):
    normalized_initial = normalize_and_validate_initial(initial)
    print(f"Checking registration for initial: {normalized_initial}")
    if normalized_initial in face_embeddings:
        return {"status": "registered"}
    else:
        return {"status": "unregistered"}

@app.post("/register-face")
async def register_face(
    initial: str = Form(...),
    file: UploadFile = File(...)
):
    try:
        normalized_initial = normalize_and_validate_initial(initial)
        print(f"Registering face for initial: {normalized_initial}")

        if not file.content_type.startswith('image/'):
            raise HTTPException(status_code=422, detail="File must be an image")
        
        contents = await file.read()
        # Use frombuffer (fromstring is deprecated for binary data)
        nparr = np.frombuffer(contents, dtype=np.uint8)
        img = cv2.imdecode(nparr, cv2.IMREAD_COLOR)
        
        if img is None:
            raise HTTPException(status_code=422, detail="Invalid image data")

        embedding_data = DeepFace.represent(img, model_name="Facenet")[0]
        embedding = numpy_to_list(embedding_data)
        
        face_embeddings[normalized_initial] = embedding
        
        with open('face_embeddings.json', 'w') as f:
            json.dump(face_embeddings, f)
        
        return {"status": "success", "message": "Face registered successfully"}
        
    except Exception as e:
        raise HTTPException(status_code=422, detail=str(e))

@app.post("/verify-face")
async def verify_face(
    file: UploadFile = File(...)
):
    try:
        if not face_embeddings:
            raise HTTPException(status_code=404, detail="No faces registered in the system")
        
        contents = await file.read()
        nparr = np.frombuffer(contents, np.uint8)
        img = cv2.imdecode(nparr, cv2.IMREAD_COLOR)
        
        if img is None:
            raise HTTPException(status_code=422, detail="Invalid image data")
        
        is_live = check_liveness(img)
        if not is_live:
            return {"status": "error", "message": "Liveness check failed - eyes not detected"}
        
        current_embedding_data = DeepFace.represent(img, model_name="Facenet")[0]
        current_embedding = numpy_to_list(current_embedding_data) # Ensure it's a list
        
        best_match = None
        highest_similarity = 0
        threshold = 0.7
        
        for initial, stored_embedding in face_embeddings.items():
            # stored_embedding is already a list due to load_embeddings cleaning
            similarity = cosine_similarity(np.array(stored_embedding), np.array(current_embedding))
            
            print(f"Similarity with {initial}: {similarity}")
            
            if similarity > highest_similarity:
                highest_similarity = similarity
                best_match = initial
        
        if highest_similarity >= threshold:
            return {
                "status": "success", 
                "message": "Face verified successfully", 
                "initial": best_match,
                "similarity": float(highest_similarity)
            }
        else:
            return {
                "status": "failed", 
                "message": "No matching face found", 
                "similarity": float(highest_similarity) if highest_similarity > 0 else 0
            }
            
    except Exception as e:
        print("Error in verify_face:", str(e))
        raise HTTPException(
            status_code=500,
            detail=f"Error verifying face: {str(e)}"
        )

        
def check_liveness(image):
    try:
        cascade_path = cv2.data.haarcascades + 'haarcascade_eye.xml'
        print(f"Attempting to load eye cascade from: {cascade_path}")
        eye_cascade = cv2.CascadeClassifier(cascade_path)
        if eye_cascade.empty():
            raise Exception(f"Failed to load eye cascade classifier from {cascade_path}")
            
        # Save the input image for debugging
        cv2.imwrite("debug_liveness_input.jpg", image)
        print("Saved debug_liveness_input.jpg")

        gray = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)
        eyes = eye_cascade.detectMultiScale(
            gray,
            scaleFactor=1.1,
            minNeighbors=3, # Relaxed from 5 to 3
            minSize=(20, 20) # Relaxed from 30,30 to 20,20
        )
        
        print(f"Detected {len(eyes)} eyes.")

        # Draw rectangles around detected eyes for debugging
        debug_image = image.copy()
        for (x, y, w, h) in eyes:
            cv2.rectangle(debug_image, (x, y), (x+w, y+h), (0, 255, 0), 2)
        cv2.imwrite("debug_liveness_output.jpg", debug_image)
        print("Saved debug_liveness_output.jpg with detected eyes.")

        return len(eyes) >= 1
    except Exception as e:
        print(f"Liveness check failed: {str(e)}")
        return False # Return False on liveness check failure
    

def cosine_similarity(embedding1, embedding2):
    return np.dot(embedding1, embedding2) / (np.linalg.norm(embedding1) * np.linalg.norm(embedding2))

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)