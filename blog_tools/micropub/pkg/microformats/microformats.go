package microformats

import (
	"encoding/json"
	"fmt"
	"io"
)

type Microformat struct {
	Type       []string         `json:"type"`
	Properties map[string][]any `json:"properties"`
}

type HugoPost struct {
	Date   string         `json:"date,omitempty"`
	Tags   []string       `json:"tags,omitempty"`
	Params HugoPostParams `json:"params,omitempty"`
}
type HugoPostParams struct {
	Uid      string           `json:"uid,omitempty"`
	Photo    string           `json:"photo,omitempty"`
	Location HugoPostLocation `json:"location,omitempty"`
	Caption  string           `json:"caption,omitempty"`
}
type HugoPostLocation struct {
	Locality string `json:"locality,omitempty"`
	Region   string `json:"region,omitempty"`
	Country  string `json:"country,omitempty"`
	Lat      string `json:"lat,omitempty"`
	Lon      string `json:"lon,omitempty"`
}

func (mf Microformat) GetFirstString(prop string) (string, error) {
	res, exists := mf.Properties[prop]
	if exists != true {
		return "", fmt.Errorf("mf key does not exist: %s", prop)
	}

	if len(res) == 0 {
		return "", fmt.Errorf("mf key is an empty array: %s", prop)
	}

	str, ok := res[0].(string)
	if ok != true {
		return "", fmt.Errorf("mf key is not a string: %s", prop)
	}

	return str, nil
}

func (mf Microformat) GetFirstMicroformat(prop string) (Microformat, error) {
	outMf := Microformat{
		Properties: map[string][]any{},
		Type:       []string{},
	}

	res, exists := mf.Properties[prop]
	if exists != true {
		return outMf, fmt.Errorf("mf key does not exist: %s", prop)
	}

	if len(res) == 0 {
		return outMf, fmt.Errorf("mf key is an empty array: %s", prop)
	}

	mfProperty, ok := res[0].(map[string]interface{})
	if ok != true {
		return outMf, fmt.Errorf("mf key is not a microformat: %s :: %#v", prop, mf)
	}

	// get mf type
	for _, mfType := range mfProperty["type"].([]interface{}) {
		outMf.Type = append(outMf.Type, mfType.(string))
	}
	// get mf Properties
	for mfPropKey, mfProps := range mfProperty["properties"].(map[string]interface{}) {
		outMf.Properties[mfPropKey] = mfProps.([]interface{})
	}

	return outMf, nil
}

func (mf Microformat) GetStringSlice(prop string) ([]string, error) {
	strSlice := []string{}

	res, exists := mf.Properties[prop]
	if exists != true {
		return strSlice, fmt.Errorf("mf key does not exist: %s", prop)
	}

	if len(res) == 0 {
		return strSlice, fmt.Errorf("mf key is an empty array: %s", prop)
	}

	for _, s := range res {
		str, ok := s.(string)
		if ok {
			strSlice = append(strSlice, str)
		}
	}

	return strSlice, nil
}

func (mf Microformat) GetHugoLocation(prop string) (HugoPostLocation, error) {
	hugoLocation := HugoPostLocation{}

	locationMf, err := mf.GetFirstMicroformat(prop)
	if err != nil {
		return hugoLocation, err
	}

	locality, err := locationMf.GetFirstString("locality")
	if err != nil {
		return hugoLocation, err
	}
	hugoLocation.Locality = locality

	region, err := locationMf.GetFirstString("region")
	if err != nil {
		return hugoLocation, err
	}
	hugoLocation.Region = region

	country, err := locationMf.GetFirstString("country-name")
	if err != nil {
		return hugoLocation, err
	}
	hugoLocation.Country = country

	geoMf, err := locationMf.GetFirstMicroformat("geo")
	if err != nil {
		return hugoLocation, err
	}

	lat, err := geoMf.GetFirstString("latitude")
	if err != nil {
		return hugoLocation, err
	}
	hugoLocation.Lat = lat

	lng, err := geoMf.GetFirstString("longitude")
	if err != nil {
		return hugoLocation, err
	}
	hugoLocation.Lon = lng

	return hugoLocation, nil
}

func (mf Microformat) ToHugoPost() (HugoPost, error) {
	hugo := HugoPost{}

	dat, err := mf.GetFirstString("published")
	if err != nil {
		return hugo, err
	}
	hugo.Date = dat

	tags, err := mf.GetStringSlice("category")
	if err != nil {
		return hugo, err
	}
	hugo.Tags = tags

	photo, err := mf.GetFirstString("photo")
	if err != nil {
		return hugo, err
	}
	hugo.Params.Photo = photo

	caption, err := mf.GetFirstString("content")
	if err != nil {
		return hugo, err
	}
	hugo.Params.Caption = caption

	location, err := mf.GetHugoLocation("location")
	if err != nil {
		return hugo, err
	}
	hugo.Params.Location = location

	uid, err := mf.GetFirstString("uid")
	if err != nil {
		return hugo, err
	}
	hugo.Params.Uid = uid

	return hugo, nil
}

func Parse(r io.ReadCloser) (HugoPost, error) {
	defer r.Close()
	mf, err := parse(r)
	if err != nil {
		return HugoPost{}, err
	}

	return mf.ToHugoPost()
}

func parse(r io.Reader) (Microformat, error) {
	mf := Microformat{}
	bytes, err := io.ReadAll(r)
	if err != nil {
		return mf, fmt.Errorf("failed to read bytes :: %v", err)
	}

	if err := json.Unmarshal(bytes, &mf); err != nil {
		return mf, fmt.Errorf("failed to unmarshal json :: %v", err)
	}
	return mf, err
}
