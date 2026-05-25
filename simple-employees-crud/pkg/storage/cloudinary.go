// Package storage wraps the Cloudinary Go SDK to provide a simple API for
// uploading and removing employee avatar images.  In local development the
// upload can be disabled by leaving Cloudinary credentials empty, in which
// case the service falls back to storing a local URL placeholder.
package storage

import (
	"context"
	"fmt"
	"mime/multipart"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

// CloudinaryConfig holds the credentials required by the Cloudinary API.
type CloudinaryConfig struct {
	CloudName string
	APIKey    string
	APISecret string
	// Folder is the target folder inside the Cloudinary media library.
	Folder string
}

// CloudinaryClient wraps the official SDK client with application-specific helpers.
type CloudinaryClient struct {
	cld    *cloudinary.Cloudinary
	folder string
}

// NewCloudinaryClient initialises and validates a CloudinaryClient.
// Returns an error if the credentials are empty or if the SDK rejects them.
func NewCloudinaryClient(cfg CloudinaryConfig) (*CloudinaryClient, error) {
	if cfg.CloudName == "" || cfg.APIKey == "" || cfg.APISecret == "" {
		return nil, fmt.Errorf("storage: cloudinary credentials are incomplete")
	}

	cld, err := cloudinary.NewFromParams(cfg.CloudName, cfg.APIKey, cfg.APISecret)
	if err != nil {
		return nil, fmt.Errorf("storage: failed to create cloudinary client: %w", err)
	}

	return &CloudinaryClient{cld: cld, folder: cfg.Folder}, nil
}

// UploadResult is the minimal subset of the Cloudinary upload response that
// callers need to persist.
type UploadResult struct {
	// URL is the HTTPS CDN URL of the uploaded image.
	URL string
	// PublicID is the unique Cloudinary asset identifier needed to delete the image.
	PublicID string
}

// UploadAvatar uploads a multipart file to Cloudinary and returns the CDN URL
// and the public ID required for future deletion.
// The image is stored under c.folder/<employeeID>.
func (c *CloudinaryClient) UploadAvatar(
	ctx context.Context,
	file multipart.File,
	employeeID string,
) (*UploadResult, error) {
	resp, err := c.cld.Upload.Upload(ctx, file, uploader.UploadParams{
		PublicID:       c.folder + "/" + employeeID,
		Folder:         c.folder,
		Overwrite:      boolPtr(true),
		UniqueFilename: boolPtr(false),
		// Resize to a square thumbnail on Cloudinary to save bandwidth.
		Transformation: "c_fill,h_256,w_256,q_auto,f_auto",
		ResourceType:   "image",
	})
	if err != nil {
		return nil, fmt.Errorf("storage: cloudinary upload failed: %w", err)
	}

	return &UploadResult{
		URL:      resp.SecureURL,
		PublicID: resp.PublicID,
	}, nil
}

// DeleteAvatar removes an avatar from Cloudinary by its public ID.
// This is called when an employee is deleted or their avatar is replaced.
func (c *CloudinaryClient) DeleteAvatar(ctx context.Context, publicID string) error {
	_, err := c.cld.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID:     publicID,
		ResourceType: "image",
	})
	if err != nil {
		return fmt.Errorf("storage: cloudinary delete failed: %w", err)
	}
	return nil
}

func boolPtr(b bool) *bool { return &b }
