// Package storage stores private files — payment proofs and food photos.
//
// Everything lands in a PRIVATE bucket and is served only by presigned URL
// (99 §7). A payment proof is a photograph of somebody's bank account; a
// public bucket would put those on the open internet with a guessable path.
package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Errors the caller maps to messages.
var (
	ErrTooLarge    = errors.New("storage: file is too large")
	ErrUnsupported = errors.New("storage: unsupported file type")
	ErrEmpty       = errors.New("storage: file is empty")
)

// Config points at the object store.
type Config struct {
	Endpoint       string
	PublicEndpoint string
	AccessKey      string
	SecretKey      string
	Bucket         string
	UseSSL         bool
}

// Store is the object store adapter.
type Store struct {
	client *minio.Client
	cfg    Config
}

// New connects and ensures the bucket exists.
func New(ctx context.Context, cfg Config) (*Store, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: connect: %w", err)
	}

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("storage: bucket check: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("storage: create bucket: %w", err)
		}
	}
	// The bucket policy is left at its default — PRIVATE. Nothing here ever
	// makes it public, and a presigned URL is the only way out.
	return &Store{client: client, cfg: cfg}, nil
}

// Limits per kind of upload.
const (
	MaxProofBytes = 5 << 20 // PROMPT §10
	MaxPhotoBytes = 8 << 20
)

// allowedTypes maps a declared content type to the magic bytes that must
// actually be there.
//
// The EXTENSION is never trusted and neither is the declared type: both are
// attacker-controlled, and "photo.jpg" containing a script is the oldest trick
// there is. The bytes decide (99 §7).
var magic = map[string][]byte{
	"image/jpeg":      {0xFF, 0xD8, 0xFF},
	"image/png":       {0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
	"application/pdf": []byte("%PDF-"),
}

// DetectType sniffs the real type from the leading bytes, returning the
// canonical content type.
//
// WebP is checked separately because its signature is split: "RIFF" then four
// size bytes then "WEBP".
func DetectType(head []byte) (string, bool) {
	for ct, sig := range magic {
		if len(head) >= len(sig) && bytes.Equal(head[:len(sig)], sig) {
			return ct, true
		}
	}
	if len(head) >= 12 && bytes.Equal(head[0:4], []byte("RIFF")) &&
		bytes.Equal(head[8:12], []byte("WEBP")) {
		return "image/webp", true
	}
	return "", false
}

// Upload stores a file under a generated key.
type Upload struct {
	Key         string
	ContentType string
	Bytes       int64
}

// PutProof stores a payment proof.
//
// The key is GENERATED, never taken from the client: a client-supplied path is
// how an upload escapes its prefix or overwrites someone else's file. The
// customer id is in the path so an accidental listing is still attributable.
func (s *Store) PutProof(ctx context.Context, customerID, orderID uuid.UUID,
	r io.Reader, size int64) (Upload, error) {
	return s.put(ctx, r, size, MaxProofBytes,
		fmt.Sprintf("proofs/%s/%s/%s", time.Now().UTC().Format("2006/01"),
			customerID.String(), orderID.String()))
}

// PutFoodPhoto stores a dish photograph.
func (s *Store) PutFoodPhoto(ctx context.Context, foodID uuid.UUID,
	r io.Reader, size int64) (Upload, error) {
	return s.put(ctx, r, size, MaxPhotoBytes, "foods/"+foodID.String())
}

func (s *Store) put(ctx context.Context, r io.Reader, size, max int64, prefix string) (Upload, error) {
	if size <= 0 {
		return Upload{}, ErrEmpty
	}
	if size > max {
		return Upload{}, fmt.Errorf("%w: %d bytes, limit %d", ErrTooLarge, size, max)
	}

	// Read the head to sniff the type, then stream the rest — the whole file is
	// never held in memory, so a 5 MB upload costs 512 bytes of buffer.
	head := make([]byte, 512)
	n, err := io.ReadFull(r, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return Upload{}, fmt.Errorf("storage: read: %w", err)
	}
	head = head[:n]

	contentType, ok := DetectType(head)
	if !ok {
		return Upload{}, fmt.Errorf("%w: the file is not a JPEG, PNG, WebP or PDF", ErrUnsupported)
	}

	key := prefix + "/" + uuid.Must(uuid.NewV7()).String() + extensionFor(contentType)
	body := io.MultiReader(bytes.NewReader(head), r)

	_, err = s.client.PutObject(ctx, s.cfg.Bucket, key, body, size, minio.PutObjectOptions{
		ContentType: contentType,
		// Nothing is cached publicly; the presigned URL carries its own expiry.
		CacheControl: "private, no-store",
	})
	if err != nil {
		return Upload{}, fmt.Errorf("storage: put: %w", err)
	}
	return Upload{Key: key, ContentType: contentType, Bytes: size}, nil
}

// PresignedURL returns a short-lived read URL.
//
// Short-lived on purpose: a proof URL pasted into a chat should stop working.
func (s *Store) PresignedURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if strings.Contains(key, "..") || strings.HasPrefix(key, "/") {
		return "", fmt.Errorf("storage: refusing suspicious key %q", key)
	}
	if ttl <= 0 || ttl > time.Hour {
		ttl = 15 * time.Minute
	}
	u, err := s.client.PresignedGetObject(ctx, s.cfg.Bucket, key, ttl, url.Values{})
	if err != nil {
		return "", fmt.Errorf("storage: presign: %w", err)
	}
	if s.cfg.PublicEndpoint != "" {
		// The browser reaches MinIO on a different host from the service.
		if pub, err := url.Parse(s.cfg.PublicEndpoint); err == nil {
			u.Scheme, u.Host = pub.Scheme, pub.Host
		}
	}
	return u.String(), nil
}

// Remove deletes an object, for the UU PDP deletion flow.
func (s *Store) Remove(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.cfg.Bucket, key, minio.RemoveObjectOptions{})
}

func extensionFor(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "application/pdf":
		return ".pdf"
	default:
		return path.Ext("")
	}
}
