package main

import "testing"

func TestOutputFormat(t *testing.T) {
	tests := []struct {
		input    string
		mimeType string
		format   string
		wantErr  bool
	}{
		{input: "jpeg", mimeType: "image/jpeg", format: "jpeg"},
		{input: "jpg", mimeType: "image/jpeg", format: "jpeg"},
		{input: "png", mimeType: "image/png", format: "png"},
		{input: "webp", wantErr: true},
	}

	for _, test := range tests {
		mimeType, format, err := outputFormat(test.input)
		if test.wantErr {
			if err == nil {
				t.Errorf("outputFormat(%q) returned no error", test.input)
			}
			continue
		}
		if err != nil || mimeType != test.mimeType || format != test.format {
			t.Errorf("outputFormat(%q) = (%q, %q, %v), want (%q, %q, nil)", test.input, mimeType, format, err, test.mimeType, test.format)
		}
	}
}

func TestValidateOutputExtension(t *testing.T) {
	tests := []struct {
		path    string
		format  string
		wantErr bool
	}{
		{path: "image.jpg", format: "jpeg"},
		{path: "image.jpeg", format: "jpeg"},
		{path: "image.png", format: "png"},
		{path: "image.png", format: "jpeg", wantErr: true},
		{path: "image.jpg", format: "png", wantErr: true},
		{path: "image", format: "jpeg", wantErr: true},
	}

	for _, test := range tests {
		err := validateOutputExtension(test.path, test.format)
		if test.wantErr && err == nil {
			t.Errorf("validateOutputExtension(%q, %q) returned no error", test.path, test.format)
		}
		if !test.wantErr && err != nil {
			t.Errorf("validateOutputExtension(%q, %q) returned %v", test.path, test.format, err)
		}
	}
}

func TestGeneratedImage(t *testing.T) {
	convenience := &apiResponse{OutputImage: &apiImage{Type: "image", Data: "convenience"}}
	if image := convenience.generatedImage(); image == nil || image.Data != "convenience" {
		t.Fatal("expected the output_image convenience field to be used")
	}

	steps := &apiResponse{Steps: []apiStep{
		{Content: []apiImage{{Type: "text"}, {Type: "image", Data: "first"}}},
		{Content: []apiImage{{Type: "image", Data: "last"}}},
	}}
	if image := steps.generatedImage(); image == nil || image.Data != "last" {
		t.Fatal("expected the last image from the REST steps response")
	}

	if image := (&apiResponse{}).generatedImage(); image != nil {
		t.Fatal("expected no image for an empty response")
	}
}
