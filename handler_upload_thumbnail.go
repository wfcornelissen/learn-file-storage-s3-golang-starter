package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20 //10MB
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, 501, "Error with multipart parsing.", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, 501, "Error getting file data", err)
		return
	}
	defer file.Close()

	var ext string
	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	switch mediaType {
	case "image/jpeg":
		ext = ".jpeg"
	case "image/png":
		ext = ".png"
	default:
		respondWithError(w, http.StatusBadRequest, "Invalid file type", err)
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video", err)
		return
	}
	if video.CreateVideoParams.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You are not the owner", nil)
		return
	}

	key := make([]byte, 32)
	rand.Read(key)
	videoIDString = base64.RawURLEncoding.EncodeToString(key)
	filePath := filepath.Join(cfg.assetsRoot, videoIDString+ext)

	fileOnDisk, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create file.", err)
		return
	}
	defer fileOnDisk.Close()

	_, err = io.Copy(fileOnDisk, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't write data to file", err)
		return
	}

	thumbnailURL := "http://localhost:" + cfg.port + "/assets/" + videoIDString + ext
	video.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
