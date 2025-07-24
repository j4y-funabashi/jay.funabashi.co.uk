package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	sm "github.com/flopp/go-staticmaps"
	"github.com/fogleman/gg"
	"github.com/golang/geo/s2"
	"github.com/tkrajina/gpxgo/gpx"
)

type HikeData struct {
	StartTime        time.Time
	EndTime          time.Time
	MovingTime       float64
	DistanceInMeters float64
	TimeTaken        time.Duration
	Ascent           float64
	Descent          float64
	StartLocation    Location
	EndLocation      Location
}

type Location struct {
	Locality  string
	Region    string
	Country   string
	Latitude  float64
	Longitude float64
}

func (l Location) String() string {
	return fmt.Sprintf("%s, %s", l.Locality, l.Region)
}

func (h HikeData) String() string {
	return fmt.Sprintf("from %s to %s. %.2f km, %.2fm ascent, %.2fm decent in %s", h.StartLocation, h.EndLocation, h.DistanceInMeters/1000, h.Ascent, h.Descent, h.TimeTaken)
}

func (h HikeData) MapFileName() string {
	return fmt.Sprintf("%s_map.png", h.StartTime.Format(time.RFC3339))
}

func main() {
	inputDir := "/home/jayr/Downloads/garmin-connect-export/2025-05-14_garmin_connect_export"

	err := filepath.Walk(inputDir,
		func(path string, info os.FileInfo, err error) error {

			ext := filepath.Ext(path)
			if ext != ".gpx" {
				return nil
			}
			if path != "/home/jayr/Downloads/garmin-connect-export/2025-05-14_garmin_connect_export/activity_19092581469.gpx" {
				return nil
			}
			hikeData := ProcessFile(path)
			log.Printf("=== hike: %s", hikeData)
			return nil
		})

	if err != nil {

		log.Fatalf("failed walking dir %s", err.Error())
	}

}

func ProcessFile(inputFilePath string) HikeData {
	log.Printf("processing file: %s", inputFilePath)

	inputFile, err := os.ReadFile(inputFilePath)
	if err != nil {
		log.Fatalf("failed opening file")
	}

	gpxFile, err := gpx.ParseBytes(inputFile)
	if err != nil {
		log.Fatalf("failed parsing bytes")
	}
	gpxFileData := ParseGpxFile(gpxFile)

	hike := HikeData{
		StartTime:        gpxFile.TimeBounds().StartTime,
		EndTime:          gpxFile.TimeBounds().EndTime,
		TimeTaken:        gpxFile.TimeBounds().EndTime.Sub(gpxFile.TimeBounds().StartTime),
		MovingTime:       gpxFile.MovingData().MovingTime,
		DistanceInMeters: gpxFile.Length2D(),
		Ascent:           gpxFile.UphillDownhill().Uphill,
		Descent:          gpxFile.UphillDownhill().Downhill,
	}

	// create static map
	ctx := sm.NewContext()
	ctx.SetSize(800, 600)

	ctx.AddObject(
		sm.NewPath(gpxFileData.Path, color.RGBA{255, 0, 0, 255}, 4.0),
	)

	ctx.AddObject(
		sm.NewMarker(
			s2.LatLngFromDegrees(gpxFileData.FirstPoint.Latitude, gpxFileData.FirstPoint.Longitude),
			color.RGBA{0, 255, 0, 255},
			16.0,
		))

	ctx.AddObject(
		sm.NewMarker(
			s2.LatLngFromDegrees(gpxFileData.LastPoint.Latitude, gpxFileData.LastPoint.Longitude),
			color.RGBA{255, 0, 0, 255},
			16.0,
		),
	)

	img, err := ctx.Render()
	if err != nil {
		log.Fatalf("failed to render image")
	}

	err = gg.SavePNG(hike.MapFileName(), img)
	if err != nil {
		log.Fatalf("failed to save image")
	}

	hike.StartLocation = ReverseGeo(gpxFileData.FirstPoint)
	hike.EndLocation = ReverseGeo(gpxFileData.LastPoint)

	return hike
}

type GpxFileData struct {
	Path       []s2.LatLng
	FirstPoint gpx.GPXPoint
	LastPoint  gpx.GPXPoint
}

func ParseGpxFile(gpxFile *gpx.GPX) GpxFileData {
	response := GpxFileData{}

	for _, track := range gpxFile.Tracks {
		for _, seg := range track.Segments {
			response.FirstPoint = seg.Points[0]
			response.LastPoint = seg.Points[len(seg.Points)-1]
			for _, point := range seg.Points {
				response.Path = append(response.Path, s2.LatLngFromDegrees(point.Latitude, point.Longitude))
			}
		}
	}

	return response
}

func ReverseGeo(point gpx.GPXPoint) Location {

	// build URL
	baseURL := "https://nominatim.openstreetmap.org/reverse"
	requestURL, err := url.Parse(baseURL)
	if err != nil {
		log.Fatalf("failed to parse url")
	}
	q := requestURL.Query()
	q.Set("format", "geocodejson")
	q.Set("zoom", "18")
	q.Set("addressdetails", "1")
	lat := strconv.FormatFloat(point.Latitude, 'f', -1, 64)
	lon := strconv.FormatFloat(point.Longitude, 'f', -1, 64)
	q.Set("lat", lat)
	q.Set("lon", lon)
	requestURL.RawQuery = q.Encode()

	log.Printf("lat=%s&lon=%s", lat, lon)

	res, err := http.Get(requestURL.String())
	if err != nil {
		log.Fatalf("failed to get url")
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("failed to read body")
	}

	response := ReverseGeocodeResponse{}
	json.Unmarshal(body, &response)

	locality := response.GetLocality()
	if locality == "" {
		log.Fatalf("failed to get locality :: %+v", response)
	}
	region := response.GetRegion()
	if region == "" {
		log.Fatalf("failed to get region :: %+v", response)
	}

	return Location{
		Latitude:  point.Latitude,
		Longitude: point.Longitude,
		Locality:  locality,
		Region:    region,
		Country:   response.Features[0].Properties.Geocoding.Country,
	}
}
func (res ReverseGeocodeResponse) GetRegion() string {

	fields := []string{
		res.Features[0].Properties.Geocoding.County,
		res.Features[0].Properties.Geocoding.State,
	}

	for _, loc := range fields {
		if loc != "" {
			return loc
		}
	}
	return ""
}

func (res ReverseGeocodeResponse) GetLocality() string {
	fields := []string{
		res.Features[0].Properties.Geocoding.Locality,
		res.Features[0].Properties.Geocoding.Admin.Level10,
		res.Features[0].Properties.Geocoding.Admin.Level8,
		res.Features[0].Properties.Geocoding.Admin.Level5,
		res.Features[0].Properties.Geocoding.City,
	}

	for _, loc := range fields {
		if loc != "" {
			return loc
		}
	}
	return ""
}

type ReverseGeocodeResponse struct {
	Features []Feature `json:"features,omitempty"`
}

type Feature struct {
	Properties Properties `json:"properties"`
}

type Properties struct {
	Geocoding Geocoding `json:"geocoding"`
}

type Geocoding struct {
	Admin    AdminLevels `json:"admin,omitempty"`
	Locality string      `json:"locality,omitempty"`
	County   string      `json:"county,omitempty"`
	Country  string      `json:"country,omitempty"`
	State    string      `json:"state,omitempty"`
	City     string      `json:"city,omitempty"`
}

type AdminLevels struct {
	Level10 string `json:"level10,omitempty"`
	Level8  string `json:"level8,omitempty"`
	Level6  string `json:"level6,omitempty"`
	Level5  string `json:"level5,omitempty"`
	Level4  string `json:"level4,omitempty"`
}
