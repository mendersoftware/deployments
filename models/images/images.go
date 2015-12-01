package images

import (
	"errors"
	"time"

	"github.com/mendersoftware/services/models/users"
)

var (
	// Missing required attibute
	ErrMissingImageAttrName = errors.New("Required field missing: 'name'")
)

type ImagesModelI interface {
	Find(user users.UserI) ([]*ImageMeta, error)
	FindOne(user users.UserI, id string) (*ImageMeta, error)
	Exists(user users.UserI, id string) (bool, error)

	// ImageMeta.Name attribute is required to be unique
	Insert(user users.UserI, image *ImageMeta) (string, error)
	Update(user users.UserI, image *ImageMeta) error
	Delete(user users.UserI, id string) error
}

// Public - READ ONLY
type ImageMetaPrivate struct {

	// Unique field
	Id string `json:"id"`

	Verified    bool      `json:"verified"`
	LastUpdated time.Time `json:"modified"`
}

// Public - WRITTABLE (CREATE / EDIT)
type ImageMetaPublic struct {

	//Unique field
	Name string `json:"name"`

	Description string `json:"description"`
	Md5         string `json:"md5"`
	Model       string `json:"model"`
}

// NewImageMetaPublic create new struct. Name is required field.
func NewImageMetaPublic(name string) *ImageMetaPublic {
	return &ImageMetaPublic{
		Name: name,
	}
}

// Check if required fields are set.
// Can be improved with some reflection and tag magic ("required" tag)
func (i *ImageMetaPublic) Valid() error {

	if i.Name == "" {
		return ErrMissingImageAttrName
	}

	return nil
}

type ImageMeta struct {
	*ImageMetaPublic
	*ImageMetaPrivate
}

func NewImageMetaMerge(public *ImageMetaPublic, private *ImageMetaPrivate) *ImageMeta {
	img := &ImageMeta{
		ImageMetaPublic:  public,
		ImageMetaPrivate: private,
	}

	img.LastUpdated = time.Now()

	return img
}

func NewImageMetaFromPublic(public *ImageMetaPublic) *ImageMeta {

	return &ImageMeta{
		ImageMetaPublic: public,
		ImageMetaPrivate: &ImageMetaPrivate{
			LastUpdated: time.Now(),
		},
	}
}
