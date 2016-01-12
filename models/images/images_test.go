package images

import "testing"

func TestImageMetaPublicValid(t *testing.T) {

	testList := []struct {
		out   error
		image *ImageMetaPublic
	}{
		{ErrMissingImageAttrName, &ImageMetaPublic{}},
		{nil, NewImageMetaPublic("SOMETHING")},
	}

	for _, test := range testList {
		if test.out != test.image.Valid() {
			t.FailNow()
		}
	}
}
