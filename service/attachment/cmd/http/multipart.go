package main

import (
	"context"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"

	"taskmanager/common/errorcode"
	attachmentv1 "taskmanager/common/gen/go/attachment/v1"
	"taskmanager/service/attachment/internal/config"
)

const (
	uploadPath        = "/v1/attachments/upload"
	downloadPrefix    = "/v1/attachments/"
	downloadSuffix    = "/download"
	multipartOverhead = 1 << 20
	requestTimeout    = 60 * time.Second
)

type attachmentHTTPHandler struct {
	mux      *runtime.ServeMux
	client   attachmentv1.AttachmentServiceClient
	maxBytes int64
}

func newAttachmentHTTPHandler(mux *runtime.ServeMux, conn *grpc.ClientConn, cfg config.Config) *attachmentHTTPHandler {
	maxBytes := cfg.MaxAttachmentBytes
	if maxBytes <= 0 {
		maxBytes = 10 << 20
	}
	return &attachmentHTTPHandler{
		mux:      mux,
		client:   attachmentv1.NewAttachmentServiceClient(conn),
		maxBytes: maxBytes,
	}
}

func (h *attachmentHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == uploadPath:
		h.handleUpload(w, r)
	case r.Method == http.MethodGet && isDownloadPath(r.URL.Path):
		h.handleDownload(w, r)
	default:
		h.mux.ServeHTTP(w, r)
	}
}

func isDownloadPath(path string) bool {
	if !strings.HasPrefix(path, downloadPrefix) || !strings.HasSuffix(path, downloadSuffix) {
		return false
	}
	id := strings.TrimSuffix(strings.TrimPrefix(path, downloadPrefix), downloadSuffix)
	return id != "" && !strings.Contains(id, "/")
}

func (h *attachmentHTTPHandler) handleUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.maxBytes+multipartOverhead)
	if err := r.ParseMultipartForm(h.maxBytes + multipartOverhead); err != nil {
		h.writeError(w, r, errorcode.Error(errorcode.AttachmentFileTooLarge))
		return
	}

	var content []byte
	var fileName string
	var contentType string

	file, header, err := r.FormFile("file")
	if err == nil {
		defer func() { _ = file.Close() }()
		content, err = io.ReadAll(io.LimitReader(file, h.maxBytes+1))
		if err != nil {
			h.writeError(w, r, errorcode.Error(errorcode.AttachmentFileRequired))
			return
		}
		fileName = header.Filename
		contentType = header.Header.Get("Content-Type")
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	resp, err := h.client.UploadAttachment(ctx, &attachmentv1.UploadAttachmentRequest{
		ProfileId:   r.FormValue("profile_id"),
		OwnerType:   r.FormValue("owner_type"),
		OwnerId:     r.FormValue("owner_id"),
		FileName:    fileName,
		ContentType: contentType,
		Content:     content,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	body, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(resp.GetAttachment())
	if err != nil {
		h.writeError(w, r, errorcode.Error(errorcode.CommonInternal))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (h *attachmentHTTPHandler) handleDownload(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, downloadPrefix), downloadSuffix)

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	resp, err := h.client.GetAttachment(ctx, &attachmentv1.GetAttachmentRequest{Id: id})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	item := resp.GetAttachment()
	contentType := item.GetContentType()
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", contentDisposition(item.GetFileName()))
	w.Header().Set("Content-Length", strconv.Itoa(len(resp.GetContent())))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=0, must-revalidate")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp.GetContent())
}

func contentDisposition(fileName string) string {
	name := strings.TrimSpace(fileName)
	if name == "" {
		name = "attachment"
	}
	value := mime.FormatMediaType("attachment", map[string]string{"filename": name})
	if value == "" {
		return "attachment"
	}
	return value
}

func (h *attachmentHTTPHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	runtime.DefaultHTTPErrorHandler(
		r.Context(),
		h.mux,
		&runtime.JSONPb{MarshalOptions: protojson.MarshalOptions{EmitUnpopulated: true}},
		w,
		r,
		err,
	)
}
