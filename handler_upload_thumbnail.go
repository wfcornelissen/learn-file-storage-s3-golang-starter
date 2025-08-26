package main

import (
	"fmt"
	"io"
	"net/http"

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

	// TODO: implement the upload here
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	fileData, fileHeaders, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldnt read from FormFile", err)
	}

	mediaType := fileHeaders.Header.Get("Content-Type")

	fileDataByte, err := io.ReadAll(fileData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error reading fileData", err)
	}

	videoMetadata, err := cfg.db.GetVideo(videoID)
	if videoMetadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
	}

	newThumb := thumbnail{
		data:      fileDataByte,
		mediaType: mediaType,
	}

	videoThumbnails[videoID] = newThumb

	thumbnailURL := fmt.Sprintf("http://localhost:8091/api/thumbnails/%s", videoID)
	videoMetadata.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update database", err)
	}

	respondWithJSON(w, http.StatusOK, videoMetadata)
}
