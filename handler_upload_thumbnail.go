package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "ParseMultipartForm() error", err)
		return
	}

	// "thumbnail" should match the HTML form input name
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	ct := header.Header.Get("Content-Type")

	data, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to read thumbnail file content", err)
		return
	}

	vid, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "GetVideo() error", err)
		return
	}
	if vid.UserID != userID {
		respondWithJSON(w, http.StatusUnauthorized, struct{}{})
		return
	}

	videoThumbnails[vid.ID] = thumbnail{
		data:      data,
		mediaType: ct,
	}

	// CH1 L7
	// TODO: assuming header is well formed
	// fmt.Println(ct)
	ext := strings.Split(ct, "/")[1]
	imgFile := fmt.Sprintf("%s.%s", vid.ID, ext)
	imgPath := filepath.Join(cfg.assetsRoot, imgFile)
	// fmt.Println(imgPath)
	f, err := os.Create(imgPath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "os.Create() error", err)
		return
	}
	defer f.Close()
	// TODO: supposed to use io.Copy(), but that does not make sense, also didn't work
	// we already read the multipart form into data[], so why read it again
	// using Copy() would assume the client and server are on the same machine
	f.Write(data)
	// TODO: not sure how to do this without hardcode, ASSETS_ROOT is no good
	// seems like cfg should have a root url
	url := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, imgFile)
	vid.ThumbnailURL = &url
	err = cfg.db.UpdateVideo(vid)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "UpdateVideo() error", err)
		return
	}

	respondWithJSON(w, http.StatusOK, vid)
}
